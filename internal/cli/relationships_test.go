package cli_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// Edges live on the dependent alone; the inverse (blocks/children) is derived on
// read, never written to the referenced files.
func TestCreateStoresEdgesOneSided(t *testing.T) {
	h := beavertest.New(t).Init()
	base := h.DecodeJSON(h.MustRun("create", "Foundational work").Stdout)["id"].(string)
	epic := h.DecodeJSON(h.MustRun("create", "The epic").Stdout)["id"].(string)

	out := h.DecodeJSON(h.MustRun("create", "Dependent work", "--depends-on", base, "--parent", epic).Stdout)
	child := out["id"].(string)
	if got := strSlice(out["depends_on"]); !slices.Equal(got, []string{base}) {
		t.Errorf("created depends_on = %v, want [%s]", got, base)
	}
	if out["parent"] != epic {
		t.Errorf("created parent = %v, want %s", out["parent"], epic)
	}

	shownBase := showJSON(t, h, base)
	if got := strSlice(shownBase["depends_on"]); len(got) != 0 {
		t.Errorf("target depends_on = %v, want [] (edges are one-sided)", got)
	}
	if got := strSlice(rels(t, shownBase)["blocks"]); !slices.Equal(got, []string{child}) {
		t.Errorf("target derived blocks = %v, want [%s]", got, child)
	}
	shownEpic := showJSON(t, h, epic)
	if got := strSlice(rels(t, shownEpic)["children"]); !slices.Equal(got, []string{child}) {
		t.Errorf("epic derived children = %v, want [%s]", got, child)
	}
	if baseFile := readIssueFile(t, h, base, "Foundational work"); strings.Contains(baseFile, child) {
		t.Errorf("target file names the dependent %q — the inverse must not be stored:\n%s", child, baseFile)
	}
}

// An edge accepts any reference the resolver takes but stores the canonical id, and
// two references to one issue collapse to one edge.
func TestCreateResolvesAndDedupesEdges(t *testing.T) {
	h := beavertest.New(t).Init()
	base := h.DecodeJSON(h.MustRun("create", "Shared base").Stdout)["id"].(string)

	// Reference the same issue three ways in one create: by slug, then twice by id.
	out := h.DecodeJSON(h.MustRun("create", "Uses the base",
		"--depends-on", "shared-base", // the slug
		"--depends-on", base+","+base, // the id, twice, comma-joined
	).Stdout)
	if got := strSlice(out["depends_on"]); !slices.Equal(got, []string{base}) {
		t.Errorf("depends_on = %v, want a single resolved, deduped [%s]", got, base)
	}
}

// A typo must not persist as a dangling edge, so create writes no file.
func TestCreateRejectsUnknownEdge(t *testing.T) {
	for _, flag := range []string{"--depends-on", "--parent"} {
		h := beavertest.New(t).Init()
		r := h.Run("create", "New issue", flag, "nope99")
		if r.Code != 3 {
			t.Errorf("create %s nope99 exit = %d, want 3 (not-found)", flag, r.Code)
		}
		if files := h.IssueFiles(); len(files) != 0 {
			t.Errorf("create with an unresolvable %s wrote %v, want no file", flag, files)
		}
	}
}

func TestListReadyExcludesBlocked(t *testing.T) {
	h := beavertest.New(t).Init()
	seedGraph(t, h)

	got := listIDs(t, h, "--ready")
	slices.Sort(got)
	if want := []string{"rdy100", "rdy200"}; !slices.Equal(got, want) {
		t.Errorf("list --ready = %v, want %v (blocked, stuck, in-progress, and done all excluded)", got, want)
	}
}

// --blocked is only unstarted todo work: an in-progress or closed issue stays out
// even when an edge is unmet.
func TestListBlockedQueue(t *testing.T) {
	h := beavertest.New(t).Init()
	seedGraph(t, h)

	got := listIDs(t, h, "--blocked")
	slices.Sort(got)
	if want := []string{"blk100", "stk100"}; !slices.Equal(got, want) {
		t.Errorf("list --blocked = %v, want %v (todo + unmet; in-progress ipb100 and done dnb100 excluded)", got, want)
	}
}

