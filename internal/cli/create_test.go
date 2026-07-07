package cli_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/busy-beaver/internal/beavertest"
	"github.com/builtbystef/busy-beaver/internal/issue"
)

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
