package cli_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

// --- create: setting labels and priority ---

// AC: `create --label --priority` sets the fields. They land in the JSON result and
// on disk, and unset ones stay off the file entirely while showing as null/[] in
// JSON (the no-missing-keys contract, ADR 0013).
func TestCreateSetsLabelsAndPriority(t *testing.T) {
	h := beavertest.New(t).Init()

	out := h.DecodeJSON(h.MustRun("create", "Fix login crash", "--priority", "urgent", "--label", "bug", "--label", "security").Stdout)
	if out["priority"] != "urgent" {
		t.Errorf("created priority = %v, want urgent", out["priority"])
	}
	if got := strSlice(out["labels"]); !slices.Equal(got, []string{"bug", "security"}) {
		t.Errorf("created labels = %v, want [bug security]", got)
	}

	// The values are on disk, not just echoed.
	file := readIssueFile(t, h, out["id"].(string), "Fix login crash")
	for _, want := range []string{"priority: urgent", "labels:", "- bug", "- security"} {
		if !strings.Contains(file, want) {
			t.Errorf("issue file missing %q:\n%s", want, file)
		}
	}
}

// An issue created with neither flag omits both keys from the file (omitempty), yet
// still reports priority: null and labels: [] in JSON.
func TestCreateOmitsUnsetClassification(t *testing.T) {
	h := beavertest.New(t).Init()

	out := h.DecodeJSON(h.MustRun("create", "Plain issue").Stdout)
	if out["priority"] != nil {
		t.Errorf("unset priority = %v, want null", out["priority"])
	}
	if got := strSlice(out["labels"]); len(got) != 0 {
		t.Errorf("unset labels = %v, want []", got)
	}

	file := readIssueFile(t, h, out["id"].(string), "Plain issue")
	for _, absent := range []string{"priority:", "labels:"} {
		if strings.Contains(file, absent) {
			t.Errorf("file should omit unset %q:\n%s", absent, file)
		}
	}
}

// A repeated or comma-joined --label collapses to one label per distinct value, in
// first-seen order — the same normalization edges get.
func TestCreateDedupesAndSplitsLabels(t *testing.T) {
	h := beavertest.New(t).Init()

	out := h.DecodeJSON(h.MustRun("create", "Tag soup", "--label", "bug,ux", "--label", "bug").Stdout)
	if got := strSlice(out["labels"]); !slices.Equal(got, []string{"bug", "ux"}) {
		t.Errorf("labels = %v, want deduped, split [bug ux]", got)
	}
}

// An invalid --priority is a usage error (exit 2) caught before any store work, so
// no file is written.
func TestCreateRejectsInvalidPriority(t *testing.T) {
	h := beavertest.New(t).Init()

	r := h.Run("create", "Bad", "--priority", "huge")
	if r.Code != 2 {
		t.Errorf("create --priority huge exit = %d, want 2 (usage)", r.Code)
	}
	if !strings.Contains(r.Stderr, "priority") {
		t.Errorf("error should name the bad priority:\n%s", r.Stderr)
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("invalid create wrote %v, want no file", files)
	}
}

// --- priority command: mutate after creation ---

// AC: priority is mutable after creation. `priority <ref> <level>` sets it and
// bumps `updated`; the change persists; state is untouched.
func TestPrioritySetsAndChanges(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Some work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	h.Clock.Advance(time.Hour)

	out := h.DecodeJSON(h.MustRun("priority", "iss001", "high").Stdout)
	if out["priority"] != "high" {
		t.Errorf("priority = %v, want high", out["priority"])
	}
	if out["state"] != "todo" {
		t.Errorf("state = %v, want unchanged todo", out["state"])
	}
	if out["updated"] == "2026-06-27T18:30:00Z" {
		t.Errorf("updated = %v, want bumped by the write", out["updated"])
	}

	// A second call changes it, and the new value persists.
	h.MustRun("priority", "iss001", "low")
	if got := showJSON(t, h, "iss001")["priority"]; got != "low" {
		t.Errorf("persisted priority = %v, want low", got)
	}
}

// `priority <ref> none` clears a set priority back to unprioritized (null in JSON,
// absent from the file).
func TestPriorityClears(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Some work", issue.StateTodo, issue.PriorityHigh, nil, "", beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("priority", "iss001", "none").Stdout)
	if out["priority"] != nil {
		t.Errorf("priority after clear = %v, want null", out["priority"])
	}
	if file := readIssueFile(t, h, "iss001", "Some work"); strings.Contains(file, "priority:") {
		t.Errorf("cleared priority should be absent from the file:\n%s", file)
	}
}

