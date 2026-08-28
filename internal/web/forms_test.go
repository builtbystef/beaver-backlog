package web_test

// The write surface: what each form route lands in the file on disk, and how a
// refusal comes back. Rules (what a change set means, when a cycle is refused)
// belong to the core's own tests; what is asserted here is the mapping from a
// posted form to a core call and from a core error to a status.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestCreateFormIsReachableFromEveryPage(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, page := range []string{"/", "/issues", "/issues/" + target.ID, "/nope"} {
		if body := get(h, page).Body.String(); !strings.Contains(body, `href="/issues/new"`) {
			t.Errorf("%s does not link the create form:\n%s", page, body)
		}
	}
	if res := get(h, "/issues/new"); res.Code != http.StatusOK {
		t.Errorf("GET /issues/new = %d, want 200", res.Code)
	}
}

func TestCreateWritesTheIssueAndRedirectsToIt(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	blocker := create(t, svc, core.Draft{Title: "Groundwork"})
	parent := create(t, svc, core.Draft{Title: "The spec"})

	res := post(newHandler(t, dir), "/issues", url.Values{
		"title":       {"Fix flag parsing"},
		"description": {"The parser drops the last flag."},
		"priority":    {"high"},
		"labels":      {"bug, cli"},
		"depends_on":  {"groundwork"},
		"parent":      {parent.ID},
	})

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	created := lastCreated(t, svc, "Fix flag parsing")
	if want := "/issues/" + created.ID; res.Header().Get("Location") != want {
		t.Errorf("redirected to %q, want %q", res.Header().Get("Location"), want)
	}
	file := readIssueFile(t, dir, created.ID)
	for _, want := range []string{
		"title: Fix flag parsing", "priority: high", "bug", "cli",
		"depends_on:", blocker.ID, "parent: " + parent.ID,
		"The parser drops the last flag.",
	} {
		if !strings.Contains(file, want) {
			t.Errorf("file on disk missing %q:\n%s", want, file)
		}
	}
}

