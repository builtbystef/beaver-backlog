package web

// This file holds the detail view: one issue as the browser sees it. Nothing
// here derives a relationship or resolves a reference, since the core has
// already done both, so the only work is naming the pieces the template lays
// out.

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// detailPage is everything one issue's file holds, split into the parts the
// page renders separately: the issue's own fields, the description, the notes
// parsed out of the body, the derived relationships, and the frontmatter keys
// Beaver Backlog does not define.
type detailPage struct {
	page
	Issue       issue.Issue
	Description template.HTML
	Notes       []noteView
	Rel         issue.Relationship
	Custom      []customField
	// Moves are the state changes the lifecycle allows from where the issue
	// stands, each a one-button form, so changing state never needs the board.
	Moves []move
	// Note is the box for appending to the log: empty on a page being read,
	// holding the rejected text and the core's words when a note was refused.
	Note noteForm
}

// noteView is one log entry as the page draws it: the attribution, and the
// text rendered as the Markdown it is stored as.
type noteView struct {
	Author string
	Time   time.Time
	Text   template.HTML
}

// move is one legal state change as a form: where it posts, what it posts, and
// the word on the button.
type move struct {
	Label  string
	Action string
	State  issue.State // empty for the start route, which needs no field
}

// customField is one user-defined frontmatter key rendered for a reader.
type customField struct {
	Key   string
	Value string
}

// matchesPage is the disambiguation view: the issues a reference names when it
// names more than one, so a reader can pick the one they meant by ID.
type matchesPage struct {
	page
	Ref     string
	Matches []issue.Issue
}

// detail renders one issue resolved from the reference in the URL. A reference
// that names several issues is not an error page but a choice: the reader picks
// from the matches by ID.
func (s *server) detail(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ref := r.PathValue("ref")
	got, err := svc.Get(ref)
	if err != nil {
		s.failRef(w, r, ref, got.Warnings, err)
		return
	}
	s.renderDetail(w, r, got, noteForm{}, http.StatusOK)
}

// renderDetail draws one issue's page. The note box travels in because a
// refused note is this same page again: the log the reader was writing to, with
// their words still in the box.
func (s *server) renderDetail(w http.ResponseWriter, r *http.Request, got core.Detail, note noteForm, status int) {
	p := s.page(got.Issue.Title, got.Warnings)
	p.Live = true
	p.Section = "issues"
	s.render(w, r, "detail.html", status, detailPage{
		page:        p,
		Issue:       got.Issue,
		Description: description(got.Issue.Body),
		Notes:       noteViews(issue.ParseNotes(got.Issue.Body)),
		Rel:         got.Relationship,
		Custom:      customFields(got.Issue.Custom),
		Moves:       moves(got.Issue),
		Note:        note,
	})
}

// description renders the issue's prose, or nothing when there is none, so the
// template's "no description" placeholder still has an empty value to test.
func description(body string) template.HTML {
	src := issue.Description(body)
	if src == "" {
		return ""
	}
	return renderMarkdown(src)
}

func noteViews(notes []issue.Note) []noteView {
	views := make([]noteView, len(notes))
	for i, n := range notes {
		views[i] = noteView{Author: n.Author, Time: n.Time, Text: renderMarkdown(n.Text)}
	}
	return views
}

// moves lists the state changes the lifecycle allows from the issue's current
// state, the same table Transition enforces, phrased as buttons. Start is the
// odd one out: beginning work also claims the issue, so it posts to its own
// route (and the core, not this list, still has the final word on every move).
func moves(iss issue.Issue) []move {
	base := "/issues/" + iss.ID
	switch iss.State {
	case issue.StateTodo:
		return []move{
			{Label: "Start", Action: base + "/start"},
			{Label: "Done", Action: base + "/state", State: issue.StateDone},
			{Label: "Cancel", Action: base + "/state", State: issue.StateCancelled},
		}
	case issue.StateInProgress:
		return []move{
			{Label: "Done", Action: base + "/state", State: issue.StateDone},
			{Label: "Cancel", Action: base + "/state", State: issue.StateCancelled},
		}
	case issue.StateDone, issue.StateCancelled:
		return []move{{Label: "Reopen", Action: base + "/state", State: issue.StateTodo}}
	}
	return nil // a state outside the lifecycle belongs to doctor
}

// customFields renders the preserved frontmatter keys in the order the YAML
// encoder writes them, so the page and the file agree.
func customFields(m map[string]any) []customField {
	keys := issue.CustomKeys(m)
	fields := make([]customField, len(keys))
	for i, k := range keys {
		fields[i] = customField{Key: k, Value: customValue(m[k])}
	}
	return fields
}

// customValue renders an uninterpreted frontmatter value: scalars plainly, and
// sequences and maps as compact JSON, since a nested value has no layout of its
// own on a page that gives each key one row.
func customValue(v any) string {
	switch v.(type) {
	case nil:
		return ""
	case []any, map[string]any:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return fmt.Sprintf("%v", v)
}
