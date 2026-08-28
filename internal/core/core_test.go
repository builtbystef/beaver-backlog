package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/store"
)

func TestOpenWalksUpToTheStore(t *testing.T) {
	root := newStore(t)
	deep := filepath.Join(root, "src", "pkg", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	seed(t, root, mkIssue("aaa111", "Findable"))

	svc := open(t, deep)
	detail, err := svc.Get("aaa111")
	if err != nil {
		t.Fatalf("Get from a subdirectory: %v", err)
	}
	if detail.Issue.Title != "Findable" {
		t.Errorf("title = %q, want Findable", detail.Issue.Title)
	}
}

func TestOpenWithoutAStore(t *testing.T) {
	if _, err := core.Open(t.TempDir()); !errors.Is(err, core.ErrNoStore) {
		t.Errorf("Open outside a store = %v, want ErrNoStore", err)
	}
}

// Get answers with the relationship facts no file stores: what the issue waits
// on, its readiness, and the inverse edges derived by scanning the rest.
func TestGetDerivesRelationships(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("dep111", "Dependency"))
	seed(t, root, withDeps(mkIssue("mid222", "Middle"), "dep111"))
	seed(t, root, withParent(mkIssue("kid333", "Child"), "mid222"))

	detail, err := open(t, root).Get("mid222")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	rel := detail.Relationship
	if len(rel.BlockedOn) != 1 || rel.BlockedOn[0].ID != "dep111" {
		t.Errorf("blocked_on = %v, want the todo dependency dep111", rel.BlockedOn)
	}
	if !rel.Blocked || rel.Ready || rel.Stuck {
		t.Errorf("readiness = blocked:%v ready:%v stuck:%v, want blocked only", rel.Blocked, rel.Ready, rel.Stuck)
	}
	if !slices.Equal(rel.Children, []string{"kid333"}) {
		t.Errorf("children = %v, want [kid333]", rel.Children)
	}
	if detail2, err := open(t, root).Get("dep111"); err != nil {
		t.Fatalf("Get dependency: %v", err)
	} else if !slices.Equal(detail2.Relationship.Blocks, []string{"mid222"}) {
		t.Errorf("blocks = %v, want [mid222]", detail2.Relationship.Blocks)
	}
}

// A reference names an issue by id, slug, or file name, canonical or stale; the
// core resolves all of them to the same issue.
func TestGetResolvesEveryReferenceForm(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("aaa111", "Write the docs"))

	for _, ref := range []string{"aaa111", "write-the-docs", "aaa111-write-the-docs", "aaa111-stale-name.md"} {
		detail, err := open(t, root).Get(ref)
		if err != nil {
			t.Errorf("Get(%q): %v", ref, err)
			continue
		}
		if detail.Issue.ID != "aaa111" {
			t.Errorf("Get(%q) resolved to %q, want aaa111", ref, detail.Issue.ID)
		}
	}
}

func TestGetUnknownRefIsNotFound(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("aaa111", "Only issue"))

	_, err := open(t, root).Get("nope")
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Get of an unknown ref = %v, want ErrNotFound", err)
	}
}

// A slug two issues share names no single issue: the error carries the
// candidates so a caller can list them, and still reads as a not-found.
func TestGetSharedSlugIsAmbiguous(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("zzz999", "Same title"))
	seed(t, root, mkIssue("aaa111", "Same title"))

	_, err := open(t, root).Get("same-title")
	var ambiguous *core.AmbiguousRefError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("Get of a shared slug = %v, want *AmbiguousRefError", err)
	}
	if ambiguous.Ref != "same-title" {
		t.Errorf("Ref = %q, want same-title", ambiguous.Ref)
	}
	if got := ids(ambiguous.Matches); !slices.Equal(got, []string{"aaa111", "zzz999"}) {
		t.Errorf("matches = %v, want both candidates sorted by id", got)
	}
	if !errors.Is(err, core.ErrNotFound) {
		t.Error("an ambiguous reference should still read as a not-found")
	}
}

