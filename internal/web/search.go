package web

// This file holds the one search box's whole behaviour: which of the two things
// a query means. The decision is the core's, not the interface's — whether the
// text resolves as a reference is exactly what Get answers — so all that
// happens here is turning that answer into an address.

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// search sends the reader wherever their query meant to go: an exact reference
// to that issue, anything else to the list filtered by the text. A query that
// matches nothing is still a filter — an empty list is an answer, not an error —
// so this route never renders a failure of its own.
//
// A reference several issues share redirects to the detail route all the same,
// which is where the choice between them is offered.
func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Redirect(w, r, "/issues", http.StatusSeeOther)
		return
	}
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var ambiguous *core.AmbiguousRefError
	switch got, err := svc.Get(q); {
	case err == nil:
		http.Redirect(w, r, "/issues/"+got.Issue.ID, http.StatusSeeOther)
	case errors.As(err, &ambiguous):
		http.Redirect(w, r, "/issues/"+url.PathEscape(q), http.StatusSeeOther)
	case errors.Is(err, core.ErrNotFound):
		http.Redirect(w, r, "/issues?"+url.Values{"search": {q}}.Encode(), http.StatusSeeOther)
	default:
		s.fail(w, r, err)
	}
}
