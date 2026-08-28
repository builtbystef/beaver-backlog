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

// filterBar is the toolbar as a template draws it: every control already marked
// against the address, so the markup compares nothing for itself.
type filterBar struct {
	Action string // the view the toolbar narrows
	// ClearURL is that view with every filter off — but still carrying what the
	// bar does not own, because clearing filters is not the same as putting a
	// column the reader opened in full back behind its window.
	ClearURL string
	// Menus are the toggle groups the toolbar keeps behind a button each, so a
	// vocabulary this wide still reads as one bar.
	Menus     []menu
	Assignees []option
	Actor     string
	Labels    string // the label conjunction as one field of words
	Parent    string
	Search    string
	Keep      []param // the query the bar does not own, carried through
	Active    bool
	// Chips are the active filters said one by one beside the controls, each
	// with the address that takes just that one off — the current view minus one
	// filter, so narrowing is visible and undoable in a click.
	Chips []chip
	// Refused is the core's own words about a reference the bar carries that
	// names no issue, said beside the box that holds it.
	Refused string
}

// menu is one group of toggles behind a button: what the button says, and the
// checkboxes it opens onto. Whether the button reads as active is the live
// state of the boxes inside it, which is the stylesheet's to see — a count
// rendered here would go stale the moment a box was ticked, since narrowing the
// view redraws the listing and not the controls that asked for it.
type menu struct {
	Label   string
	Toggles []toggle
}

// chip is one active filter as the toolbar shows it.
type chip struct {
	Label     string
	RemoveURL string
}

// toggle is one checkbox: what it posts, under which name, and whether the
// address has it on.
type toggle struct {
	Name  string
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
	return filterBar{
		Action:   action,
		ClearURL: unfiltered(action, current),
		Menus: []menu{
			{Label: "State", Toggles: stateToggles(f.States)},
			{Label: "Condition", Toggles: conditionToggles(f.Ready, f.Blocked)},
			{Label: "Priority", Toggles: priorityToggles(f.Priorities)},
		},
		Assignees: assigneeOptions(f.AssigneeMode),
		Actor:     f.Actor,
		Labels:    strings.Join(f.Labels, " "),
		Parent:    f.Parent,
		Search:    f.Search,
		Keep:      carried(current),
		Active:    f.active(),
		Chips:     f.chips(action, current),
		Refused:   refused,
	}
}

// unfiltered is the view with nothing narrowing it, keeping every parameter the
// bar does not own.
func unfiltered(action string, current url.Values) string {
	v := url.Values{}
	for _, p := range carried(current) {
		v.Add(p.Name, p.Value)
	}
	if q := v.Encode(); q != "" {
		return action + "?" + q
	}
	return action
}

// stateToggles is the lifecycle group: the same four states the board columns
// are, in the same order.
func stateToggles(on []issue.State) []toggle {
	toggles := make([]toggle, len(boardStates))
	for i, state := range boardStates {
		toggles[i] = toggle{
			Name:  "state",
			Value: string(state),
			Label: string(state),
			On:    slices.Contains(on, state),
		}
	}
	return toggles
}

// conditionToggles is the derived-condition group. Each is a filter of its own
// rather than a value of one, which is why the two carry different names.
func conditionToggles(ready, blocked bool) []toggle {
	return []toggle{
		{Name: "ready", Value: "1", Label: "ready", On: ready},
		{Name: "blocked", Value: "1", Label: "blocked", On: blocked},
	}
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
		toggles[i] = toggle{Name: "priority", Value: level, Label: label, On: err == nil && slices.Contains(on, selected)}
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

// chips words each active filter, pairing it with the address that removes it
// alone. Every removal address is built from a fresh encoding of the bar's
// state, mutated for that one chip, with the carried parameters put back — the
// same round trip the form itself makes.
func (f filters) chips(action string, current url.Values) []chip {
	var out []chip
	add := func(label string, mutate func(url.Values)) {
		out = append(out, chip{Label: label, RemoveURL: f.removeURL(action, current, mutate)})
	}
	for _, state := range f.States {
		s := string(state)
		add(s, func(v url.Values) { v["state"] = drop(v["state"], s) })
	}
	if f.Ready {
		add("ready", func(v url.Values) { v.Del("ready") })
	}
	if f.Blocked {
		add("blocked", func(v url.Values) { v.Del("blocked") })
	}
	for _, level := range f.Priorities {
		p := priorityValue(level)
		add("priority: "+p, func(v url.Values) { v["priority"] = drop(v["priority"], p) })
	}
	switch f.AssigneeMode {
	case assigneeUnassigned:
		add("unassigned", func(v url.Values) { v.Del("assignee") })
	case assigneeActor:
		label := "assignee: …"
		if f.Actor != "" {
			label = "assignee: " + f.Actor
		}
		add(label, func(v url.Values) { v.Del("assignee"); v.Del("actor") })
	}
	for _, l := range f.Labels {
		label := l
		add("label: "+label, func(v url.Values) { v["label"] = drop(v["label"], label) })
	}
	if f.Parent != "" {
		add("parent: "+f.Parent, func(v url.Values) { v.Del("parent") })
	}
	if f.Search != "" {
		add("text: "+f.Search, func(v url.Values) { v.Del("search") })
	}
	return out
}

// removeURL is the current address re-written without one filter: the bar's
// canonical encoding, mutated, with everything the bar does not own put back.
func (f filters) removeURL(action string, current url.Values, mutate func(url.Values)) string {
	v := f.encode()
	mutate(v)
	for _, p := range carried(current) {
		v.Add(p.Name, p.Value)
	}
	// A key mutated down to no values must go, or Encode writes it as empty.
	for name, values := range v {
		if len(values) == 0 {
			delete(v, name)
		}
	}
	if q := v.Encode(); q != "" {
		return action + "?" + q
	}
	return action
}

// encode is the bar's state as the canonical query string it would submit —
// the inverse of parseFilters, one parameter per value.
func (f filters) encode() url.Values {
	v := url.Values{}
	for _, s := range f.States {
		v.Add("state", string(s))
	}
	if f.Ready {
		v.Set("ready", "1")
	}
	if f.Blocked {
		v.Set("blocked", "1")
	}
	for _, level := range f.Priorities {
		v.Add("priority", priorityValue(level))
	}
	switch f.AssigneeMode {
	case assigneeUnassigned:
		v.Set("assignee", assigneeUnassigned)
	case assigneeActor:
		v.Set("assignee", assigneeActor)
		if f.Actor != "" {
			v.Set("actor", f.Actor)
		}
	}
	for _, l := range f.Labels {
		v.Add("label", l)
	}
	if f.Parent != "" {
		v.Set("parent", f.Parent)
	}
	if f.Search != "" {
		v.Set("search", f.Search)
	}
	return v
}

// priorityValue spells a priority the way an address does, where the
// unprioritized level is written "none".
func priorityValue(p issue.Priority) string {
	if p == "" {
		return "none"
	}
	return string(p)
}

// drop is list without the first occurrence of value.
func drop(list []string, value string) []string {
	var out []string
	removed := false
	for _, v := range list {
		if !removed && v == value {
			removed = true
			continue
		}
		out = append(out, v)
	}
	return out
}
