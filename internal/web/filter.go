package web

// This file holds the filter bar the board and the list share: the translation
// between an address and a core query, and back into the controls that produced
// it. Nothing here decides what a filter means — ready, blocked, a conjunction
// of labels, the unprioritized — every one of those is the core's, and each
// field below lands in a core.Query untouched. Both directions live together
// because they are one contract: a view is only bookmarkable while the address
// the bar writes is the address the bar can read back.

import (
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The three things the assignee control can ask for. Unassigned is a filter of
// its own rather than an empty name, because "nobody holds this" and "anybody
// may" are different questions and a text box cannot tell them apart — hence a
// mode beside the name rather than one overloaded field.
const (
	assigneeAny        = "any"
	assigneeUnassigned = "unassigned"
	assigneeActor      = "actor"
)

// filterParams are the query parameters the bar owns. Anything else on the
// address belongs to another control and is carried through untouched.
var filterParams = []string{"state", "ready", "blocked", "label", "priority", "assignee", "actor", "parent", "search"}

// filters is one address's filter state. It holds what the reader asked for,
// not what the core makes of it: an unresolvable parent or a priority nobody
// spells is still what the box says, so the bar can offer it back for fixing.
type filters struct {
	States       []issue.State
	Ready        bool
	Blocked      bool
	Labels       []string
	Priorities   []issue.Priority
	AssigneeMode string // one of the three assignee constants
	Actor        string // the name, when the mode asks for one
	Parent       string
	Search       string
}

// parseFilters reads the filter state out of a query string. A value the bar
// could not have produced — a state outside the lifecycle, a priority that is
// no level — is dropped rather than refused: an address is not a form, and a
// stale bookmark should still draw a page.
func parseFilters(v url.Values) filters {
	f := filters{
		Ready:        checked(v.Get("ready")),
		Blocked:      checked(v.Get("blocked")),
		AssigneeMode: strings.TrimSpace(v.Get("assignee")),
		Actor:        strings.TrimSpace(v.Get("actor")),
		Parent:       strings.TrimSpace(v.Get("parent")),
		Search:       strings.TrimSpace(v.Get("search")),
	}
	if f.AssigneeMode != assigneeUnassigned && f.AssigneeMode != assigneeActor {
		f.AssigneeMode = assigneeAny
	}
	// Each of the three list filters reads the same two ways: the repeated
	// parameter a checkbox group posts, and the several values one text box
	// holds — so a hand-written address works however it was written.
	for _, raw := range v["state"] {
		for _, one := range values(raw) {
			if state := issue.State(one); slices.Contains(boardStates, state) {
				f.States = append(f.States, state)
			}
		}
	}
	for _, raw := range v["priority"] {
		for _, one := range values(raw) {
			if level, err := core.ParsePriority(one); err == nil {
				f.Priorities = append(f.Priorities, level)
			}
		}
	}
	for _, raw := range v["label"] {
		f.Labels = append(f.Labels, values(raw)...)
	}
	return f
}

// query is the core query this address asks for — the only place the bar's
// state turns into a selection, so both views select identically.
func (f filters) query() core.Query {
	q := core.Query{
		States:     f.States,
		Ready:      f.Ready,
		Blocked:    f.Blocked,
		Labels:     f.Labels,
		Priorities: f.Priorities,
		Text:       f.Search,
	}
	switch {
	case f.AssigneeMode == assigneeUnassigned:
		unassigned := ""
		q.Assignee = &unassigned
	case f.AssigneeMode == assigneeActor && f.Actor != "":
		actor := f.Actor
		q.Assignee = &actor
	}
	if f.Parent != "" {
		parent := f.Parent
		q.Parent = &parent
	}
	return q
}

// active reports whether this address narrows anything, which is what tells an
// empty view "nothing matches" from "nothing here yet".
func (f filters) active() bool {
	return len(f.States) > 0 || f.Ready || f.Blocked || len(f.Labels) > 0 || len(f.Priorities) > 0 ||
		f.AssigneeMode != assigneeAny || f.Parent != "" || f.Search != ""
}

// filterBar is the bar as a template draws it: every control already marked
// against the address, so the markup compares nothing for itself.
type filterBar struct {
	Action     string // where the form goes without JavaScript: the view it filters
	States     []toggle
	Ready      bool
	Blocked    bool
	Labels     string // the label conjunction as one field of words
	Priorities []toggle
	Assignees  []option
	Actor      string
	Parent     string
	Search     string
	Keep       []param // the query the bar does not own, carried through
	Active     bool
	// Refused is the core's own words about a reference the bar carries that
	// names no issue, said beside the box that holds it.
	Refused string
}

// toggle is one checkbox: what it posts and whether the address has it on.
type toggle struct {
	Value string
	Label string
	On    bool
}

// param is one query parameter travelling through the form as a hidden field.
type param struct {
	Name  string
	Value string
}

// bar builds the controls for this address. current is the whole query string
// the request arrived with — everything in it the bar does not own comes back as
// a hidden field, so submitting a filter never drops a neighbouring control's
// state, such as the board's column shown in full.
func (f filters) bar(action string, current url.Values, refused string) filterBar {
	b := filterBar{
		Action:     action,
		Ready:      f.Ready,
		Blocked:    f.Blocked,
		Labels:     strings.Join(f.Labels, " "),
		Priorities: priorityToggles(f.Priorities),
		Assignees:  assigneeOptions(f.AssigneeMode),
		Actor:      f.Actor,
		Parent:     f.Parent,
		Search:     f.Search,
		Keep:       carried(current),
		Active:     f.active(),
		Refused:    refused,
	}
	for _, state := range boardStates {
		b.States = append(b.States, toggle{
			Value: string(state),
			Label: string(state),
			On:    slices.Contains(f.States, state),
		})
	}
	return b
}

// priorityToggles is the priority group: the four levels and the unprioritized
// one, which the core spells "none" wherever it is written as text.
func priorityToggles(on []issue.Priority) []toggle {
	toggles := make([]toggle, len(priorityLevels))
	for i, level := range priorityLevels {
		selected, err := core.ParsePriority(level)
		label := level
		if level == "none" {
			label = "unprioritized"
		}
		toggles[i] = toggle{Value: level, Label: label, On: err == nil && slices.Contains(on, selected)}
	}
	return toggles
}

// assigneeOptions is the assignee select, marked against the current mode.
func assigneeOptions(mode string) []option {
	modes := []option{
		{Value: assigneeAny, Label: "anyone"},
		{Value: assigneeUnassigned, Label: "unassigned"},
		{Value: assigneeActor, Label: "actor…"},
	}
	for i := range modes {
		modes[i].Selected = modes[i].Value == mode
	}
	return modes
}

// carried is everything on the address the bar does not own, in the order the
// parameters were named, so the form reproduces the query string it came from.
func carried(current url.Values) []param {
	var kept []param
	for _, name := range slices.Sorted(maps.Keys(current)) {
		if slices.Contains(filterParams, name) {
			continue
		}
		for _, value := range current[name] {
			kept = append(kept, param{Name: name, Value: value})
		}
	}
	return kept
}

// checked reads a checkbox's parameter. A browser posts "on" and a hand-written
// address might say anything; only an absent or empty value is off.
func checked(value string) bool {
	return strings.TrimSpace(value) != ""
}