// Setting the priority an issue already has is an idempotent no-op: success, but no
// rewrite, so `updated` is not churned and the file bytes are unchanged.
func TestPriorityIdempotentNoop(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, issue.PriorityMedium, nil, "", beavertest.DefaultNow)
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour) // a stray write would bump updated

	r := h.Run("priority", "iss001", "medium")
	if r.Code != 0 {
		t.Errorf("re-setting same priority exit = %d, want 0", r.Code)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("no-op priority rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// An invalid level, or the wrong number of arguments, is a usage error (exit 2).
func TestPriorityUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	cases := [][]string{
		{"priority", "iss001", "bogus"},
		{"priority", "iss001"},
		{"priority"},
		{"priority", "iss001", "high", "extra"},
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

// --- label command: mutate after creation ---

// AC: labels are mutable after creation. Positional labels are added; --remove drops
// them. Labels are a set, existing ones are kept, and the change persists with a
// bumped `updated`.
func TestLabelAddsAndRemoves(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", []string{"bug"}, "", beavertest.DefaultNow)

	// Add two (one already present → set semantics keep a single copy).
	out := h.DecodeJSON(h.MustRun("label", "iss001", "bug", "ux").Stdout)
	if got := strSlice(out["labels"]); !slices.Equal(got, []string{"bug", "ux"}) {
		t.Errorf("after add = %v, want [bug ux]", got)
	}

	// Remove one; the rest persists to disk.
	h.MustRun("label", "iss001", "--remove", "bug")
	if got := strSlice(showJSON(t, h, "iss001")["labels"]); !slices.Equal(got, []string{"ux"}) {
		t.Errorf("after remove = %v, want [ux]", got)
	}
}

// Adds and removes may be combined in one call, and a label named in both is removed
// (removal wins), so the outcome is independent of ordering.
func TestLabelAddAndRemoveInOneCall(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", []string{"old"}, "", beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("label", "iss001", "new", "dup", "--remove", "old", "--remove", "dup").Stdout)
	if got := strSlice(out["labels"]); !slices.Equal(got, []string{"new"}) {
		t.Errorf("labels = %v, want [new] (old and dup removed, dup's removal winning its add)", got)
	}
}

// An invocation whose net effect changes nothing is an idempotent no-op: no rewrite,
// so `updated` and the file bytes are untouched.
func TestLabelNoopWhenUnchanged(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", []string{"bug"}, "", beavertest.DefaultNow)
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour)

	// Add a label already present and remove one that is absent → nothing changes.
	r := h.Run("label", "iss001", "bug", "--remove", "ghost")
	if r.Code != 0 {
		t.Errorf("no-op label exit = %d, want 0", r.Code)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("no-op label rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// label with no add and no remove, or no reference at all, is a usage error (exit 2).
func TestLabelUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	cases := [][]string{
		{"label", "iss001"}, // nothing to add or remove
		{"label"},           // no reference
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

// --- list: sorting and filtering ---

// AC: list sorts by priority — urgent first, then high, medium, low, then the
// unprioritized — and within one priority falls back to the stable creation order.
// The fixture misaligns priority from creation time so both rules are exercised at
// once: bbb (urgent, newest) leads, the two highs order by creation (ccc before
// aaa), and ddd (no priority, oldest) sorts last, not first.
func TestListSortsByPriority(t *testing.T) {
	h := beavertest.New(t).Init()
	base := beavertest.DefaultNow
	seedClassified(t, h, "ddd000", "None oldest", issue.StateTodo, "", nil, "", base)
	seedClassified(t, h, "ccc000", "High earlier", issue.StateTodo, issue.PriorityHigh, nil, "", base.Add(1*time.Minute))
	seedClassified(t, h, "aaa000", "High later", issue.StateTodo, issue.PriorityHigh, nil, "", base.Add(2*time.Minute))
	seedClassified(t, h, "bbb000", "Urgent newest", issue.StateTodo, issue.PriorityUrgent, nil, "", base.Add(3*time.Minute))
	seedClassified(t, h, "eee000", "Low", issue.StateTodo, issue.PriorityLow, nil, "", base.Add(4*time.Minute))

	got := listIDs(t, h, "--state", "all")
	want := []string{"bbb000", "ccc000", "aaa000", "eee000", "ddd000"}
	if !slices.Equal(got, want) {
		t.Errorf("priority sort = %v, want %v", got, want)
	}
}

// AC: list filters by priority. --priority <level> keeps only that level; --priority
// none keeps only the unprioritized.
func TestListFiltersByPriority(t *testing.T) {
	h := beavertest.New(t).Init()
	base := beavertest.DefaultNow
	seedClassified(t, h, "urg000", "Urgent", issue.StateTodo, issue.PriorityUrgent, nil, "", base)
	seedClassified(t, h, "hih000", "High", issue.StateTodo, issue.PriorityHigh, nil, "", base.Add(time.Minute))
	seedClassified(t, h, "non000", "None", issue.StateTodo, "", nil, "", base.Add(2*time.Minute))

	if got := listIDs(t, h, "--priority", "urgent"); !slices.Equal(got, []string{"urg000"}) {
		t.Errorf("--priority urgent = %v, want [urg000]", got)
	}
	if got := listIDs(t, h, "--priority", "none"); !slices.Equal(got, []string{"non000"}) {
		t.Errorf("--priority none = %v, want [non000] (the unprioritized)", got)
	}
}

// AC: list filters by label, with AND semantics — repeating --label narrows to the
// issues carrying every named label.
func TestListFiltersByLabelAND(t *testing.T) {
	h := beavertest.New(t).Init()
	base := beavertest.DefaultNow
	seedClassified(t, h, "both00", "Bug and v1", issue.StateTodo, "", []string{"bug", "v1"}, "", base)
	seedClassified(t, h, "bug000", "Bug only", issue.StateTodo, "", []string{"bug"}, "", base.Add(time.Minute))
	seedClassified(t, h, "v1only", "V1 only", issue.StateTodo, "", []string{"v1"}, "", base.Add(2*time.Minute))

	if got := listIDs(t, h, "--label", "bug"); !slices.Equal(got, []string{"both00", "bug000"}) {
		t.Errorf("--label bug = %v, want the two bug issues", got)
	}
	if got := listIDs(t, h, "--label", "bug", "--label", "v1"); !slices.Equal(got, []string{"both00"}) {
		t.Errorf("--label bug --label v1 = %v, want only the issue with both", got)
	}
}

// AC: list filters by assignee — exact match on the assignee field.
func TestListFiltersByAssignee(t *testing.T) {
	h := beavertest.New(t).Init()
	base := beavertest.DefaultNow
	seedClassified(t, h, "als000", "Alice work", issue.StateTodo, "", nil, "alice", base)
	seedClassified(t, h, "bob000", "Bob work", issue.StateTodo, "", nil, "bob", base.Add(time.Minute))

	if got := listIDs(t, h, "--assignee", "alice"); !slices.Equal(got, []string{"als000"}) {
		t.Errorf("--assignee alice = %v, want [als000]", got)
	}
}

// The attribute filters combine with each other (AND) and refine a base selector
// rather than replacing it, so `--priority high --label bug --state todo` is the
// intersection of all three.
func TestListFiltersCombineAndRefineSelectors(t *testing.T) {
	h := beavertest.New(t).Init()
	base := beavertest.DefaultNow
	seedClassified(t, h, "hit000", "Match", issue.StateTodo, issue.PriorityHigh, []string{"bug"}, "alice", base)
	seedClassified(t, h, "wrgpri", "Wrong priority", issue.StateTodo, issue.PriorityLow, []string{"bug"}, "alice", base.Add(time.Minute))
	seedClassified(t, h, "wrglab", "Wrong label", issue.StateTodo, issue.PriorityHigh, []string{"ux"}, "alice", base.Add(2*time.Minute))
	seedClassified(t, h, "wrgsta", "Wrong state", issue.StateDone, issue.PriorityHigh, []string{"bug"}, "alice", base.Add(3*time.Minute))

	got := listIDs(t, h, "--priority", "high", "--label", "bug", "--assignee", "alice", "--state", "todo")
	if !slices.Equal(got, []string{"hit000"}) {
		t.Errorf("combined filters = %v, want only [hit000]", got)
	}
}

// An invalid --priority filter value is a usage error (exit 2), like the setter.
func TestListRejectsInvalidPriorityFilter(t *testing.T) {
	h := beavertest.New(t).Init()
	if r := h.Run("list", "--priority", "huge"); r.Code != 2 {
		t.Errorf("list --priority huge exit = %d, want 2 (usage)", r.Code)
	}
}

// --- show ---

// AC: show exposes labels and priority. The human view lists both fields; the JSON
// view carries them under their keys.
func TestShowExposesPriorityAndLabels(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Visible fields", issue.StateTodo, issue.PriorityHigh, []string{"bug", "v1"}, "", beavertest.DefaultNow)

	h.IsTTY = true
	human := h.MustRun("show", "iss001").Stdout
	for _, want := range []string{"priority", "high", "labels", "bug", "v1"} {
		if !strings.Contains(human, want) {
			t.Errorf("human show missing %q:\n%s", want, human)
		}
	}

	h.IsTTY = false
	shown := showJSON(t, h, "iss001")
	if shown["priority"] != "high" {
		t.Errorf("JSON priority = %v, want high", shown["priority"])
	}
	if got := strSlice(shown["labels"]); !slices.Equal(got, []string{"bug", "v1"}) {
		t.Errorf("JSON labels = %v, want [bug v1]", got)
	}
}

// --- helpers ---

// seedClassified writes an issue file directly with priority, labels, and assignee
// set, so filtering and sorting tests can shape a store no single command sequence
// produces and pin creation times exactly.
func seedClassified(t *testing.T, h *beavertest.Harness, id, title string, state issue.State, priority issue.Priority, labels []string, assignee string, created time.Time) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: state, Priority: priority, Labels: labels,
		Assignee: assignee, Created: created, Updated: created,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+issue.FileName(id, issue.Slug(title)), string(data))
}
