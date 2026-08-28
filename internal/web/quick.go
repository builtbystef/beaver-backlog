package web

// This file holds the quick view: one issue's key facts as a fragment, drawn
// over the graph so that inspecting an issue costs the reader neither the page
// they are on nor the window they had panned to. Nothing here derives a
// relationship, since the core has already done that, and nothing here renders
// a page: what this address answers is only ever laid over one the browser
// already has.

import (
	"net/http"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// quickView is one issue as the overlay draws it: the issue itself, the slug
// its title reads as, and the relationship the core derived for it.
type quickView struct {
	Issue issue.Issue
	Slug  string
	Rel   issue.Relationship
}

// quick renders one issue's key facts. The graph's nodes hold exact IDs, so a
// reference naming several issues cannot arise here; anything that fails to
// name one issue has no quick view at all.
func (s *server) quick(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	svc, err := s.open()
	if err != nil {
		s.quickMissing(w, ref)
		return
	}
	got, err := svc.Get(ref)
	if err != nil {
		s.quickMissing(w, ref)
		return
	}
	s.renderTemplate(w, "quick.html", "quick", http.StatusOK, quickView{
		Issue: got.Issue,
		Slug:  issue.Slug(got.Issue.Title),
		Rel:   got.Relationship,
	})
}

// quickMissing answers every way this address can fail to name one issue: an
// unknown reference, a reference naming several, a store that has gone. They
// all mean the same thing here, that there is nothing to draw, so they all get
// the contract's 404, worded as a fragment, since an error page laid over the
// graph would be a second copy of the whole application.
func (s *server) quickMissing(w http.ResponseWriter, ref string) {
	s.renderTemplate(w, "quick.html", "quick-missing", http.StatusNotFound, ref)
}