// An unusable file is data, not a failure: the read serves the valid issues and
// hands back what it skipped for the caller to report (ADR 0003).
func TestReadsReportSkippedFilesAsWarnings(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("good11", "Keep me"))
	writeRaw(t, root, "broken.md", "this is not an issue file\n")

	listing, err := open(t, root).List(core.Query{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := ids(listing.Issues); !slices.Equal(got, []string{"good11"}) {
		t.Errorf("listed %v, want just the valid issue", got)
	}
	assertWarnsAbout(t, "List", listing.Warnings)

	detail, err := open(t, root).Get("good11")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertWarnsAbout(t, "Get", detail.Warnings)
}

// A fingerprint answers one question, whether anything about the files has
// changed since last time, so it must move for a written, edited, or deleted
// issue and hold still for a store nobody touched.
func TestFingerprintMovesOnlyWhenTheFilesDo(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("aaa111", "Groundwork"))
	svc := open(t, root)

	first := fingerprint(t, svc)
	if again := fingerprint(t, svc); again != first {
		t.Errorf("fingerprint of an untouched store moved: %q then %q", first, again)
	}

	seed(t, root, mkIssue("bbb222", "Something new"))
	added := fingerprint(t, svc)
	if added == first {
		t.Error("fingerprint did not move for an added issue")
	}

	seed(t, root, withDeps(mkIssue("aaa111", "Groundwork"), "bbb222"))
	edited := fingerprint(t, svc)
	if edited == added {
		t.Error("fingerprint did not move for an edited issue")
	}

	if _, err := svc.Delete("bbb222"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted := fingerprint(t, svc); deleted == edited {
		t.Error("fingerprint did not move for a deleted issue")
	}
}

// A store whose issues directory is gone is an empty store, not a failure: the
// same forgiveness every other read gives it.
func TestFingerprintOfAStoreWithoutIssues(t *testing.T) {
	root := newStore(t)
	svc := open(t, root)
	if err := os.RemoveAll(filepath.Join(root, ".beaver", "issues")); err != nil {
		t.Fatalf("remove issues dir: %v", err)
	}
	if _, err := svc.Fingerprint(); err != nil {
		t.Errorf("Fingerprint without an issues directory = %v, want no error", err)
	}
}

func fingerprint(t *testing.T, svc *core.Service) string {
	t.Helper()
	fp, err := svc.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	return fp
}

// The warnings belong to the scan, not to the answer, so a lookup that fails
// still tells the caller which files it skipped.
func TestFailedGetStillReportsWarnings(t *testing.T) {
	root := newStore(t)
	writeRaw(t, root, "broken.md", "this is not an issue file\n")

	detail, err := open(t, root).Get("nope")
	if !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("Get = %v, want ErrNotFound", err)
	}
	assertWarnsAbout(t, "a failed Get", detail.Warnings)
}

// assertWarnsAbout checks that a read reported the one seeded broken file,
// naming both the file and what is wrong with it.
func assertWarnsAbout(t *testing.T, what string, warnings []core.Warning) {
	t.Helper()
	if len(warnings) != 1 {
		t.Fatalf("%s reported %d warnings, want 1", what, len(warnings))
	}
	w := warnings[0]
	if filepath.Base(w.Path) != "broken.md" {
		t.Errorf("%s warned about %q, want broken.md", what, w.Path)
	}
	if w.Err == nil || !strings.Contains(w.Err.Error(), "frontmatter") {
		t.Errorf("%s warning reason = %v, want the frontmatter problem", what, w.Err)
	}
}

var fixedTime = time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)

// mkIssue builds a minimal todo issue at the fixed time.
func mkIssue(id, title string) issue.Issue {
	return issue.Issue{ID: id, Title: title, State: issue.StateTodo, Created: fixedTime, Updated: fixedTime}
}

func withState(iss issue.Issue, state issue.State) issue.Issue {
	iss.State = state
	return iss
}

func withDeps(iss issue.Issue, deps ...string) issue.Issue {
	iss.DependsOn = deps
	return iss
}

func withParent(iss issue.Issue, parent string) issue.Issue {
	iss.Parent = parent
	return iss
}

// The project's name is the store directory's until the project says otherwise.
// Naming it records the name for every later reader, a fresh service included.
func TestNamingTheProjectOverridesTheDirectoryName(t *testing.T) {
	root := filepath.Join(t.TempDir(), "orbital-mechanics")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	svc := open(t, root)

	if name, err := svc.ConfiguredProjectName(); err != nil || name != "" {
		t.Errorf("a fresh store reports the configured name %q (err %v), want none", name, err)
	}
	if got := svc.ProjectName(); got != "orbital-mechanics" {
		t.Errorf("project name = %q, want the directory's %q", got, "orbital-mechanics")
	}

	if err := svc.SetProjectName("Apollo Guidance"); err != nil {
		t.Fatalf("SetProjectName: %v", err)
	}

	if name, err := svc.ConfiguredProjectName(); err != nil || name != "Apollo Guidance" {
		t.Errorf("configured name = %q (err %v), want the name just set", name, err)
	}
	if got := open(t, root).ProjectName(); got != "Apollo Guidance" {
		t.Errorf("a service opened afterwards calls the project %q, want the configured name", got)
	}
}

// open returns a service over the store found from dir, failing the test if
// there is none.
func open(t *testing.T, dir string) *core.Service {
	t.Helper()
	svc, err := core.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return svc
}

func newStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, _, err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return root
}

// seed writes an issue to its canonical file, so tests can set ids, states, and
// timestamps exactly rather than going through a write path.
func seed(t *testing.T, root string, iss issue.Issue) {
	t.Helper()
	data, err := issue.Marshal(iss)
	if err != nil {
		t.Fatalf("marshal seed %s: %v", iss.ID, err)
	}
	writeRaw(t, root, issue.FileName(iss.ID, issue.Slug(iss.Title)), string(data))
}

// writeRaw writes literal bytes as an issue file, for seeding content Marshal
// would never produce.
func writeRaw(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".beaver", "issues", name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func ids(issues []issue.Issue) []string {
	out := make([]string, len(issues))
	for i, iss := range issues {
		out[i] = iss.ID
	}
	return out
}
