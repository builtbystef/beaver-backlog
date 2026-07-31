package web

// This file holds the detail view: one issue as the browser sees it. Nothing
// here derives a relationship or resolves a reference — the core has already
// done both — so the only work is naming the pieces the template lays out.

import (
	"encoding/json"
	"errors"
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
	var ambiguous *core.AmbiguousRefError
	switch {
	case errors.As(err, &ambiguous):
		s.render(w, r, "matches.html", http.StatusOK, matchesPage{
			page:    s.page(ref, got.Warnings),
			Ref:     ref,
			Matches: ambiguous.Matches,
		})
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	s.render(w, r, "detail.html", http.StatusOK, detailPage{
		page:        s.page(got.Issue.Title, got.Warnings),
		Issue:       got.Issue,
		Description: issue.Description(got.Issue.Body),
		Notes:       issue.ParseNotes(got.Issue.Body),
		Rel:         got.Relationship,
		Custom:      customFields(got.Issue.Custom),
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
