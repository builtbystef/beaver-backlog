package web

// This file holds the detail view: one issue as the browser sees it. Nothing
// here derives a relationship or resolves a reference — the core has already
// done both — so the only work is naming the pieces the template lays out.

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	Description string
	Notes       []issue.Note
	Rel         issue.Relationship
	Custom      []customField
	// Note is the box for appending to the log: empty on a page being read,
	// holding the rejected text and the core's words when a note was refused.
	Note noteForm
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
// refused note is this same page again — the log the reader was writing to, with
// their words still in the box.
func (s *server) renderDetail(w http.ResponseWriter, r *http.Request, got core.Detail, note noteForm, status int) {
	p := s.page(got.Issue.Title, got.Warnings)
	p.Live = true
	s.render(w, r, "detail.html", status, detailPage{
		page:        p,
		Issue:       got.Issue,
		Description: issue.Description(got.Issue.Body),
		Notes:       issue.ParseNotes(got.Issue.Body),
		Rel:         got.Relationship,
		Custom:      customFields(got.Issue.Custom),
		Note:        note,
	})
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
