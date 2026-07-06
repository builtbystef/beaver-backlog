package cli_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/busy-beaver/internal/beavertest"
	"github.com/builtbystef/busy-beaver/internal/issue"
)

func TestListDefaultShowsAll(t *testing.T) {
	h := beavertest.New(t).Init()
	s := seedAllStates(t, h)

	got := listIDs(t, h)
	if want := []string{s.done, s.todo, s.cancel, s.prog}; !slices.Equal(got, want) {
		t.Errorf("default list = %v, want all issues %v", got, want)
	}
	if all := listIDs(t, h, "--state", "all"); !slices.Equal(got, all) {
		t.Errorf("default %v differs from --state all %v", got, all)
	}
}

func TestListStateFilters(t *testing.T) {
	h := beavertest.New(t).Init()
	s := seedAllStates(t, h)

	cases := []struct {
		state string
		want  []string
	}{
		{"todo", []string{s.todo}},
		{"in-progress", []string{s.prog}},
		{"done", []string{s.done}},
		{"cancelled", []string{s.cancel}},
	}
	for _, c := range cases {
		if got := listIDs(t, h, "--state", c.state); !slices.Equal(got, c.want) {
			t.Errorf("--state %s = %v, want %v", c.state, got, c.want)
		}
	}
}

// seedAllStates deliberately misaligns creation time from id order, so output in
// creation order proves the display sort is doing the work, not the store's
// id-sorted file order.
func TestListOrdersByCreationNotFileOrder(t *testing.T) {
	h := beavertest.New(t).Init()
	s := seedAllStates(t, h)

	got := listIDs(t, h, "--state", "all")
	byCreation := []string{s.done, s.todo, s.cancel, s.prog}
	if !slices.Equal(got, byCreation) {
		t.Errorf("order = %v, want creation order %v", got, byCreation)
	}
	byID := append([]string(nil), got...)
	slices.Sort(byID)
	if slices.Equal(got, byID) {
		t.Fatal("fixture is too weak: creation order coincides with id order, so this test proves nothing")
	}
}

// Issues minted at the same instant (routine under a fixed clock) still need a
// reproducible order, so the id is the tiebreak.
func TestListOrderingTieBreaksByID(t *testing.T) {
	h := beavertest.New(t).Init()
	same := beavertest.DefaultNow
	seed(t, h, "mmm222", "b", issue.StateTodo, same)
	seed(t, h, "aaa111", "a", issue.StateTodo, same)
	seed(t, h, "zzz333", "c", issue.StateTodo, same)

	got := listIDs(t, h, "--state", "all")
	if want := []string{"aaa111", "mmm222", "zzz333"}; !slices.Equal(got, want) {
		t.Errorf("tie-break order = %v, want ascending id %v", got, want)
	}
}

func TestListJSONCarriesAllFields(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "aaa111", "Only issue", issue.StateTodo, beavertest.DefaultNow)

	rows := decodeArray(t, h.MustRun("list", "--state", "all").Stdout)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	row := rows[0]

	for _, k := range []string{"assignee", "priority", "parent"} {
		if v, ok := row[k]; !ok || v != nil {
			t.Errorf("unset field %q = %v (present=%v), want null", k, v, ok)
		}
	}
	for _, k := range []string{"labels", "depends_on"} {
		if arr, ok := row[k].([]any); !ok || len(arr) != 0 {
			t.Errorf("unset list %q = %v, want []", k, row[k])
		}
	}
	if _, ok := row["custom"].(map[string]any); !ok {
		t.Errorf("custom = %v, want an object", row["custom"])
	}
	if row["id"] != "aaa111" || row["title"] != "Only issue" || row["state"] != "todo" {
		t.Errorf("core fields wrong: id=%v title=%v state=%v", row["id"], row["title"], row["state"])
	}
	if row["created"] != "2026-06-27T18:30:00Z" {
		t.Errorf("created = %v, want the injected clock's time", row["created"])
	}
}

func TestListHumanTable(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seed(t, h, "aaa111", "Write the docs", issue.StateInProgress, beavertest.DefaultNow)

	out := h.MustRun("list").Stdout
	if trimmed := strings.TrimSpace(out); strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		t.Fatalf("expected a human table at a TTY, got JSON:\n%s", out)
	}
	for _, want := range []string{"ID", "STATE", "TITLE", "aaa111", "in-progress", "Write the docs"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q in:\n%s", want, out)
		}
	}
}