func TestReadyAndBlockedPartitionTodo(t *testing.T) {
	h := beavertest.New(t).Init()
	seedGraph(t, h)

	ready := listIDs(t, h, "--ready")
	blocked := listIDs(t, h, "--blocked")
	for _, id := range ready {
		if slices.Contains(blocked, id) {
			t.Errorf("%s is in both --ready and --blocked; the queues must be disjoint", id)
		}
	}
	together := append(append([]string{}, ready...), blocked...)
	slices.Sort(together)
	if want := []string{"blk100", "rdy100", "rdy200", "stk100"}; !slices.Equal(together, want) {
		t.Errorf("--ready ∪ --blocked = %v, want exactly the todo issues %v", together, want)
	}
}

// A cancelled dependency can never be met: the dependent is surfaced as stuck
// rather than silently freed.
func TestCancelledDependencyIsStuckNotReady(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "cncl00", "Abandoned base", issue.StateCancelled, nil, "")
	seedDep(t, h, "wait00", "Waits on the abandoned base", issue.StateTodo, []string{"cncl00"}, "")

	if got := listIDs(t, h, "--ready"); slices.Contains(got, "wait00") {
		t.Errorf("list --ready = %v, must exclude the stuck dependent wait00", got)
	}
	if got := listIDs(t, h, "--blocked"); !slices.Contains(got, "wait00") {
		t.Errorf("list --blocked = %v, must include the stuck dependent wait00", got)
	}

	r := rels(t, showJSON(t, h, "wait00"))
	if r["ready"] != false || r["blocked"] != true || r["stuck"] != true {
		t.Errorf("show wait00 relationships = %v, want ready=false blocked=true stuck=true", r)
	}
	blk := r["blocked_on"].([]any)
	if len(blk) != 1 {
		t.Fatalf("blocked_on = %v, want the one cancelled dependency", blk)
	}
	if b := blk[0].(map[string]any); b["id"] != "cncl00" || b["state"] != "cancelled" || b["missing"] != false {
		t.Errorf("blocked_on[0] = %v, want {cncl00, cancelled, missing=false}", b)
	}

	h.IsTTY = true
	if out := h.MustRun("show", "wait00").Stdout; !strings.Contains(out, "stuck") {
		t.Errorf("human show of a stuck issue should say 'stuck':\n%s", out)
	}
}

// Only done satisfies a dependency.
func TestReadyClearsOnlyWhenDependencyDone(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "prog00", "The prerequisite", issue.StateInProgress, nil, "")
	seedDep(t, h, "wait00", "Waits on the prerequisite", issue.StateTodo, []string{"prog00"}, "")

	if got := listIDs(t, h, "--ready"); slices.Contains(got, "wait00") {
		t.Fatalf("wait00 is ready while its dependency is in-progress: %v", got)
	}
	h.MustRun("done", "prog00") // the only thing that satisfies the edge

	got := listIDs(t, h, "--ready")
	if !slices.Contains(got, "wait00") {
		t.Errorf("list --ready = %v, want wait00 now that its dependency is done", got)
	}
	if slices.Contains(got, "prog00") {
		t.Errorf("list --ready = %v, the now-done dependency must not appear", got)
	}
}

// A dangling dependency degrades gracefully: the dependent is blocked and the
// blocker marked missing, rather than an error.
func TestMissingDependencyBlocksGracefully(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "wait00", "Waits on a ghost", issue.StateTodo, []string{"gone00"}, "")

	if got := listIDs(t, h, "--ready"); slices.Contains(got, "wait00") {
		t.Errorf("an issue with a missing dependency must not be ready: %v", got)
	}
	if got := listIDs(t, h, "--blocked"); !slices.Contains(got, "wait00") {
		t.Errorf("an issue with a missing dependency must be blocked: %v", got)
	}
	blk := rels(t, showJSON(t, h, "wait00"))["blocked_on"].([]any)
	if len(blk) != 1 {
		t.Fatalf("blocked_on = %v, want the one dangling dependency", blk)
	}
	if b := blk[0].(map[string]any); b["id"] != "gone00" || b["missing"] != true || b["state"] != nil {
		t.Errorf("blocked_on[0] = %v, want {gone00, missing=true, state=null}", b)
	}
}

