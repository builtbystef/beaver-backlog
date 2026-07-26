package core

// This file holds the lifecycle rules: which state changes are legal, which are
// idempotent no-ops, and what beginning work does beyond moving the state.

import (
	"errors"
	"fmt"
	"slices"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// ErrNotATransition means a state is not one Transition can move an issue to.
// in-progress is the only such state: beginning work also claims the issue and
// reports what it waits on, so it belongs to Start.
var ErrNotATransition = errors.New("not a transition target")

// IllegalTransitionError reports a state change the issue's current state
// forbids — closing an already-closed issue, or starting one. It carries the
// issue and both ends of the refused move, so a caller can phrase the refusal
// in its own words.
type IllegalTransitionError struct {
	ID   string      // the issue that was not moved
	From issue.State // the state it is in
	To   issue.State // the state the caller asked for
}

func (e *IllegalTransitionError) Error() string {
	return fmt.Sprintf("%s is %s and cannot become %s", e.ID, e.From, e.To)
}

// enterFrom is the legality table: for each state a transition may set, the
// states an issue may enter it from. Closing works from the two open states,
// and reopening restores either closed state; everything else is refused.
var enterFrom = map[issue.State][]issue.State{
	issue.StateDone:      {issue.StateTodo, issue.StateInProgress},
	issue.StateCancelled: {issue.StateTodo, issue.StateInProgress},
	issue.StateTodo:      {issue.StateDone, issue.StateCancelled},
}

// Transition moves the issue ref names to the state to. An issue already in
// that state is an idempotent no-op — reported with Changed false, with nothing
// written — and a move the current state forbids is an *IllegalTransitionError
// that never touches the file. to must be a state a transition can set;
// in-progress yields ErrNotATransition, since starting work is Start's.
func (s *Service) Transition(ref string, to issue.State) (Outcome, error) {
	sources, ok := enterFrom[to]
	if !ok {
		return Outcome{}, fmt.Errorf("%w: %s", ErrNotATransition, to)
	}
	snap, warnings, err := s.scan()
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	iss, path, err := resolve(snap, ref)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}

	if iss.State == to {
		return Outcome{Issue: iss, Previous: iss, Warnings: warnings}, nil
	}
	if !slices.Contains(sources, iss.State) {
		return Outcome{Warnings: warnings}, &IllegalTransitionError{ID: iss.ID, From: iss.State, To: to}
	}

	previous := iss
	iss.State = to
	iss, err = s.write(path, iss)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	return Outcome{Issue: iss, Previous: previous, Changed: true, Warnings: warnings}, nil
}

// ClaimedError reports that the ownership guard refused an issue another actor
// holds. It is advisory coordination, not a lock: a forced call steals the
// issue, and concurrent claims on two branches surface as a merge conflict on
// the `assignee:` line rather than as silent double-ownership.
type ClaimedError struct {
	ID string // the issue that was not started
	By string // the actor holding it
}

func (e *ClaimedError) Error() string {
	return fmt.Sprintf("%s is claimed by %s", e.ID, e.By)
}

// StartOutcome is Start's result. Beyond the write itself — and the before-and-
// after pair every outcome carries — it holds the relationship view of the issue
// now in progress and the dependencies work began in spite of.
type StartOutcome struct {
	Outcome
	Relationship      issue.Relationship // the derived view of the started issue
	UnmetDependencies []issue.Blocker    // dependencies unmet when work began; empty unless this call began it
}

// Start begins work on the issue ref names: it moves the issue to in-progress
// and makes actor its assignee, auto-claiming an unowned issue. An issue
// another actor holds is refused with a *ClaimedError unless force steals it,
// and a closed issue is refused with an *IllegalTransitionError — it must be
// reopened first. Starting an issue that is already the actor's and already in
// progress writes nothing.
//
// Unmet dependencies never refuse a start: beginning blocked work is sometimes
// the right call, so they come back as data for the caller to surface.
func (s *Service) Start(ref, actor string, force bool) (StartOutcome, error) {
	snap, warnings, err := s.scan()
	if err != nil {
		return startFailure(warnings, err)
	}
	iss, path, err := resolve(snap, ref)
	if err != nil {
		return startFailure(warnings, err)
	}

	if iss.State == issue.StateDone || iss.State == issue.StateCancelled {
		return startFailure(warnings, &IllegalTransitionError{ID: iss.ID, From: iss.State, To: issue.StateInProgress})
	}
	if iss.Assignee != "" && iss.Assignee != actor && !force {
		return startFailure(warnings, &ClaimedError{ID: iss.ID, By: iss.Assignee})
	}

	// One scan serves the reference, the unmet dependencies, and the
	// relationship view: starting changes only this issue's own state, never a
	// dependency's, so the pre-write graph still describes the started issue.
	rel := issue.NewRelations(snap.Issues())

	// The dependencies are reported only when work actually begins — an issue
	// already in progress had its turn when it started.
	var unmet []issue.Blocker
	if iss.State == issue.StateTodo {
		unmet = rel.BlockedOn(iss)
	}

	previous := iss
	iss.State = issue.StateInProgress
	iss.Assignee = actor
	changed := iss.State != previous.State || iss.Assignee != previous.Assignee
	if changed {
		if iss, err = s.write(path, iss); err != nil {
			return startFailure(warnings, err)
		}
	}
	return StartOutcome{
		Outcome:           Outcome{Issue: iss, Previous: previous, Changed: changed, Warnings: warnings},
		Relationship:      rel.For(iss),
		UnmetDependencies: unmet,
	}, nil
}

// startFailure carries a scan's warnings out alongside an error, so a caller
// that started nothing still learns which files were skipped.
func startFailure(warnings []Warning, err error) (StartOutcome, error) {
	return StartOutcome{Outcome: Outcome{Warnings: warnings}}, err
}
