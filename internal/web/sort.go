package web

// This file holds the list's ordering: the translation between the address's
// sort parameters and a rearranged listing, and back into the column headers
// that produced it. Ordering is presentation, not a rule — the core's listing
// order stays the default, and a sorted view is the same issues told to stand
// differently.

import (
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// order is one address's sort state: which column, and which way. The zero
// order means the core's own listing order, untouched.
type order struct {
	Key  string
	Desc bool
}

// listColumns are the table's columns left to right. A column without a key
// offers no ordering — labels are a set, and a set has no first.
var listColumns = []struct {
	Key   string
	Label string
	// TimeFirst marks a column whose first click sorts newest first, because
	// nobody asks a time column for the oldest.
	TimeFirst bool
}{
	{Key: "id", Label: "ID"},
	{Key: "title", Label: "Title"},
	{Key: "state", Label: "State"},
	{Key: "priority", Label: "Priority"},
	{Key: "", Label: "Labels"},
	{Key: "assignee", Label: "Assignee"},
	{Key: "updated", Label: "Updated", TimeFirst: true},
}

// header is one column heading as the template draws it: the label, the
// address that sorts by it, and the direction mark when this column is the one
// the address already sorts by.
type header struct {
	Label string
	URL   string // empty for a column that offers no ordering
	Mark  string
}

// parseOrder reads the sort state off the address. A key naming no column is
// dropped rather than refused — an address is not a form, and a stale bookmark
// should still draw the list (in the core's order).
func parseOrder(v url.Values) order {
	key := strings.TrimSpace(v.Get("sort"))
	for _, col := range listColumns {
		if col.Key != "" && col.Key == key {
			return order{Key: key, Desc: v.Get("dir") == "desc"}
		}
	}
	return order{}
}

// apply rearranges issues in place. The sort is stable, so issues equal under
// the chosen column keep the core's order between them — ties break the same
// way on the same store.
func (o order) apply(issues []issue.Issue) {
	if o.Key == "" {
		return
	}
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if o.Desc {
			a, b = b, a
		}
		switch o.Key {
		case "id":
			return a.ID < b.ID
		case "title":
			return strings.ToLower(a.Title) < strings.ToLower(b.Title)
		case "state":
			return stateRank(a.State) < stateRank(b.State)
		case "priority":
			return priorityRank(a.Priority) < priorityRank(b.Priority)
		case "assignee":
			return assigneeKey(a.Assignee) < assigneeKey(b.Assignee)
		case "updated":
			return a.Updated.Before(b.Updated)
		}
		return false
	})
}

// headers builds the column headings for this address, each sorting link
// carrying the whole query it arrived with, so sorting never drops a filter.
func (o order) headers(current *url.URL) []header {
	out := make([]header, len(listColumns))
	for i, col := range listColumns {
		h := header{Label: col.Label}
		if col.Key != "" {
			h.URL = o.sortURL(current, col.Key, col.TimeFirst)
			if o.Key == col.Key {
				if o.Desc {
					h.Mark = "▼"
				} else {
					h.Mark = "▲"
				}
			}
		}
		out[i] = h
	}
	return out
}

// sortURL is current re-addressed to sort by key: a fresh column starts in its
// natural direction, and the column already sorted flips.
func (o order) sortURL(current *url.URL, key string, timeFirst bool) string {
	desc := timeFirst
	if o.Key == key {
		desc = !o.Desc
	}
	next := *current
	q := next.Query()
	q.Set("sort", key)
	if desc {
		q.Set("dir", "desc")
	} else {
		q.Del("dir")
	}
	next.RawQuery = q.Encode()
	return next.RequestURI()
}

// stateRank orders states the way the board's columns run, with anything
// outside the lifecycle last.
func stateRank(s issue.State) int {
	if i := slices.Index(boardStates, s); i >= 0 {
		return i
	}
	return len(boardStates)
}

// priorityRank orders priorities urgent first, the unprioritized last —
// "sort by priority" means the pressing work on top.
func priorityRank(p issue.Priority) int {
	switch p {
	case issue.PriorityUrgent:
		return 0
	case issue.PriorityHigh:
		return 1
	case issue.PriorityMedium:
		return 2
	case issue.PriorityLow:
		return 3
	}
	return 4
}

// assigneeKey sorts the unassigned after every name rather than before the
// alphabet, since a blank is an absence, not a small name.
func assigneeKey(a string) string {
	if a == "" {
		return "￿"
	}
	return strings.ToLower(a)
}
