package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/busy-beaver/internal/issue"
	"github.com/builtbystef/busy-beaver/internal/store"
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

// Resolve finds an issue by its frontmatter id even when the filename has drifted.
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

// A corrupt file carries no identity Resolve can match, even when its filename
// hints at one: ErrNotFound, never a match on the filename.
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

// The ID is deliberately unrelated to the slug, so a hit proves slug matching
// and not a coincidental match on the ID.
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

// A slug two issues share resolves to no single issue: a not-found that also
// carries the candidates, sorted by ID.
func TestResolveSharedSlugIsNotFoundWithCandidates(t *testing.T) {
	root := newStore(t)
	// Path order (zzz, aaa) is the reverse of ID order (aaa111, bbb222), so a
	// passing sort assertion proves the ordering is on the ID.
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
	// Each remains reachable by its ID and its full "<id>-<slug>" name.
	for _, id := range []string{"aaa111", "bbb222"} {
		if got, _, err := st.Resolve(id); err != nil || got.ID != id {
			t.Errorf("Resolve(%q) = %q, %v; want it to resolve", id, got.ID, err)
		}
		if got, _, err := st.Resolve(id + "-fix-bug"); err != nil || got.ID != id {
			t.Errorf("Resolve(%q) = %q, %v; want it to resolve", id+"-fix-bug", got.ID, err)
		}
	}
}

// An exact ID wins over another issue whose title slugifies to the same string.
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

// The slug is the title's canonical slug, never the filename's — a drifted
// filename neither creates a phantom match nor hides the real one.
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

// An empty reference matches nothing — not even an issue whose title slugifies
// to the empty string. That issue stays reachable by its ID.
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

// A stale on-disk name (id joined to an outdated slug, .md suffix optional)
// resolves via its id part; a made-up name around a nonexistent id stays not-found.
func TestResolveStaleFileName(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "abc123-old-name.md", mkIssue("abc123", "New Name"))
	st, _ := store.Discover(root)

	for _, ref := range []string{"abc123-old-name", "abc123-old-name.md", "abc123.md"} {
		if got, _, err := st.Resolve(ref); err != nil || got.ID != "abc123" {
			t.Errorf("Resolve(%q) = %q, %v; want abc123 via the id part", ref, got.ID, err)
		}
	}
	if _, _, err := st.Resolve("zzzzzz-anything"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Resolve(zzzzzz-anything) = %v, want ErrNotFound (no such id exists)", err)
	}
}

// The stale-name fallback never shadows a living slug, even when the
// reference's leading segment happens to be another issue's id.
func TestResolveSlugBeatsStaleNameFallback(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "fix.md", mkIssue("fix", "Unrelated"))
	writeIssueFile(t, root, "abc123-fix-login.md", mkIssue("abc123", "Fix login"))
	st, _ := store.Discover(root)

	got, _, err := st.Resolve("fix-login")
	if err != nil || got.ID != "abc123" {
		t.Errorf("Resolve(fix-login) = %q, %v; want abc123 (the slug match, not the id prefix)", got.ID, err)
	}
}

// A Snapshot answers Resolve, Issues, and IDTaken from one scan with the
// store's exact contracts.
func TestSnapshotAnswersLikeTheStore(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "aaa111-first.md", mkIssue("aaa111", "First"))
	writeIssueFile(t, root, "bbb222-second.md", mkIssue("bbb222", "Second"))
	st, _ := store.Discover(root)

	snap, err := st.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if got, _, err := snap.Resolve("first"); err != nil || got.ID != "aaa111" {
		t.Errorf("snapshot Resolve(first) = %q, %v; want aaa111", got.ID, err)
	}
	if _, _, err := snap.Resolve("nope"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("snapshot Resolve(nope) = %v, want ErrNotFound", err)
	}
	if !snap.IDTaken("bbb222") || snap.IDTaken("zzzzzz") {
		t.Errorf("IDTaken = %v/%v, want true/false", snap.IDTaken("bbb222"), snap.IDTaken("zzzzzz"))
	}
	issues := snap.Issues()
	if len(issues) != 2 || issues[0].ID != "aaa111" || issues[1].ID != "bbb222" {
		t.Errorf("Issues() = %+v, want both issues in path order", issues)
	}
}

