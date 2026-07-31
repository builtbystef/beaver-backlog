package web

// This file holds the board's one piece of presentation logic: turning a
// listing into columns. Nothing here is a rule — column membership is just the
// issue's state and card order is the listing's own, so the only decision the
// board makes for itself is how far back its terminal columns reach.

import (
	"net/url"
	"slices"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// boardStates is the lifecycle the columns run left to right.
var boardStates = []issue.State{issue.StateTodo, issue.StateInProgress, issue.StateDone, issue.StateCancelled}

// boardWindow is how far back a terminal column reaches by default. Closed work
// accumulates forever and stops being news, so done and cancelled show only what
// moved recently; todo and in-progress are never windowed, because unstarted and
// unfinished work is the point of the board however long it has sat there.
const boardWindow = 14 * 24 * time.Hour

// boardPage is the board view's data: the columns, left to right.
type boardPage struct {
	page
	Filters filterBar
	Columns []column
}

// column is one state's stack of cards. Hidden counts what the window is keeping
// out of sight and ShowAllURL is the address that reveals it — both empty for a
// column that is showing everything it has.
type column struct {
	State      issue.State
	Cards      []issue.Issue
	Hidden     int
	ShowAllURL string
}

// columns splits a listing into the board's columns, keeping each issue in the
// order the listing gave it. current is the request's address, which the "show
// all" links extend rather than replace, so a column's escape hatch survives
// whatever else rides the query string.
func columns(issues []issue.Issue, now time.Time, current *url.URL) []column {
	shown := current.Query()["all"]
	cols := make([]column, len(boardStates))
	for i, state := range boardStates {
		cols[i] = column{State: state}
	}
	for _, iss := range issues {
		i := slices.Index(boardStates, iss.State)
		if i < 0 {
			continue // a state outside the lifecycle belongs to doctor, not the board
		}
		col := &cols[i]
		if windowed(iss.State) && !slices.Contains(shown, string(iss.State)) && now.Sub(iss.Updated) > boardWindow {
			col.Hidden++
			continue
		}
		col.Cards = append(col.Cards, iss)
	}
	for i := range cols {
		if cols[i].Hidden > 0 {
			cols[i].ShowAllURL = showAll(current, cols[i].State)
		}
	}
	return cols
}

// windowed reports whether a column shows only recent work: the two terminal
// states, and only those.
func windowed(state issue.State) bool {
	return state == issue.StateDone || state == issue.StateCancelled
}

// showAll is current with state added to the columns asked for in full.
func showAll(current *url.URL, state issue.State) string {
	next := *current
	q := next.Query()
	q["all"] = append(slices.Clone(q["all"]), string(state))
	next.RawQuery = q.Encode()
	return next.RequestURI()
}
