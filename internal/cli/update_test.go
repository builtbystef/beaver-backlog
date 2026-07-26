package cli_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The command surface is what these tests pin — flag parsing, exclusivity, usage
// errors, rendering, exit codes, and one happy path. What a change set does to an
// issue is the core's, and is covered at the core seam.

func TestUpdateChangesSeveralFieldsAtOnce(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, issue.PriorityLow, []string{"old"}, "", beavertest.DefaultNow)
	h.Clock.Advance(time.Hour)

	out := h.DecodeJSON(h.MustRun("update", "iss001",
		"--title", "Reworked work",
		"--assignee", "alice",
		"--priority", "high",
		"--label", "new,-old").Stdout)

	if out["title"] != "Reworked work" {
		t.Errorf("title = %v, want the new title", out["title"])
	}
	if out["assignee"] != "alice" {
		t.Errorf("assignee = %v, want alice", out["assignee"])
	}
	if out["priority"] != "high" {
		t.Errorf("priority = %v, want high", out["priority"])
	}
	if got := strSlice(out["labels"]); !slices.Equal(got, []string{"new"}) {
		t.Errorf("labels = %v, want [new] (old removed, new added)", got)
	}
	if out["state"] != "todo" {
		t.Errorf("state = %v, want it untouched — update never moves an issue's state", out["state"])
	}
	if out["updated"] == "2026-06-27T18:30:00Z" {
		t.Errorf("updated = %v, want bumped by the write", out["updated"])
	}
	// The change is on disk, not just echoed back.
	if got := showJSON(t, h, "iss001")["assignee"]; got != "alice" {
		t.Errorf("persisted assignee = %v, want alice", got)
	}
}

func TestUpdateClearsFields(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, issue.PriorityHigh, nil, "bob", beavertest.DefaultNow)
	seedDep(t, h, "prn001", "The parent", issue.StateTodo, nil, "")
	h.MustRun("update", "iss001", "--parent", "prn001")

	out := h.DecodeJSON(h.MustRun("update", "iss001", "--unassign", "--priority", "none", "--no-parent").Stdout)
	if out["assignee"] != nil {
		t.Errorf("assignee after --unassign = %v, want null", out["assignee"])
	}
	if out["priority"] != nil {
		t.Errorf("priority after --priority none = %v, want null", out["priority"])
	}
	if out["parent"] != nil {
		t.Errorf("parent after --no-parent = %v, want null", out["parent"])
	}
}

func TestUpdateAddsAndRemovesDependencies(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "iss001", "Work", issue.StateTodo, []string{"dep001"}, "")
	seedDep(t, h, "dep001", "First prerequisite", issue.StateTodo, nil, "")
	seedDep(t, h, "dep002", "Second prerequisite", issue.StateTodo, nil, "")

	out := h.DecodeJSON(h.MustRun("update", "iss001", "--depends-on", "+dep002", "--depends-on", "-dep001").Stdout)
	if got := strSlice(out["depends_on"]); !slices.Equal(got, []string{"dep002"}) {
		t.Errorf("depends_on = %v, want [dep002] (dep001 removed, dep002 added)", got)
	}
}

// The confirmation line names the issue, and a change that nets out to nothing
// says so rather than claiming a write that never happened.
func TestUpdateHumanLines(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	h.IsTTY = true

	if got := h.MustRun("update", "iss001", "--priority", "high").Stdout; got != "Updated iss001\n" {
		t.Errorf("human line = %q, want %q", got, "Updated iss001\n")
	}
	if got := h.MustRun("update", "iss001", "--priority", "high").Stdout; got != "iss001 is unchanged\n" {
		t.Errorf("no-op human line = %q, want %q", got, "iss001 is unchanged\n")
	}
}

