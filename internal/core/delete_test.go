package core_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// Deletion removes the file and nothing else: the result names what went, so a
// caller can confirm it after the file is gone.
func TestDeleteRemovesTheIssueFile(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("keep11", "Keep me"))
	seed(t, root, mkIssue("junk22", "Junk duplicate"))

	deleted, err := openAt(t, root).Delete("junk-duplicate")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.Issue.ID != "junk22" || deleted.Issue.Title != "Junk duplicate" {
		t.Errorf("deleted %+v, want the junk issue", deleted.Issue)
	}
	if deleted.Path == "" {
		t.Error("Path is empty; a caller cannot say which file went")
	}
	if got := issueFiles(t, root); !slices.Equal(got, []string{"keep11-keep-me.md"}) {
		t.Errorf("files after Delete = %v, want only the kept issue", got)
	}
	if _, err := open(t, root).Get("junk22"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Get of a deleted issue = %v, want ErrNotFound", err)
	}
}

func TestDeleteOfAnUnknownRefRemovesNothing(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("keep11", "Keep me"))

	if _, err := openAt(t, root).Delete("nope"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Delete of an unknown ref = %v, want ErrNotFound", err)
	}
	if got := issueFiles(t, root); !slices.Equal(got, []string{"keep11-keep-me.md"}) {
		t.Errorf("a failed Delete touched the store: %v", got)
	}
}

// The scan's warnings travel with the result, so deleting one issue still reports
// the broken file it read past.
func TestDeleteReportsSkippedFiles(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("junk22", "Junk"))
	writeRaw(t, root, "broken.md", "this is not an issue file\n")

	deleted, err := openAt(t, root).Delete("junk22")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertWarnsAbout(t, "Delete", deleted.Warnings)
}
