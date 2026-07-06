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

// The fake editor stands in for a human hand-editing the raw file.
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

// The refusal must come before any editor spawns — a real one would block forever
// on a non-tty stdin — so the fake editor fails the test if it is ever opened.
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

// An edit that leaves the file invalid fails, but the file stays as the human
// saved it — tolerated for doctor rather than reverting their work.
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

// The canonicalizing write trusts the id to name the file, so an authoring that
// rewrites the machine-owned id to an existing issue's would silently replace that
// issue. It is refused, and the authoring stashed as a draft, not discarded.
func TestInteractiveCreateRefusesEditedID(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	seed(t, h, "iss001", "Existing work", issue.StateTodo, beavertest.DefaultNow)
	before := h.ReadFile("issues/iss001-existing-work.md")

	h.NewID = func() string { return "fresh1" }
	h.Editor = editWith(t, func(s string) string {
		s = strings.Replace(s, "id: fresh1", "id: iss001", 1)
		return setTitleLine(s, "Existing work") + "\nA hijacking body.\n"
	})

	r := h.Run("create")
	if r.Code == 0 {
		t.Fatal("create that rewrote the minted id should fail, got exit 0")
	}
	if !strings.Contains(r.Stderr, "fresh1") || !strings.Contains(r.Stderr, "iss001") {
		t.Errorf("refusal should name the minted and the edited id:\n%s", r.Stderr)
	}
	if after := h.ReadFile("issues/iss001-existing-work.md"); after != before {
		t.Errorf("the existing issue's file was clobbered:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if files := h.IssueFiles(); !slices.Equal(files, []string{"iss001-existing-work.md"}) {
		t.Errorf("issue files = %v, want only the pre-existing issue", files)
	}
	if draft := h.ReadFile("drafts/fresh1.md"); !strings.Contains(draft, "A hijacking body.") {
		t.Errorf("the typed-into authoring should be stashed as a draft:\n%s", draft)
	}
}

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

func TestInteractiveCreateWithTitleSkipsEditor(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY, h.StdinIsTTY = true, true
	h.Editor = neverCalled(t)

	out := h.DecodeJSON(h.MustRun("create", "Given on the command line", "--format", "json").Stdout)
	if out["title"] != "Given on the command line" {
		t.Errorf("title = %v, want the command-line title", out["title"])
	}
}

// An aborted authoring must not leave a half-formed issue in the store, but what
// the human typed is not ours to lose: it is stashed as a draft, not deleted.
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

// An unparseable result is not imported, but the human's typed content is stashed
// as a draft, never deleted.
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

// A skeleton the human never changed is plain junk: deleted outright, no draft.
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

// editWith returns a fake $EDITOR that applies transform to the file it is handed —
// the same read-modify-write a human performs in an editor.
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
