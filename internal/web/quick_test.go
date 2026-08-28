package web_test

// The quick view as a reader meets it: one issue's key facts drawn over the
// graph rather than in place of it. What is asserted here is the fragment's
// surface — the facts it carries, that it is a fragment and not a page, and the
// way through to the issue — never what blocked or ready mean, which is the
// core's to say.

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The spec's worked example: an issue in todo, priority high, waiting on one
// unfinished dependency.
func TestQuickViewSaysWhatItsIssueIs(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	base := create(t, svc, core.Draft{Title: "Groundwork"})
	target := create(t, svc, core.Draft{
		Title:     "Fix login",
		Priority:  issue.PriorityHigh,
		DependsOn: []string{base.ID},
	})

	res := get(newHandler(t, dir), "/issues/"+target.ID+"/quick")

	if res.Code != http.StatusOK {
		t.Fatalf("GET the quick view = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, want := range []string{"Fix login", target.ID, "todo", "high", "blocked"} {
		if !strings.Contains(body, want) {
			t.Errorf("the quick view does not say %q:\n%s", want, body)
		}
	}
}

func TestQuickViewOfAnUnknownIssueIsNotFound(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Perfectly fine"})

	res := get(newHandler(t, dir), "/issues/no-such-id/quick")

	if res.Code != http.StatusNotFound {
		t.Errorf("GET the quick view of an unknown issue = %d, want 404", res.Code)
	}
}

// A quick view is how an issue is recognised without opening it, so every field
// it promises is either filled in or visibly empty.
func TestQuickViewAnswersEveryFieldItPromises(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	parent := create(t, svc, core.Draft{Title: "The spec"})
	filled := create(t, svc, core.Draft{
		Title:  "Redraw the toolbar",
		Labels: []string{"ui", "chore"},
		Parent: parent.ID,
	})
	start(t, svc, filled.ID)
	bare := create(t, svc, core.Draft{Title: "Groundwork"})
	h := newHandler(t, dir)

	body := get(h, "/issues/"+filled.ID+"/quick").Body.String()

	for _, want := range []string{"redraw-the-toolbar", "tester", "ui", "chore", parent.ID} {
		if !strings.Contains(body, want) {
			t.Errorf("the quick view does not say %q:\n%s", want, body)
		}
	}

	// An issue with none of them says so rather than leaving the reader to
	// guess whether the field was drawn at all.
	empty := get(h, "/issues/"+bare.ID+"/quick").Body.String()
	for _, field := range []string{"Assignee", "Labels", "Parent", "Priority"} {
		if !strings.Contains(empty, field) {
			t.Errorf("the quick view of an issue with no %s does not name the field:\n%s", field, empty)
		}
	}
	if blanks := strings.Count(empty, `class="none"`); blanks != 4 {
		t.Errorf("the quick view marks %d fields as empty, want the 4 the issue does not fill:\n%s", blanks, empty)
	}
}

// The quick view is drawn over a page the browser already has, so it is the
// overlay's own markup and nothing else — the shell around it would be a second
// copy of the whole application.
func TestQuickViewIsAFragmentWithAWayThroughToTheIssue(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix login"})

	body := get(newHandler(t, dir), "/issues/"+target.ID+"/quick").Body.String()

	if link := `href="/issues/` + target.ID + `"`; !strings.Contains(body, link) {
		t.Errorf("the quick view is no way through to the issue (%s):\n%s", link, body)
	}
	for _, chrome := range []string{"<!doctype", "<html", "<aside", "</body>", `href="/graph"`, `href="/doctor"`} {
		if strings.Contains(strings.ToLower(body), chrome) {
			t.Errorf("the quick view carries %s — it is the overlay only:\n%s", chrome, body)
		}
	}
	// The same fragment whether or not the asker sounds like htmx: this address
	// has no whole-document form to fall back to.
	if hxGet(newHandler(t, dir), "/issues/"+target.ID+"/quick").Body.String() != body {
		t.Error("the quick view answers an htmx request with something other than the fragment")
	}
}

// The graph is where the quick view is read, so the picture carries the overlay
// the script fills — empty on arrival, since what it holds is fetched from the
// node the reader clicked.
func TestTheGraphCarriesAnEmptyOverlayForTheQuickView(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	create(t, open(t, dir), core.Draft{Title: "Fix login"})

	body := get(h, "/graph").Body.String()

	if !emptyOverlay.MatchString(body) {
		t.Errorf("the graph has no empty overlay for the quick view to land in:\n%s", body)
	}
	// Nothing to draw is nothing to click, so the overlay goes with the picture.
	if empty := get(newHandler(t, newStore(t)), "/graph").Body.String(); strings.Contains(empty, `id="quick-view"`) {
		t.Errorf("a graph with no nodes still carries the quick view's overlay:\n%s", empty)
	}
}

var emptyOverlay = regexp.MustCompile(`<dialog id="quick-view"[^>]*></dialog>`)
