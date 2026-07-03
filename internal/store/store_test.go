package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"beaver/internal/issue"
	"beaver/internal/store"
)

func TestDiscoverFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	if _, _, err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	sub := filepath.Join(root, "src", "pkg", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	st, err := store.Discover(sub)
	if err != nil {
		t.Fatalf("Discover from subdirectory: %v", err)
	}
	if want := filepath.Join(root, ".beaver"); st.Root() != want {
		t.Errorf("root = %q, want %q", st.Root(), want)
	}
}

func TestDiscoverNoStore(t *testing.T) {
	if _, err := store.Discover(t.TempDir()); !errors.Is(err, store.ErrNoStore) {
		t.Errorf("got %v, want ErrNoStore", err)
	}
}

// Resolve must find an issue by its authoritative frontmatter id even when the
// filename has drifted from it (ADR 0002, ADR 0005).
func TestResolveFallsBackToFrontmatterID(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "totally-wrong-name.md", issue.Issue{
		ID: "m3k8", Title: "Drifted", State: issue.StateTodo,
		Created: fixedTime, Updated: fixedTime,
	})

	st, _ := store.Discover(root)
	got, _, err := st.Resolve("m3k8")
	if err != nil {
		t.Fatalf("Resolve by frontmatter id: %v", err)
	}
	if got.ID != "m3k8" {
		t.Errorf("resolved id = %q, want m3k8", got.ID)
	}
}

func TestResolveNotFound(t *testing.T) {
	st, _ := store.Discover(newStore(t))
	if _, _, err := st.Resolve("nope"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// A corrupt file has no readable frontmatter ID, so it carries no identity Resolve
// can match — even when its filename hints at one. Resolve reports ErrNotFound
// rather than trusting the filename (ADR 0002, ADR 0005); surfacing the corruption
// itself is doctor's job (b8q3).
func TestResolveSkipsCorruptFile(t *testing.T) {
	root := newStore(t)
	path := filepath.Join(root, ".beaver", "issues", "m3k8-broken.md")
	if err := os.WriteFile(path, []byte("this is not an issue file"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, _ := store.Discover(root)
	if _, _, err := st.Resolve("m3k8"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// A store whose issues/ directory has been deleted (a half-merged or hand-edited
// store) is a normal, recoverable state: List reports zero issues rather than
// leaking a raw OS error (ADR 0005).
func TestListMissingIssuesDir(t *testing.T) {
	root := newStore(t)
	if err := os.RemoveAll(filepath.Join(root, ".beaver", "issues")); err != nil {
		t.Fatal(err)
	}

	st, _ := store.Discover(root)
	files, err := st.List()
	if err != nil {
		t.Fatalf("List with missing issues dir: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

// Update overwrites an already-canonical file in place, returning the same path
// and leaving exactly one file for the id.
func TestUpdateOverwritesCanonicalInPlace(t *testing.T) {
	root := newStore(t)
	canonical := issue.FileName("abc123", issue.Slug("Fix the bug"))
	writeIssueFile(t, root, canonical, issue.Issue{
		ID: "abc123", Title: "Fix the bug", State: issue.StateTodo,
		Created: fixedTime, Updated: fixedTime,
	})
	st, _ := store.Discover(root)
	iss, path, err := st.Resolve("abc123")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	iss.State = issue.StateDone
	newPath, err := st.Update(path, iss)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if filepath.Base(newPath) != canonical {
		t.Errorf("newPath = %s, want canonical %s", filepath.Base(newPath), canonical)
	}
	if files, _ := st.List(); len(files) != 1 {
		t.Fatalf("want exactly one file, got %v", files)
	}
	if got, _, _ := st.Resolve("abc123"); got.State != issue.StateDone {
		t.Errorf("state = %q, want done", got.State)
	}
}

// Update renames a drifted file to its canonical name — writing the canonical file
// and removing the stale one — so a read-modify-write never leaves two files with
// the same id (ADR 0005).
func TestUpdateRenamesDriftedFile(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "totally-wrong-name.md", issue.Issue{
		ID: "abc123", Title: "Fix the bug", State: issue.StateTodo,
		Created: fixedTime, Updated: fixedTime,
	})
	st, _ := store.Discover(root)
	iss, path, err := st.Resolve("abc123")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	iss.State = issue.StateDone
	if _, err := st.Update(path, iss); err != nil {
		t.Fatalf("Update: %v", err)
	}

	want := issue.FileName("abc123", issue.Slug("Fix the bug"))
	files, _ := st.List()
	if len(files) != 1 || filepath.Base(files[0]) != want {
		t.Fatalf("files = %v, want exactly [%s] (drifted name removed)", files, want)
	}
	if got, _, _ := st.Resolve("abc123"); got.State != issue.StateDone {
		t.Errorf("state = %q, want done", got.State)
	}
}

var fixedTime = time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)

func newStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, _, err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return root
}

func writeIssueFile(t *testing.T, root, name string, iss issue.Issue) {
	t.Helper()
	data, err := issue.Marshal(iss)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".beaver", "issues", name), data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
