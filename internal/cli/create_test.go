package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
)

// The title is an argument, never something a session is interactive enough to
// be asked for: a terminal gets the same usage error a pipe does.
func TestCreateAlwaysRequiresATitle(t *testing.T) {
	for _, tty := range []bool{false, true} {
		h := beavertest.New(t).Init()
		h.IsTTY, h.StdinIsTTY = tty, tty

		r := h.Run("create")
		if r.Code != 2 {
			t.Errorf("create with no title (tty=%v) exit = %d, want 2 (usage)", tty, r.Code)
		}
		if !strings.Contains(strings.ToLower(r.Stderr), "title") {
			t.Errorf("error should say a title is required:\n%s", r.Stderr)
		}
		if files := h.IssueFiles(); len(files) != 0 {
			t.Errorf("a failed create wrote files: %v", files)
		}
	}
}

func TestCreateWithBodyFlag(t *testing.T) {
	h := beavertest.New(t).Init()

	out := h.DecodeJSON(h.MustRun("create", "Fix the parser", "--body", "It chokes on nested quotes.", "--format", "json").Stdout)
	if out["body"] != "It chokes on nested quotes." {
		t.Errorf("body = %v, want the --body text", out["body"])
	}

	id, _ := out["id"].(string)
	file := h.ReadFile("issues/" + id + "-fix-the-parser.md")
	if !strings.Contains(file, "It chokes on nested quotes.") {
		t.Errorf("issue file should contain the --body text:\n%s", file)
	}
}

func TestCreateBodyFromStdin(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinText = "Line one.\n\nLine two, after a blank.\n"

	out := h.DecodeJSON(h.MustRun("create", "Fix the parser", "--body-file", "-", "--format", "json").Stdout)
	if out["body"] != "Line one.\n\nLine two, after a blank.\n" {
		t.Errorf("body = %q, want the piped stdin verbatim", out["body"])
	}
}

func TestCreateBodyFromFile(t *testing.T) {
	h := beavertest.New(t).Init()
	if err := os.WriteFile(filepath.Join(h.Dir, "body.md"), []byte("From a file.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A relative path resolves against the working directory, not the store.
	out := h.DecodeJSON(h.MustRun("create", "Fix the parser", "--body-file", "body.md", "--format", "json").Stdout)
	if out["body"] != "From a file.\n" {
		t.Errorf("body = %q, want the file content verbatim", out["body"])
	}
}

func TestCreateBodyFlagsAreExclusive(t *testing.T) {
	h := beavertest.New(t).Init()

	r := h.Run("create", "Fix the parser", "--body", "inline", "--body-file", "-")
	if r.Code != 2 {
		t.Errorf("create with both --body and --body-file exit = %d, want 2 (usage)", r.Code)
	}
	if !strings.Contains(r.Stderr, "mutually exclusive") {
		t.Errorf("error should say the flags are mutually exclusive:\n%s", r.Stderr)
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("a refused create wrote files: %v", files)
	}
}

func TestCreateBodyFileUnreadable(t *testing.T) {
	h := beavertest.New(t).Init()

	r := h.Run("create", "Fix the parser", "--body-file", "no-such-file.md")
	if r.Code != 1 {
		t.Errorf("create with an unreadable body file exit = %d, want 1", r.Code)
	}
	if files := h.IssueFiles(); len(files) != 0 {
		t.Errorf("a failed create wrote files: %v", files)
	}
}
