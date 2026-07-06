package cli_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

// AC: `edit` opens the file in $EDITOR and the result is re-validated. The fake
// editor stands in for a human hand-editing the raw file — here flipping the state
// and adding a description — and the change must land on disk (re-read through
// show), not merely be echoed.
func TestEditOpensEditorAndRevalidates(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	seed(t, h, "iss001", "Original title", issue.StateTodo, beavertest.DefaultNow)

	h.Editor = editWith(t, func(s string) string {
		s = strings.Replace(s, "state: todo", "state: in-progress", 1)
		return s + "\nA description typed in the editor.\n"
	})

	r := h.MustRun("edit", "iss001")
	if !strings.Contains(r.Stdout, "iss001") {
		t.Errorf("edit should confirm the issue it edited:\n%s", r.Stdout)
	}

	shown := h.DecodeJSON(h.MustRun("show", "iss001", "--format", "json").Stdout)
	if shown["state"] != "in-progress" {
		t.Errorf("edited state = %v, want in-progress (the hand-edit must persist)", shown["state"])
	}
	if body, _ := shown["body"].(string); !strings.Contains(body, "typed in the editor") {
		t.Errorf("edited body not persisted: %q", body)
	}
}

// AC: a non-interactive `edit` errors instead of hanging. The refusal must come
// before any editor is spawned (a real editor would block forever on a non-tty
// stdin), so the fake editor here fails the test if it is ever opened, and the
// file is left byte-for-byte unchanged.
func TestEditRefusesNonInteractive(t *testing.T) {
	h := beavertest.New(t).Init() // non-interactive by default
	seed(t, h, "iss001", "Some work", issue.StateTodo, beavertest.DefaultNow)
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Editor = neverCalled(t)

	r := h.Run("edit", "iss001")
	if r.Code == 0 {
		t.Error("edit in a non-interactive session should fail, got exit 0")
	}
	stderr := strings.ToLower(r.Stderr)
	if !strings.Contains(stderr, "terminal") && !strings.Contains(stderr, "interactive") {
		t.Errorf("refusal should explain the missing terminal:\n%s", r.Stderr)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("a refused edit modified the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// The re-validation has teeth: an edit that leaves the file no longer a usable
// issue (here an illegal state) fails and says so. The file is left as the human
// saved it — ADR 0005 tolerates an invalid file for doctor rather than reverting
// the human's work — so it is still present, not deleted.
func TestEditRevalidationRejectsBrokenResult(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	seed(t, h, "iss001", "Some work", issue.StateTodo, beavertest.DefaultNow)

	h.Editor = editWith(t, func(s string) string {
		return strings.Replace(s, "state: todo", "state: bogus", 1)
	})

	r := h.Run("edit", "iss001")
	if r.Code == 0 {
		t.Error("edit that leaves the file invalid should fail, got exit 0")
	}
	if !strings.Contains(strings.ToLower(r.Stderr), "valid") {
		t.Errorf("error should report the invalid result:\n%s", r.Stderr)
	}
	if files := h.IssueFiles(); len(files) != 1 {
		t.Errorf("edit must not remove the file it validated; files = %v", files)
	}
}

// AC: interactive `create` opens an editor for the description. With no title on
// the command line but an interactive session, create seeds the new file, hands it
// to the editor, and takes the title and body the human writes. The empty-title
// skeleton is canonicalized away, leaving exactly one file at its <id>-<slug> name.
func TestInteractiveCreateOpensEditor(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true

	h.Editor = editWith(t, func(s string) string {
		return setTitleLine(s, "Authored in the editor") + "\nThe description body.\n"
	})

	out := h.DecodeJSON(h.MustRun("create", "--format", "json").Stdout)
	if out["title"] != "Authored in the editor" {
		t.Errorf("title = %v, want the editor-supplied title", out["title"])
	}
	if body, _ := out["body"].(string); !strings.Contains(body, "The description body") {
		t.Errorf("body = %q, want the editor-supplied description", body)
	}
	id, _ := out["id"].(string)

	want := id + "-authored-in-the-editor.md"
	if files := h.IssueFiles(); !slices.Equal(files, []string{want}) {
		t.Errorf("issue files = %v, want exactly [%s] (skeleton canonicalized, not left behind)", files, want)
	}
	if shown := h.DecodeJSON(h.MustRun("show", id, "--format", "json").Stdout); shown["title"] != "Authored in the editor" {
		t.Errorf("persisted title = %v, want the editor-supplied title", shown["title"])
	}
}

// AC: a non-interactive `create` still requires a title. With no title and no
// terminal to open an editor in, it is a plain usage error — and no editor is
// spawned and no file is written.
func TestNonInteractiveCreateRequiresTitle(t *testing.T) {
	h := beavertest.New(t).Init() // non-interactive
	h.Editor = neverCalled(t)

	r := h.Run("create")
	if r.Code != 2 {
		t.Errorf("non-interactive create with no title exit = %d, want 2 (usage)", r.Code)
	}
	if !strings.Contains(strings.ToLower(r.Stderr), "title") {
		t.Errorf("error should say a title is required:\n%s", r.Stderr)
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("a failed create wrote files: %v", files)
	}
}

// A title on the command line is sufficient input, so an interactive create with
// one goes straight to writing the issue and never opens the editor.
func TestInteractiveCreateWithTitleSkipsEditor(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	h.Editor = neverCalled(t)

	out := h.DecodeJSON(h.MustRun("create", "Given on the command line", "--format", "json").Stdout)
	if out["title"] != "Given on the command line" {
		t.Errorf("title = %v, want the command-line title", out["title"])
	}
}

// If the editor produces a file with no title, create fails (the one input it
// fundamentally needs is missing) and clears the skeleton out of the issues
// directory, so an aborted authoring never leaves a half-formed issue in the
// store. What the human typed is not theirs to lose: the changed file is stashed
// under .beaver/drafts and its location reported.
func TestInteractiveCreateAbortsOnEmptyTitle(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	h.NewID = func() string { return "aaaaaa" }
	h.Editor = editWith(t, func(s string) string { return s + "\nA body but no title.\n" })

	r := h.Run("create")
	if r.Code == 0 {
		t.Error("create with no title supplied in the editor should fail, got exit 0")
	}
	if !strings.Contains(strings.ToLower(r.Stderr), "title") {
		t.Errorf("error should mention the missing title:\n%s", r.Stderr)
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("aborted create left a skeleton behind: %v", files)
	}
	if !strings.Contains(r.Stderr, "draft") {
		t.Errorf("stderr should point at the stashed draft:\n%s", r.Stderr)
	}
	if draft := h.ReadFile("drafts/aaaaaa.md"); !strings.Contains(draft, "A body but no title.") {
		t.Errorf("draft should hold what the human typed:\n%s", draft)
	}
}

// If the editor leaves the file unparseable, create reports it and likewise clears
// the skeleton out of the issues directory rather than importing a broken issue —
// but the human's typed content is stashed as a draft, never deleted.
func TestInteractiveCreateCleansUpInvalidResult(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	h.NewID = func() string { return "aaaaaa" }
	h.Editor = editWith(t, func(string) string { return "not a valid issue file\n" })

	r := h.Run("create")
	if r.Code == 0 {
		t.Error("create whose editor result is invalid should fail, got exit 0")
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("aborted create left an invalid skeleton behind: %v", files)
	}
	if draft := h.ReadFile("drafts/aaaaaa.md"); !strings.Contains(draft, "not a valid issue file") {
		t.Errorf("draft should hold what the human saved:\n%s", draft)
	}
}

// A skeleton the human never changed — they quit the editor without writing — is
// plain junk: it is deleted outright, and no draft is stashed.
func TestInteractiveCreateAbandonedUnchangedIsDeleted(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	h.NewID = func() string { return "aaaaaa" }
	h.Editor = editWith(t, func(s string) string { return s }) // opened, saved untouched

	r := h.Run("create")
	if r.Code == 0 {
		t.Error("an abandoned create should fail (no title), got exit 0")
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("abandoned create left a skeleton behind: %v", files)
	}
	if _, err := os.Stat(filepath.Join(h.BeaverDir(), "drafts", "aaaaaa.md")); !os.IsNotExist(err) {
		t.Errorf("an untouched skeleton must not be stashed as a draft (stat err = %v)", err)
	}
}

// AC: `delete` removes the issue file, and subsequent commands no longer see it.
func TestDeleteRemovesIssue(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "keep11", "Keep me", issue.StateTodo, beavertest.DefaultNow)
	seed(t, h, "junk22", "Junk duplicate", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("delete", "junk22").Stdout)
	if out["deleted"] != true || out["id"] != "junk22" {
		t.Errorf("delete result = %v, want junk22 deleted", out)
	}

	if r := h.Run("show", "junk22"); r.Code != 3 {
		t.Errorf("show of a deleted issue exit = %d, want 3 (not-found)", r.Code)
	}
	if got := listIDs(t, h); !slices.Equal(got, []string{"keep11"}) {
		t.Errorf("list after delete = %v, want [keep11]", got)
	}
	if files := h.IssueFiles(); !slices.Equal(files, []string{"keep11-keep-me.md"}) {
		t.Errorf("files after delete = %v, want only the kept issue", files)
	}
}

// delete's error paths mirror the other issue-addressing commands: a missing issue
// is not-found (exit 3), a missing ref is a usage error (exit 2), and no store at
// all is not-found (exit 3) pointing at init.
func TestDeleteNotFoundAndUsage(t *testing.T) {
	h := beavertest.New(t).Init()
	if r := h.Run("delete", "zzzzzz"); r.Code != 3 {
		t.Errorf("delete of a missing issue exit = %d, want 3 (not-found)", r.Code)
	}
	if r := h.Run("delete"); r.Code != 2 {
		t.Errorf("delete with no ref exit = %d, want 2 (usage)", r.Code)
	}

	noStore := beavertest.New(t) // no init
	r := noStore.Run("delete", "x")
	if r.Code != 3 {
		t.Errorf("delete without a store exit = %d, want 3 (not-found)", r.Code)
	}
	if !strings.Contains(r.Stderr, "init") {
		t.Errorf("delete without a store should suggest init:\n%s", r.Stderr)
	}
}

// --- helpers ---

// editWith returns a fake editor standing in for $EDITOR: it reads the file it is
// handed, applies transform to the contents, and writes the result back — the same
// read-modify-write a human performs in an editor.
func editWith(t *testing.T, transform func(string) string) func(string) error {
	t.Helper()
	return func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte(transform(string(data))), 0o644)
	}
}

// neverCalled returns a fake editor that fails the test if it is ever opened —
// used to prove a command refused, or short-circuited, before spawning one.
func neverCalled(t *testing.T) func(string) error {
	t.Helper()
	return func(string) error {
		t.Error("the editor was opened, but the command should not have opened one")
		return nil
	}
}

// setTitleLine rewrites the `title:` frontmatter line to the given title,
// independent of the exact form the empty-title skeleton serializes to.
func setTitleLine(s, title string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(ln, "title:") {
			lines[i] = "title: " + title
			break
		}
	}
	return strings.Join(lines, "\n")
}
