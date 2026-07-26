package core_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The hand-editing seam hands out the file a reference resolves to, so an
// interface can put a human's editor on the issue itself.
func TestEditableNamesTheFileTheIssueLivesIn(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Some work"))

	target, err := openAt(t, root).Editable("some-work")
	if err != nil {
		t.Fatalf("Editable: %v", err)
	}
	if target.Issue.ID != "iss001" {
		t.Errorf("resolved to %q, want iss001", target.Issue.ID)
	}
	if data, err := os.ReadFile(target.Path); err != nil {
		t.Errorf("Path does not name a readable file: %v", err)
	} else if !strings.Contains(string(data), "id: iss001") {
		t.Errorf("Path names the wrong file:\n%s", data)
	}
}

func TestEditableOfAnUnknownRefIsNotFound(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Some work"))

	if _, err := openAt(t, root).Editable("nope"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Editable of an unknown ref = %v, want ErrNotFound", err)
	}
}

// A hand-edit is honored verbatim: what comes back is what the file now says,
// and the file is not rewritten — the name it drifted to is doctor's lint.
func TestRereadTakesAHandEditAsSaved(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Some work"))
	svc := openAt(t, root)
	target, err := svc.Editable("iss001")
	if err != nil {
		t.Fatalf("Editable: %v", err)
	}
	handEdit(t, target.Path, "state: todo", "state: in-progress")

	edited, err := svc.Reread(target.Path)
	if err != nil {
		t.Fatalf("Reread: %v", err)
	}
	if edited.State != issue.StateInProgress {
		t.Errorf("state = %s, want the hand-edited in-progress", edited.State)
	}
	if got := issueFiles(t, root); len(got) != 1 || got[0] != "iss001-some-work.md" {
		t.Errorf("files after a hand-edit = %v, want the one file, left where it was", got)
	}
}

// An edit that leaves the file unusable says why, and leaves the file exactly as
// the human saved it rather than reverting their work.
func TestRereadReportsAnEditThatBrokeTheIssue(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Some work"))
	svc := openAt(t, root)
	target, err := svc.Editable("iss001")
	if err != nil {
		t.Fatalf("Editable: %v", err)
	}
	handEdit(t, target.Path, "state: todo", "state: bogus")

	if _, err := svc.Reread(target.Path); err == nil {
		t.Fatal("Reread of an invalid result = nil, want the validation failure")
	} else if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("Reread error = %v, want it to name the invalid state", err)
	}
	if body := issueFile(t, root, "iss001-some-work.md"); !strings.Contains(body, "state: bogus") {
		t.Errorf("the human's saved file was rewritten:\n%s", body)
	}
}

// handEdit rewrites a file in place, standing in for a human editing it.
func handEdit(t *testing.T, path, old, new string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	edited := strings.Replace(string(data), old, new, 1)
	if edited == string(data) {
		t.Fatalf("hand-edit found no %q in %s", old, path)
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
