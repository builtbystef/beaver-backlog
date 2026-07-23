package cli_test

import (
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/issue"
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