// A deleted issues/ directory is a normal, recoverable state: List reports
// zero issues rather than leaking a raw OS error.
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

// Update overwrites an already-canonical file in place, leaving exactly one
// file for the id.
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

// Update writes the canonical file and removes a drifted one, so a
// read-modify-write never leaves two files with the same id.
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

// Rename moves a drifted file to its canonical name and nothing more: the
// bytes are the file's own, unrewritten.
func TestRenameMovesDriftedFileWithoutRewriting(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "drifted-name.md", issue.Issue{
		ID: "abc123", Title: "Real Title", State: issue.StateTodo,
		Created: fixedTime, Updated: fixedTime,
	})
	st, _ := store.Discover(root)
	iss, path, err := st.Resolve("abc123")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	newPath, err := st.Rename(path, iss)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}

	want := issue.FileName("abc123", issue.Slug("Real Title"))
	if filepath.Base(newPath) != want {
		t.Errorf("renamed to %q, want %q", filepath.Base(newPath), want)
	}
	files, _ := st.List()
	if len(files) != 1 || filepath.Base(files[0]) != want {
		t.Fatalf("files = %v, want exactly [%s]", files, want)
	}
	if after, _ := os.ReadFile(newPath); string(after) != string(before) {
		t.Errorf("Rename rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// Rename returns ErrNameCollision rather than overwrite a different file that
// already holds the canonical name.
func TestRenameRefusesToClobber(t *testing.T) {
	root := newStore(t)
	// A drifted file whose canonical name a *different* issue already occupies.
	writeIssueFile(t, root, "drifted-name.md", issue.Issue{
		ID: "abc123", Title: "Real Title", State: issue.StateTodo,
		Created: fixedTime, Updated: fixedTime,
	})
	writeIssueFile(t, root, issue.FileName("abc123", issue.Slug("Real Title")), mkIssue("zzz999", "Squatter"))

	st, _ := store.Discover(root)
	drifted := filepath.Join(root, ".beaver", "issues", "drifted-name.md")
	if _, err := st.Rename(drifted, mkIssue("abc123", "Real Title")); !errors.Is(err, store.ErrNameCollision) {
		t.Fatalf("Rename over an occupied name = %v, want ErrNameCollision", err)
	}
	// Both files survive untouched.
	if files, _ := st.List(); len(files) != 2 {
		t.Errorf("files = %v, want both still present", files)
	}
}

// Renaming a file already at its canonical name is a no-op success.
func TestRenameCanonicalIsNoOp(t *testing.T) {
	root := newStore(t)
	name := issue.FileName("abc123", issue.Slug("Already Canonical"))
	writeIssueFile(t, root, name, mkIssue("abc123", "Already Canonical"))
	st, _ := store.Discover(root)
	iss, path, _ := st.Resolve("abc123")

	newPath, err := st.Rename(path, iss)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if filepath.Base(newPath) != name {
		t.Errorf("newPath = %q, want unchanged %q", filepath.Base(newPath), name)
	}
}

// A scan skips each unusable file and reports it through the warning handler
// while still returning the valid issues. A drifted filename is lint, not a
// hard error: it loads with no warning.
func TestReadAllSkipsAndReportsInvalidFiles(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "good111-fine.md", mkIssue("good111", "Fine"))
	writeIssueFile(t, root, "drifted-name.md", mkIssue("lint11", "Untidy but valid"))

	const stamps = "created: 2026-06-27T18:30:00Z\nupdated: 2026-06-27T18:30:00Z\n"
	writeRaw(t, root, "bad-yaml.md", "this is not an issue file\n")
	writeRaw(t, root, "no-id.md", "---\ntitle: No id\nstate: todo\n"+stamps+"---\n")
	writeRaw(t, root, "bad-state.md", "---\nid: sta111\ntitle: Bad state\nstate: archived\n"+stamps+"---\n")

	st, _ := store.Discover(root)
	warned := map[string]error{}
	st.OnWarn(func(w store.Warning) { warned[filepath.Base(w.Path)] = w.Err })

	got, err := st.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	gotIDs := map[string]bool{}
	for _, iss := range got {
		gotIDs[iss.ID] = true
	}
	if len(got) != 2 || !gotIDs["good111"] || !gotIDs["lint11"] {
		t.Errorf("ReadAll returned %d issues %v, want just the two valid ones (good111, lint11)", len(got), gotIDs)
	}

	// One warning per invalid file; none for the valid-but-untidy file.
	wantReasons := map[string]string{"bad-yaml.md": "frontmatter", "no-id.md": "id", "bad-state.md": "state"}
	if len(warned) != len(wantReasons) {
		t.Fatalf("warnings = %v, want exactly one per invalid file %v", warned, wantReasons)
	}
	for file, sub := range wantReasons {
		err, ok := warned[file]
		if !ok || err == nil {
			t.Errorf("no warning for %s", file)
			continue
		}
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("warning for %s = %q, want it to mention %q", file, err, sub)
		}
	}
}

