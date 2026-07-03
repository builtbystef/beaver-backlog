package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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

// A slug resolves to its issue. The ID is deliberately unrelated to the slug, so a
// hit proves slug matching and not a coincidental match on the ID.
func TestResolveBySlug(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "q1w2e3-fix-the-login-bug.md", mkIssue("q1w2e3", "Fix the login bug"))
	st, _ := store.Discover(root)

	got, _, err := st.Resolve("fix-the-login-bug")
	if err != nil {
		t.Fatalf("Resolve slug: %v", err)
	}
	if got.ID != "q1w2e3" {
		t.Errorf("resolved id = %q, want q1w2e3", got.ID)
	}
}

// The full "<id>-<slug>" name — the canonical file name a user sees on disk —
// resolves to its issue, since it carries the unique ID.
func TestResolveByIDSlugName(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "q1w2e3-fix-the-login-bug.md", mkIssue("q1w2e3", "Fix the login bug"))
	st, _ := store.Discover(root)

	got, _, err := st.Resolve("q1w2e3-fix-the-login-bug")
	if err != nil {
		t.Fatalf("Resolve <id>-<slug>: %v", err)
	}
	if got.ID != "q1w2e3" {
		t.Errorf("resolved id = %q, want q1w2e3", got.ID)
	}
}

// A slug is derived from the mutable title and so is not unique. A slug two issues
// share names no single issue: it does not resolve, counting as a not-found
// (SharedSlugError Unwraps to ErrNotFound) while also carrying the candidates,
// sorted by ID, so a caller can list them. Each issue stays reachable by its unique
// ID and its full "<id>-<slug>" name.
func TestResolveSharedSlugIsNotFoundWithCandidates(t *testing.T) {
	root := newStore(t)
	// Drifted filenames whose path order (zzz, aaa) is the reverse of the ID order
	// (aaa111, bbb222), so a passing sort assertion proves the ordering is on the
	// ID and not an artifact of directory iteration.
	writeIssueFile(t, root, "zzz-fix-bug.md", mkIssue("aaa111", "Fix bug"))
	writeIssueFile(t, root, "aaa-fix-bug.md", mkIssue("bbb222", "Fix bug"))
	st, _ := store.Discover(root)

	_, _, err := st.Resolve("fix-bug")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("shared slug: got %v, want it to count as ErrNotFound", err)
	}
	var shared *store.SharedSlugError
	if !errors.As(err, &shared) {
		t.Fatalf("shared slug: got %v, want *SharedSlugError", err)
	}
	if shared.Slug != "fix-bug" {
		t.Errorf("Slug = %q, want fix-bug", shared.Slug)
	}
	ids := []string{shared.Matches[0].ID, shared.Matches[1].ID}
	if !slices.Equal(ids, []string{"aaa111", "bbb222"}) {
		t.Errorf("candidates = %v, want sorted [aaa111 bbb222]", ids)
	}
	// Each remains reachable by its unique ID and its full "<id>-<slug>" name.
	for _, id := range []string{"aaa111", "bbb222"} {
		if got, _, err := st.Resolve(id); err != nil || got.ID != id {
			t.Errorf("Resolve(%q) = %q, %v; want it to resolve", id, got.ID, err)
		}
		if got, _, err := st.Resolve(id + "-fix-bug"); err != nil || got.ID != id {
			t.Errorf("Resolve(%q) = %q, %v; want it to resolve", id+"-fix-bug", got.ID, err)
		}
	}
}

// Precedence: an exact full ID is the authoritative identity and wins over another
// issue whose title happens to slugify to the same string.
func TestResolveExactIDBeatsCoincidentSlug(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "abc123-target.md", mkIssue("abc123", "Target"))
	writeIssueFile(t, root, "zzzzzz-abc123.md", mkIssue("zzzzzz", "abc123")) // slug == "abc123"
	st, _ := store.Discover(root)

	got, _, err := st.Resolve("abc123")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.ID != "abc123" {
		t.Errorf("resolved id = %q, want abc123 (exact id wins over a coincident slug)", got.ID)
	}
}

// The slug is the title's canonical slug, never the filename's — so a drifted
// filename (ADR 0005) neither creates a phantom match on its stale slug nor hides
// the issue behind its real one.
func TestResolveSlugUsesTitleNotFilename(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "abc123-old-name.md", mkIssue("abc123", "New Name"))
	st, _ := store.Discover(root)

	if got, _, err := st.Resolve("new-name"); err != nil || got.ID != "abc123" {
		t.Errorf("Resolve(new-name) = %q, %v; want abc123 via the canonical slug", got.ID, err)
	}
	if _, _, err := st.Resolve("old-name"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Resolve(old-name) = %v; want ErrNotFound (a stale filename slug is not matched)", err)
	}
}

// An empty reference matches nothing — not even an issue whose title has no
// alphanumerics and so has an empty canonical slug. That issue stays reachable by
// its ID.
func TestResolveEmptyRefMatchesNothing(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "aaaaaa.md", mkIssue("aaaaaa", "!!!")) // empty slug
	st, _ := store.Discover(root)

	if _, _, err := st.Resolve(""); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Resolve(empty) = %v, want ErrNotFound", err)
	}
	if got, _, err := st.Resolve("aaaaaa"); err != nil || got.ID != "aaaaaa" {
		t.Errorf("Resolve(id) = %q, %v; want aaaaaa", got.ID, err)
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

// mkIssue builds a minimal todo issue at the fixed time — enough for resolution
// tests, which care only about id and title.
func mkIssue(id, title string) issue.Issue {
	return issue.Issue{ID: id, Title: title, State: issue.StateTodo, Created: fixedTime, Updated: fixedTime}
}

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