func TestShowReportsWaitingOn(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seedDep(t, h, "base00", "The prerequisite", issue.StateTodo, nil, "")
	seedDep(t, h, "wait00", "The dependent", issue.StateTodo, []string{"base00"}, "")

	out := h.MustRun("show", "wait00").Stdout
	for _, want := range []string{"blocked", "waiting on", "base00", "todo"} {
		if !strings.Contains(out, want) {
			t.Errorf("show of a blocked issue missing %q:\n%s", want, out)
		}
	}

	// A standalone closed issue has nothing derived to report.
	seedDep(t, h, "lone00", "Nothing to relate", issue.StateDone, nil, "")
	quiet := h.MustRun("show", "lone00").Stdout
	for _, absent := range []string{"waiting on", "blocks", "children", "status"} {
		if strings.Contains(quiet, absent) {
			t.Errorf("show of an unrelated issue should not print %q:\n%s", absent, quiet)
		}
	}
}

// --ready and --blocked are mutually exclusive, and neither stacks with --state.
func TestListQueueUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	for _, args := range [][]string{
		{"list", "--ready", "--blocked"},
		{"list", "--ready", "--state", "todo"},
		{"list", "--blocked", "--state", "done"},
	} {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

// --- helpers ---

// seedGraph writes a small dependency graph exercising every readiness outcome,
// all at the default time so display order is the id order.
func seedGraph(t *testing.T, h *beavertest.Harness) {
	t.Helper()
	seedDep(t, h, "done00", "Satisfied prerequisite", issue.StateDone, nil, "")
	seedDep(t, h, "prog00", "Active prerequisite", issue.StateInProgress, nil, "")
	seedDep(t, h, "cncl00", "Abandoned prerequisite", issue.StateCancelled, nil, "")
	seedDep(t, h, "rdy100", "Ready, no deps", issue.StateTodo, nil, "")
	seedDep(t, h, "rdy200", "Ready, dep done", issue.StateTodo, []string{"done00"}, "")
	seedDep(t, h, "blk100", "Blocked by active dep", issue.StateTodo, []string{"prog00"}, "")
	seedDep(t, h, "stk100", "Stuck on cancelled dep", issue.StateTodo, []string{"cncl00"}, "")
	seedDep(t, h, "ipb100", "In progress, dep not done", issue.StateInProgress, []string{"prog00"}, "")
	seedDep(t, h, "dnb100", "Done despite an unmet dep", issue.StateDone, []string{"prog00"}, "")
}

// seedDep writes an issue file directly with dependency and parent edges, so tests
// can shape a graph no create sequence can (arbitrary states, dangling refs).
func seedDep(t *testing.T, h *beavertest.Harness, id, title string, state issue.State, deps []string, parent string) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: state, DependsOn: deps, Parent: parent,
		Created: beavertest.DefaultNow, Updated: beavertest.DefaultNow,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+issue.FileName(id, issue.Slug(title)), string(data))
}

// showJSON runs `beaver show <ref>` (JSON, since the harness is non-interactive)
// and decodes the object, including its derived relationships.
func showJSON(t *testing.T, h *beavertest.Harness, ref string) map[string]any {
	t.Helper()
	return h.DecodeJSON(h.MustRun("show", ref).Stdout)
}

// rels extracts the derived relationships object from a decoded show result.
func rels(t *testing.T, shown map[string]any) map[string]any {
	t.Helper()
	r, ok := shown["relationships"].(map[string]any)
	if !ok {
		t.Fatalf("show output has no relationships object: %v", shown)
	}
	return r
}

// readIssueFile reads an issue's on-disk file by its canonical name.
func readIssueFile(t *testing.T, h *beavertest.Harness, id, title string) string {
	t.Helper()
	return h.ReadFile("issues/" + issue.FileName(id, issue.Slug(title)))
}

// strSlice coerces a decoded JSON array into a []string.
func strSlice(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
