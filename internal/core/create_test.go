package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/clock"
	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestCreateWritesTheDraft(t *testing.T) {
	root := newStore(t)

	created, err := openWith(t, root, "new001").Create(core.Draft{
		Title:    "  Fix the parser  ",
		Body:     "It chokes on nested quotes.\n",
		Labels:   []string{"bug", "bug", "parser"}, // a repeated label is one label
		Priority: issue.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	iss := created.Issue
	if iss.ID != "new001" {
		t.Errorf("id = %q, want the minted new001", iss.ID)
	}
	if iss.Title != "Fix the parser" {
		t.Errorf("title = %q, want it trimmed", iss.Title)
	}
	if iss.State != issue.StateTodo {
		t.Errorf("state = %s, want todo (every issue starts unstarted)", iss.State)
	}
	if !slices.Equal(iss.Labels, []string{"bug", "parser"}) {
		t.Errorf("labels = %v, want the deduped [bug parser]", iss.Labels)
	}
	// A new issue has never been modified, so both stamps are the creating instant.
	if !iss.Created.Equal(writeTime) || !iss.Updated.Equal(writeTime) {
		t.Errorf("created/updated = %s/%s, want both %s", iss.Created, iss.Updated, writeTime)
	}
	if want := "new001-fix-the-parser.md"; filepath.Base(created.Path) != want {
		t.Errorf("path = %s, want the canonical %s", created.Path, want)
	}
	// The issue is on disk, not just in the returned value.
	detail, err := open(t, root).Get("new001")
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if detail.Issue.Body != "It chokes on nested quotes.\n" || detail.Issue.Priority != issue.PriorityHigh {
		t.Errorf("persisted body/priority = %q/%s, want the draft's", detail.Issue.Body, detail.Issue.Priority)
	}
}

// A draft that does not describe an issue that can exist is refused by field,
// with nothing written — the caller phrases the refusal its own way.
func TestCreateValidatesTheDraft(t *testing.T) {
	cases := []struct {
		name  string
		draft core.Draft
		field string
	}{
		{"no title", core.Draft{Title: ""}, "title"},
		{"blank title", core.Draft{Title: "   "}, "title"},
		{"unknown priority", core.Draft{Title: "Fine title", Priority: issue.Priority("critical")}, "priority"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := newStore(t)

			_, err := openWith(t, root, "new001").Create(c.draft)
			var invalid *core.ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("Create(%+v) = %v, want *ValidationError", c.draft, err)
			}
			if invalid.Field != c.field {
				t.Errorf("refused field = %q, want %q", invalid.Field, c.field)
			}
			if files := issueFiles(t, root); len(files) != 0 {
				t.Errorf("a refused draft wrote %v, want no file", files)
			}
		})
	}
}

// An edge takes any reference the resolver accepts but is stored as the
// canonical id, so no edge rests on a slug a retitling would break.
func TestCreateResolvesAndDedupesEdges(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("dep001", "Shared base"))
	seed(t, root, mkIssue("epc001", "The epic"))

	created, err := openWith(t, root, "new001").Create(core.Draft{
		Title:     "Uses the base",
		DependsOn: []string{"shared-base", "dep001", "dep001-shared-base"}, // one issue, three ways
		Parent:    "the-epic",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !slices.Equal(created.Issue.DependsOn, []string{"dep001"}) {
		t.Errorf("depends_on = %v, want the single resolved [dep001]", created.Issue.DependsOn)
	}
	if created.Issue.Parent != "epc001" {
		t.Errorf("parent = %q, want the resolved epc001", created.Issue.Parent)
	}
}

// A typo must not persist as a dangling edge, so the whole creation is refused
// before any file is written — and the error names the reference at fault, which
// the caller never had to track.
func TestCreateRefusesAnUnresolvableEdge(t *testing.T) {
	cases := map[string]core.Draft{
		"depends-on": {Title: "New issue", DependsOn: []string{"nope99"}},
		"parent":     {Title: "New issue", Parent: "nope99"},
	}
	for name, draft := range cases {
		t.Run(name, func(t *testing.T) {
			root := newStore(t)

			_, err := openWith(t, root, "new001").Create(draft)
			var unknown *core.UnknownRefError
			if !errors.As(err, &unknown) {
				t.Fatalf("Create with an unknown %s = %v, want *UnknownRefError", name, err)
			}
			if unknown.Ref != "nope99" {
				t.Errorf("Ref = %q, want the offending nope99", unknown.Ref)
			}
			if !errors.Is(err, core.ErrNotFound) {
				t.Error("an unknown reference should still read as a not-found")
			}
			if files := issueFiles(t, root); len(files) != 0 {
				t.Errorf("a refused create wrote %v, want no file", files)
			}
		})
	}
}

// Ids are random, so two issues can in principle collide; minting retries until
// the id is free rather than replacing the issue that holds it.
func TestCreateMintsPastACollision(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Already here"))

	created, err := openWith(t, root, "iss001", "fresh1").Create(core.Draft{Title: "The newcomer"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Issue.ID != "fresh1" {
		t.Errorf("id = %q, want the fresh1 drawn after the taken iss001", created.Issue.ID)
	}
	if files := issueFiles(t, root); len(files) != 2 {
		t.Errorf("issue files = %v, want both issues (the collision must not have replaced one)", files)
	}
}

// A write path reports the files it skipped like every read does, so a broken
// file is never hidden by a command that happened to succeed.
func TestCreateReportsSkippedFiles(t *testing.T) {
	root := newStore(t)
	writeRaw(t, root, "broken.md", "this is not an issue file\n")

	created, err := openWith(t, root, "new001").Create(core.Draft{Title: "Unbothered"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertWarnsAbout(t, "Create", created.Warnings)
}

// --- helpers ---

// openWith returns a service whose writes stamp writeTime and whose new issues
// draw the given ids in order, repeating the last — so a test can name the id a
// creation mints, and hand minting a collision to retry past.
func openWith(t *testing.T, dir string, ids ...string) *core.Service {
	t.Helper()
	next := 0
	svc, err := core.Open(dir,
		core.WithClock(clock.Fixed(writeTime)),
		core.WithIDSource(func() string {
			id := ids[len(ids)-1]
			if next < len(ids) {
				id = ids[next]
			}
			next++
			return id
		}))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return svc
}

// issueFiles lists the names of the store's issue files, so a test can prove a
// refusal wrote nothing.
func issueFiles(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".beaver", "issues"))
	if err != nil {
		t.Fatalf("read issues dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names
}
