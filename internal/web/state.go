package web

// This file holds what a dropped card means: the two routes a drag posts to.
// Neither decides anything about the lifecycle — which move is legal and whose
// claim stands are the core's, and the web never steals — so all that happens
// here is turning a column into a core call and the core's answer into a status
// the board's script can act on.

import (
	"errors"
	"net/http"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// dropTargets are the columns a drop posts to /state, mapped to the state it
// asks for. in-progress is missing on purpose: beginning work also claims the
// issue, so its column posts to /start instead.
var dropTargets = map[string]issue.State{
	string(issue.StateTodo):      issue.StateTodo,
	string(issue.StateDone):      issue.StateDone,
	string(issue.StateCancelled): issue.StateCancelled,
}

// setState moves the issue to the column it was dropped on — the same
// transition the CLI's done, cancel, and reopen make. A drop back on the card's
// own column is the core's idempotent no-op, which writes nothing.
func (s *server) setState(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, err)
		return
	}
	to, ok := dropTargets[r.PostForm.Get("state")]
	if !ok {
		// Not a column anything can be dropped on: the request is malformed
		// rather than refused, so it never reaches the core.
		s.refuse(w, r, http.StatusUnprocessableEntity, "No column called "+r.PostForm.Get("state")+".")
		return
	}
	ref := r.PathValue("ref")
	out, err := svc.Transition(ref, to)
	if err != nil {
		s.failDrop(w, r, ref, out.Warnings, err)
		return
	}
	http.Redirect(w, r, returnTo(r), http.StatusSeeOther)
}

// start claims the issue for the actor the server was launched as and puts it
// in progress — the in-progress column's drop. It never forces: stealing an
// issue another actor holds stays a deliberate act at the CLI.
func (s *server) start(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ref := r.PathValue("ref")
	out, err := svc.Start(ref, s.cfg.Actor, false)
	if err != nil {
		s.failDrop(w, r, ref, out.Warnings, err)
		return
	}
	http.Redirect(w, r, returnTo(r), http.StatusSeeOther)
}

// returnTo is where a state change sends the reader afterwards: the local page
// the form named, or the board — the drag's home — when it named none. Only a
// path of this site's own is followed, so a crafted form cannot bounce a
// reader off to elsewhere.
func returnTo(r *http.Request) string {
	_ = r.ParseForm() // a malformed body is just a form that named nothing
	back := r.PostForm.Get("back")
	if strings.HasPrefix(back, "/") && !strings.HasPrefix(back, "//") {
		return back
	}
	return "/"
}

// failDrop words a refused drop. A move the lifecycle forbids and an issue
// someone else holds are both conflicts with the store's current truth — the
// card goes back where it was and the reader is told why — while anything else
// is the shared failure page.
func (s *server) failDrop(w http.ResponseWriter, r *http.Request, ref string, warnings []core.Warning, err error) {
	var (
		illegal *core.IllegalTransitionError
		claimed *core.ClaimedError
	)
	switch {
	case errors.As(err, &illegal):
		s.refuse(w, r, http.StatusConflict, err.Error()+".")
	case errors.As(err, &claimed):
		// The core states the fact; where to go from here is this interface's to
		// say, since the web deliberately offers no way to steal.
		s.refuse(w, r, http.StatusConflict, err.Error()+" — steal it with `beaver start "+claimed.ID+" --force`.")
	default:
		s.failRef(w, r, ref, warnings, err)
	}
}

// refuse renders a drop the board could not make. It is the shared error page,
// so a post without JavaScript still lands somewhere readable; the board's
// script reads the message out of it to show beside the card that snapped back.
func (s *server) refuse(w http.ResponseWriter, r *http.Request, status int, msg string) {
	s.render(w, r, "error.html", status, errorPage{
		page:    s.page("Not moved", nil),
		Message: msg,
	})
}