// With no handler set, an invalid file is skipped silently.
func TestScanSkipsSilentlyWithoutHandler(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "good111-fine.md", mkIssue("good111", "Fine"))
	writeRaw(t, root, "broken.md", "not an issue\n")

	st, _ := store.Discover(root) // no OnWarn
	got, err := st.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 1 || got[0].ID != "good111" {
		t.Errorf("ReadAll = %v, want just good111", got)
	}
}

// Read applies the store-wide usable-issue contract to one named file: a good
// file returns its issue, a failing one returns the reason.
func TestReadValidatesSingleFile(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "abc123-fine.md", mkIssue("abc123", "Fine"))
	writeRaw(t, root, "broken.md", "---\nid: bad999\ntitle: Bad\nstate: archived\ncreated: 2026-06-27T18:30:00Z\nupdated: 2026-06-27T18:30:00Z\n---\n")
	st, _ := store.Discover(root)
	issuesDir := filepath.Join(root, ".beaver", "issues")

	got, err := st.Read(filepath.Join(issuesDir, "abc123-fine.md"))
	if err != nil {
		t.Fatalf("Read of a valid file: %v", err)
	}
	if got.ID != "abc123" || got.Title != "Fine" {
		t.Errorf("read issue = %+v, want abc123/Fine", got)
	}

	if _, err := st.Read(filepath.Join(issuesDir, "broken.md")); err == nil {
		t.Error("Read of a file with an illegal state returned no error")
	} else if !strings.Contains(err.Error(), "state") {
		t.Errorf("Read error = %q, want it to name the invalid state", err)
	}
}

// Delete removes an issue's file from every read path; deleting an absent
// path is an error.
func TestDeleteRemovesFile(t *testing.T) {
	root := newStore(t)
	writeIssueFile(t, root, "abc123-junk.md", mkIssue("abc123", "Junk"))
	st, _ := store.Discover(root)
	_, path, err := st.Resolve("abc123")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if err := st.Delete(path); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if files, _ := st.List(); len(files) != 0 {
		t.Errorf("after Delete, List = %v, want empty", files)
	}
	if _, _, err := st.Resolve("abc123"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("after Delete, Resolve = %v, want ErrNotFound", err)
	}
	if err := st.Delete(path); err == nil {
		t.Error("Delete of an already-removed file returned no error")
	}
}

var fixedTime = time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)

// mkIssue builds a minimal todo issue at the fixed time.
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
	writeRaw(t, root, name, string(data))
}

// writeRaw writes literal bytes as an issue file, for seeding malformed or
// otherwise invalid content that Marshal would never produce.
func writeRaw(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".beaver", "issues", name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