// An empty result must be [] (never null) for machines, whether the store is empty
// or a filter matched none.
func TestListEmptyRendersEmpty(t *testing.T) {
	h := beavertest.New(t).Init()

	if out := strings.TrimSpace(h.MustRun("list").Stdout); out != "[]" {
		t.Errorf("empty-store JSON = %q, want []", out)
	}

	seed(t, h, "aaa111", "todo item", issue.StateTodo, beavertest.DefaultNow)
	if out := strings.TrimSpace(h.MustRun("list", "--state", "done").Stdout); out != "[]" {
		t.Errorf("no-match JSON = %q, want []", out)
	}

	h.IsTTY = true
	if out := h.MustRun("list", "--state", "done").Stdout; !strings.Contains(out, "No issues") {
		t.Errorf("empty human list = %q, want a 'No issues' message", out)
	}
}

func TestListRequiresStore(t *testing.T) {
	h := beavertest.New(t) // no init
	r := h.Run("list")
	if r.Code != 3 {
		t.Errorf("list without a store exit = %d, want 3 (not-found)", r.Code)
	}
	if !strings.Contains(r.Stderr, "init") {
		t.Errorf("error should suggest init:\n%s", r.Stderr)
	}
}

func TestListUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	cases := [][]string{
		{"list", "--state", "bogus"},
		{"list", "--state", "open"},   // umbrella values are rejected
		{"list", "--state", "closed"}, // likewise
		{"list", "--state"},
		{"list", "todo"},
		{"list", "--format", "xml"},
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

// An invalid file is skipped so the command keeps serving the valid issues; the
// loud warning is asserted in TestListWarnsAndSkipsInvalidFiles.
func TestListSkipsInvalidFiles(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "aaa111", "valid one", issue.StateTodo, beavertest.DefaultNow)
	h.WriteFile("issues/broken.md", "this is not an issue file\n")

	if got := listIDs(t, h, "--state", "all"); !slices.Equal(got, []string{"aaa111"}) {
		t.Errorf("list with a broken file = %v, want just the valid issue", got)
	}
}

// --- helpers ---

// seeded holds the ID chosen for each state by seedAllStates.
type seeded struct{ todo, prog, done, cancel string }

// seedAllStates writes one issue in each of the four states. Creation times are
// staggered done<todo<cancel<prog and the IDs are chosen so id-sorted order
// differs from creation order, so ordering assertions can tell the two apart.
func seedAllStates(t *testing.T, h *beavertest.Harness) seeded {
	t.Helper()
	base := beavertest.DefaultNow
	s := seeded{todo: "ttodo0", prog: "pprog0", done: "ddone0", cancel: "ccncl0"}
	seed(t, h, s.done, "Ship release", issue.StateDone, base.Add(1*time.Minute))
	seed(t, h, s.todo, "Write docs", issue.StateTodo, base.Add(2*time.Minute))
	seed(t, h, s.cancel, "Drop feature", issue.StateCancelled, base.Add(3*time.Minute))
	seed(t, h, s.prog, "Fix bug", issue.StateInProgress, base.Add(4*time.Minute))
	return s
}

// seed writes an issue file directly, so tests can set arbitrary states and
// control ids and timestamps exactly.
func seed(t *testing.T, h *beavertest.Harness, id, title string, state issue.State, created time.Time) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: state, Created: created, Updated: created,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+issue.FileName(id, issue.Slug(title)), string(data))
}

// listIDs runs `beaver list` (JSON, since the harness is non-interactive) and
// returns the issue IDs in the order emitted.
func listIDs(t *testing.T, h *beavertest.Harness, args ...string) []string {
	t.Helper()
	r := h.MustRun(append([]string{"list"}, args...)...)
	return ids(decodeArray(t, r.Stdout))
}

// decodeArray parses s as a JSON array of objects.
func decodeArray(t *testing.T, s string) []map[string]any {
	t.Helper()
	var v []map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("decode JSON array: %v\ninput: %s", err, s)
	}
	return v
}

func ids(rows []map[string]any) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		id, _ := r["id"].(string)
		out = append(out, id)
	}
	return out
}