func TestEditFormAppliesTheWholeChangeSet(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	kept := create(t, svc, core.Draft{Title: "Kept blocker"})
	dropped := create(t, svc, core.Draft{Title: "Dropped blocker"})
	added := create(t, svc, core.Draft{Title: "Added blocker"})
	parent := create(t, svc, core.Draft{Title: "The spec"})
	target := create(t, svc, core.Draft{
		Title:     "Fix flag parsing",
		Body:      "Old words.",
		Priority:  issue.PriorityLow,
		Labels:    []string{"bug", "stale"},
		DependsOn: []string{kept.ID, dropped.ID},
	})
	h := newHandler(t, dir)

	if res := get(h, "/issues/"+target.ID+"/edit"); res.Code != http.StatusOK {
		t.Fatalf("GET edit form = %d, want 200", res.Code)
	}
	res := post(h, "/issues/"+target.ID, url.Values{
		"title":           {"Fix flag parsing properly"},
		"description":     {"New words."},
		"assignee":        {"stefan"},
		"priority":        {"urgent"},
		"keep_labels":     {"bug"}, // "stale" left unchecked, so it goes
		"add_labels":      {"cli"},
		"keep_depends_on": {kept.ID}, // dropped is left unchecked
		"add_depends_on":  {"added-blocker"},
		"parent":          {parent.ID},
	})

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	if want := "/issues/" + target.ID; res.Header().Get("Location") != want {
		t.Errorf("redirected to %q, want %q", res.Header().Get("Location"), want)
	}
	got := fetch(t, svc, target.ID)
	if got.Title != "Fix flag parsing properly" {
		t.Errorf("title = %q, want the edited one", got.Title)
	}
	if desc := issue.Description(got.Body); desc != "New words." {
		t.Errorf("description = %q, want %q", desc, "New words.")
	}
	if got.Assignee != "stefan" {
		t.Errorf("assignee = %q, want stefan", got.Assignee)
	}
	if got.Priority != issue.PriorityUrgent {
		t.Errorf("priority = %q, want urgent", got.Priority)
	}
	if want := []string{"bug", "cli"}; !slices.Equal(got.Labels, want) {
		t.Errorf("labels = %v, want %v", got.Labels, want)
	}
	if want := []string{kept.ID, added.ID}; !slices.Equal(got.DependsOn, want) {
		t.Errorf("depends_on = %v, want %v", got.DependsOn, want)
	}
	if got.Parent != parent.ID {
		t.Errorf("parent = %q, want %s", got.Parent, parent.ID)
	}

	// The same form clears what it can clear: an empty assignee unowns the
	// issue and an empty parent detaches it.
	if res := post(h, "/issues/"+target.ID, url.Values{
		"title":       {got.Title},
		"description": {"New words."},
		"assignee":    {""},
		"priority":    {"urgent"},
		"keep_labels": {"bug", "cli"},
		"parent":      {""},
	}); res.Code != http.StatusSeeOther {
		t.Fatalf("clearing edit = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	got = fetch(t, svc, target.ID)
	if got.Assignee != "" || got.Parent != "" {
		t.Errorf("assignee = %q and parent = %q, want both cleared", got.Assignee, got.Parent)
	}
	if want := []string{kept.ID, added.ID}; !slices.Equal(got.DependsOn, want) {
		t.Errorf("depends_on = %v after an edit that named none, want %v untouched", got.DependsOn, want)
	}
}

// Editing what the issue says never rewrites what was said about it.
func TestEditingTheDescriptionLeavesTheNotesByteIdentical(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	target := create(t, svc, core.Draft{Title: "Fix flag parsing", Body: "Old words."})
	note(t, svc, target.ID, "Reproduced on an empty argv.")
	note(t, svc, target.ID, "Cause is the loop bound.")
	before := notesSection(t, readIssueFile(t, dir, target.ID))

	res := post(newHandler(t, dir), "/issues/"+target.ID, url.Values{
		"title":       {"Fix flag parsing"},
		"description": {"Completely rewritten."},
		"priority":    {"none"},
	})

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	after := readIssueFile(t, dir, target.ID)
	if got := notesSection(t, after); got != before {
		t.Errorf("notes changed under an edit:\nbefore:\n%s\nafter:\n%s", before, got)
	}
	if !strings.Contains(after, "Completely rewritten.") {
		t.Errorf("the new description did not land:\n%s", after)
	}
}

func TestNoteFormAppendsAnAttributedEntry(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	target := create(t, svc, core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	res := post(h, "/issues/"+target.ID+"/notes", url.Values{"text": {"Handing this back."}})

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	if want := "/issues/" + target.ID; res.Header().Get("Location") != want {
		t.Errorf("redirected to %q, want %q", res.Header().Get("Location"), want)
	}
	notes := issue.ParseNotes(fetch(t, svc, target.ID).Body)
	if len(notes) != 1 {
		t.Fatalf("issue holds %d notes, want 1", len(notes))
	}
	if notes[0].Author != "tester" {
		t.Errorf("note author = %q, want the launch actor", notes[0].Author)
	}
	if notes[0].Time.IsZero() {
		t.Error("note carries no timestamp")
	}
	if body := get(h, "/issues/"+target.ID).Body.String(); !strings.Contains(body, "Handing this back.") {
		t.Errorf("the note is missing from the detail page:\n%s", body)
	}
}

func TestDeleteRemovesTheFileAndRedirectsHome(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Junk"})
	h := newHandler(t, dir)

	// What the post does once the reader has answered the confirmation; the
	// guard itself is TestDeletingAsksFirst.
	res := post(h, "/issues/"+target.ID+"/delete", nil)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	to := res.Header().Get("Location")
	if !strings.HasPrefix(to, "/?") && to != "/" {
		t.Errorf("redirected to %q, want the home view", to)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, ".beaver", "issues", target.ID+"-*.md")); len(matches) != 0 {
		t.Errorf("file survived the delete: %v", matches)
	}
	if body := get(h, to).Body.String(); !strings.Contains(body, target.ID) {
		t.Errorf("home view says nothing about the deleted issue:\n%s", body)
	}
}

// Every refusal the core can hand back comes out as the form again, with the
// core's own words on it, never a blank page and never a 500.
func TestRefusalsRerenderTheFormWith422(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	first := create(t, svc, core.Draft{Title: "Fix flag parsing"})
	second := create(t, svc, core.Draft{Title: "Waits on the fix", DependsOn: []string{first.ID}})
	h := newHandler(t, dir)

	cases := []struct {
		name string
		path string
		form url.Values
		want []string // fragments the re-rendered page must carry
	}{
		{
			name: "empty title on create",
			path: "/issues",
			form: url.Values{"title": {"   "}},
			want: []string{"title", "empty"},
		},
		{
			name: "unknown reference on create",
			path: "/issues",
			form: url.Values{"title": {"New thing"}, "depends_on": {"no-such-issue"}},
			want: []string{"no-such-issue", "New thing"}, // what was typed survives the refusal
		},
		{
			name: "empty title on edit",
			path: "/issues/" + first.ID,
			form: url.Values{"title": {""}},
			want: []string{"title", "empty"},
		},
		{
			name: "dependency cycle on edit",
			path: "/issues/" + first.ID,
			form: url.Values{"title": {first.Title}, "add_depends_on": {second.ID}},
			want: []string{"cycle", first.ID, second.ID},
		},
		{
			name: "empty note",
			path: "/issues/" + first.ID + "/notes",
			form: url.Values{"text": {"  "}},
			want: []string{"note text", "empty"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := post(h, c.path, c.form)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422:\n%s", res.Code, res.Body.String())
			}
			body := res.Body.String()
			for _, want := range c.want {
				if !strings.Contains(body, want) {
					t.Errorf("re-rendered form missing %q:\n%s", want, body)
				}
			}
			if !strings.Contains(body, "<form") {
				t.Errorf("refusal did not come back as a form:\n%s", body)
			}
		})
	}
}

func TestMutatingAnUnknownReferenceIs404(t *testing.T) {
	h := newHandler(t, newStore(t))

	cases := []struct {
		path string
		form url.Values
	}{
		{"/issues/no-such-issue", url.Values{"title": {"Anything"}}},
		{"/issues/no-such-issue/notes", url.Values{"text": {"Anything"}}},
		{"/issues/no-such-issue/delete", nil},
	}
	for _, c := range cases {
		if res := post(h, c.path, c.form); res.Code != http.StatusNotFound {
			t.Errorf("POST %s = %d, want 404", c.path, res.Code)
		}
	}
	if res := get(h, "/issues/no-such-issue/edit"); res.Code != http.StatusNotFound {
		t.Errorf("GET the edit form for an unknown issue = %d, want 404", res.Code)
	}
}

func post(h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(res, r)
	return res
}

// fetch reads an issue back through the core: the file on disk as the next
// reader sees it.
func fetch(t *testing.T, svc *core.Service, ref string) issue.Issue {
	t.Helper()
	got, err := svc.Get(ref)
	if err != nil {
		t.Fatalf("get %s: %v", ref, err)
	}
	return got.Issue
}

// lastCreated finds the issue a create form landed, which the test has no ID
// for until it looks.
func lastCreated(t *testing.T, svc *core.Service, title string) issue.Issue {
	t.Helper()
	listing, err := svc.List(core.Query{Text: title})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, iss := range listing.Issues {
		if iss.Title == title {
			return iss
		}
	}
	t.Fatalf("no issue titled %q was created", title)
	return issue.Issue{}
}

func readIssueFile(t *testing.T, dir, id string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".beaver", "issues", id+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("find file for %s: %v (matched %v)", id, err, matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	return string(raw)
}

// notesSection is the raw text of a file's notes section, heading included:
// the bytes an edit must leave alone.
func notesSection(t *testing.T, file string) string {
	t.Helper()
	at := strings.Index(file, issue.NotesHeading)
	if at < 0 {
		t.Fatalf("file holds no notes section:\n%s", file)
	}
	return file[at:]
}
