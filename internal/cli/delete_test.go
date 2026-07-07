package cli_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/busy-beaver/internal/beavertest"
	"github.com/builtbystef/busy-beaver/internal/issue"
)

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