// The net-change rule reaches the command surface: adding and removing the same
// label leaves the file, and `updated` with it, exactly as it was.
func TestUpdateNetNoopWritesNothing(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour) // a stray write would bump updated

	r := h.Run("update", "iss001", "--label", "+a,-a")
	if r.Code != 0 {
		t.Errorf("net no-op update exit = %d, want 0", r.Code)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("net no-op update rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// The description is the writer's to replace; the log is other actors' words, so
// --body leaves the notes section byte-identical.
func TestUpdateBodyPreservesNotes(t *testing.T) {
	h := beavertest.New(t).Init()
	notes := "## Notes\n\n**alice** — 2026-06-27T18:30:00Z\n\nTried the obvious fix; still broken."
	seedWithBody(t, h, "iss001", "Work", "The original description.\n\n"+notes)

	out := h.DecodeJSON(h.MustRun("update", "iss001", "--body", "A fresh description.").Stdout)
	body, _ := out["body"].(string)
	if strings.Contains(body, "The original description.") {
		t.Errorf("body still holds the replaced description:\n%s", body)
	}
	if !strings.Contains(body, "A fresh description.") {
		t.Errorf("body missing the new description:\n%s", body)
	}
	if !strings.Contains(body, notes) {
		t.Errorf("notes section was not preserved byte-for-byte:\n%s", body)
	}
}

func TestUpdateBodyFromStdin(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	h.StdinText = "Piped in.\n"

	out := h.DecodeJSON(h.MustRun("update", "iss001", "--body-file", "-").Stdout)
	if out["body"] != "Piped in.\n" {
		t.Errorf("body = %q, want the piped stdin verbatim", out["body"])
	}
}

func TestUpdateBodyFromFile(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	if err := os.WriteFile(filepath.Join(h.Dir, "body.md"), []byte("From a file.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A relative path resolves against the working directory, not the store.
	out := h.DecodeJSON(h.MustRun("update", "iss001", "--body-file", "body.md").Stdout)
	if out["body"] != "From a file.\n" {
		t.Errorf("body = %q, want the file content verbatim", out["body"])
	}
}

func TestUpdateBodyFileUnreadable(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	before := h.ReadFile("issues/" + h.IssueFiles()[0])

	r := h.Run("update", "iss001", "--body-file", "no-such-file.md")
	if r.Code != 1 {
		t.Errorf("update with an unreadable body file exit = %d, want 1", r.Code)
	}
	if after := h.ReadFile("issues/" + h.IssueFiles()[0]); after != before {
		t.Error("a failed update rewrote the issue file")
	}
}

// The title travels with the file name: the issue is written under its new slug,
// keeps its id, and leaves no second copy behind.
func TestUpdateTitleRenamesFile(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Old title", issue.StateTodo, "", nil, "", beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("update", "iss001", "--title", "New title").Stdout)
	if out["id"] != "iss001" {
		t.Errorf("id = %v, want iss001 — a retitling never changes identity", out["id"])
	}
	if files := h.IssueFiles(); !slices.Equal(files, []string{"iss001-new-title.md"}) {
		t.Errorf("issue files = %v, want exactly [iss001-new-title.md]", files)
	}
}

// A dependency that would close a loop is refused before anything is written.
func TestUpdateRefusesCycle(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "iss001", "First", issue.StateTodo, []string{"iss002"}, "")
	seedDep(t, h, "iss002", "Second", issue.StateTodo, nil, "")
	before := readIssueFile(t, h, "iss002", "Second")

	r := h.Run("update", "iss002", "--depends-on", "+iss001")
	if r.Code != 2 {
		t.Errorf("cycle-closing update exit = %d, want 2 (usage)", r.Code)
	}
	if !strings.Contains(r.Stderr, "cycle") || !strings.Contains(r.Stderr, "depends_on") {
		t.Errorf("error should name the field and the cycle:\n%s", r.Stderr)
	}
	if after := readIssueFile(t, h, "iss002", "Second"); after != before {
		t.Errorf("a refused update wrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestUpdateUnknownRef(t *testing.T) {
	h := beavertest.New(t).Init()

	r := h.Run("update", "zzzz", "--priority", "high")
	if r.Code != 3 {
		t.Errorf("update of an unknown ref exit = %d, want 3 (not found)", r.Code)
	}
	if !strings.Contains(r.Stderr, "zzzz") {
		t.Errorf("error should name the missing ref:\n%s", r.Stderr)
	}
}

func TestUpdateUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	cases := []struct {
		why  string
		args []string
	}{
		{"no reference", []string{"update"}},
		{"two references", []string{"update", "iss001", "iss002", "--priority", "high"}},
		{"nothing to change", []string{"update", "iss001"}},
		{"an empty title", []string{"update", "iss001", "--title", "  "}},
		{"an empty assignee", []string{"update", "iss001", "--assignee", ""}},
		{"an empty parent", []string{"update", "iss001", "--parent", ""}},
		{"a bad priority", []string{"update", "iss001", "--priority", "huge"}},
		{"a label naming nothing", []string{"update", "iss001", "--label", "-"}},
		{"a dependency naming nothing", []string{"update", "iss001", "--depends-on", "+"}},
	}
	for _, c := range cases {
		if r := h.Run(c.args...); r.Code != 2 {
			t.Errorf("update with %s exit = %d, want 2 (usage)", c.why, r.Code)
		}
	}
}

func TestUpdateExclusiveFlags(t *testing.T) {
	h := beavertest.New(t).Init()
	seedClassified(t, h, "iss001", "Work", issue.StateTodo, "", nil, "", beavertest.DefaultNow)
	before := h.ReadFile("issues/" + h.IssueFiles()[0])
	cases := [][]string{
		{"update", "iss001", "--body", "inline", "--body-file", "-"},
		{"update", "iss001", "--assignee", "alice", "--unassign"},
		{"update", "iss001", "--parent", "iss002", "--no-parent"},
	}
	for _, args := range cases {
		r := h.Run(args...)
		if r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
		if !strings.Contains(r.Stderr, "mutually exclusive") {
			t.Errorf("%v error should say the flags are mutually exclusive:\n%s", args, r.Stderr)
		}
	}
	if after := h.ReadFile("issues/" + h.IssueFiles()[0]); after != before {
		t.Error("a refused update rewrote the issue file")
	}
}

// --- helpers ---

// seedWithBody writes an issue file carrying a body, so a test can shape a
// description-and-notes pair no single command sequence produces.
func seedWithBody(t *testing.T, h *beavertest.Harness, id, title, body string) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: issue.StateTodo, Body: body,
		Created: beavertest.DefaultNow, Updated: beavertest.DefaultNow,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+issue.FileName(id, issue.Slug(title)), string(data))
}
