package web

// The filter bar's one piece of logic: what an address asks the core for. Only
// the mapping is asserted here — what ready, blocked, a label conjunction or the
// unprioritized *mean* is the core's own suite, and this file would only be
// re-asserting it.

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestFiltersFromAnAddressBecomeACoreQuery(t *testing.T) {
	unassigned, tester, spec := "", "tester", "spec-1"
	cases := []struct {
		query string
		want  core.Query
	}{
		{"", core.Query{}},
		{"state=todo&state=done", core.Query{States: []issue.State{issue.StateTodo, issue.StateDone}}},
		{"state=nonsense", core.Query{}},
		{"ready=1", core.Query{Ready: true}},
		{"blocked=1", core.Query{Blocked: true}},
		{"ready=1&blocked=1", core.Query{Ready: true, Blocked: true}},
		// A label field holds a list however it was written: one box of several
		// words, or the repeated parameter a bookmark may carry.
		{"label=spec+web", core.Query{Labels: []string{"spec", "web"}}},
		{"label=spec&label=web", core.Query{Labels: []string{"spec", "web"}}},
		{"priority=urgent&priority=none", core.Query{Priorities: []issue.Priority{issue.PriorityUrgent, ""}}},
		{"priority=nonsense", core.Query{}},
		{"assignee=unassigned", core.Query{Assignee: &unassigned}},
		{"assignee=actor&actor=tester", core.Query{Assignee: &tester}},
		// "Someone" without a name is nobody in particular: still every issue.
		{"assignee=actor&actor=+", core.Query{}},
		{"assignee=any&actor=tester", core.Query{}},
		{"parent=spec-1", core.Query{Parent: &spec}},
		{"parent=+", core.Query{}},
		{"search=+web+", core.Query{Text: "web"}},
		{
			"state=todo&label=spec&search=web",
			core.Query{States: []issue.State{issue.StateTodo}, Labels: []string{"spec"}, Text: "web"},
		},
	}
	for _, c := range cases {
		values, err := url.ParseQuery(c.query)
		if err != nil {
			t.Fatalf("parse %q: %v", c.query, err)
		}
		got := parseFilters(values).query()
		if describe(got) != describe(c.want) {
			t.Errorf("?%s\n got %s\nwant %s", c.query, describe(got), describe(c.want))
		}
	}
}

// A bar built from an address offers back exactly what the address asked for,
// which is what makes a pasted URL reproduce the view it came from.
func TestBarReflectsTheAddressItWasBuiltFrom(t *testing.T) {
	values, err := url.ParseQuery("state=todo&ready=1&label=spec+web&priority=high&assignee=actor&actor=tester&parent=spec-1&search=flag")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bar := parseFilters(values).bar("/issues", nil, "")

	if !on(bar, "state", "todo") || on(bar, "state", "done") {
		t.Errorf("state checkboxes = %v, want only todo checked", toggles(bar, "state"))
	}
	if !on(bar, "ready", "1") || on(bar, "blocked", "1") {
		t.Errorf("condition checkboxes = %v, want ready alone", toggles(bar, "ready"))
	}
	if !on(bar, "priority", "high") {
		t.Errorf("priority checkboxes = %v, want high checked", toggles(bar, "priority"))
	}
	if bar.Labels != "spec web" {
		t.Errorf("labels = %q, want %q", bar.Labels, "spec web")
	}
	if bar.Actor != "tester" || bar.Parent != "spec-1" || bar.Search != "flag" {
		t.Errorf("text fields = %q/%q/%q, want tester/spec-1/flag", bar.Actor, bar.Parent, bar.Search)
	}
	var mode string
	for _, o := range bar.Assignees {
		if o.Selected {
			mode = o.Value
		}
	}
	if mode != assigneeActor {
		t.Errorf("assignee mode = %q, want %q", mode, assigneeActor)
	}
	if !bar.Active {
		t.Error("a filtered address must know it is filtered")
	}
	if bar := parseFilters(url.Values{}).bar("/issues", nil, ""); bar.Active {
		t.Error("an unfiltered address reports itself filtered")
	}
}

// Anything the bar does not own rides along as a hidden field, so filtering
// never drops what another control put in the query string.
func TestBarCarriesTheQueryItDoesNotOwn(t *testing.T) {
	values, err := url.ParseQuery("state=todo&all=done&all=cancelled")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bar := parseFilters(values).bar("/", values, "")

	want := []param{{Name: "all", Value: "done"}, {Name: "all", Value: "cancelled"}}
	if fmt.Sprint(bar.Keep) != fmt.Sprint(want) {
		t.Errorf("carried %v, want %v", bar.Keep, want)
	}
}

// toggles are the bar's checkboxes posting under one name, wherever the menus
// happen to have put them.
func toggles(bar filterBar, name string) []toggle {
	var out []toggle
	for _, m := range bar.Menus {
		for _, t := range m.Toggles {
			if t.Name == name {
				out = append(out, t)
			}
		}
	}
	return out
}

func on(bar filterBar, name, value string) bool {
	for _, t := range toggles(bar, name) {
		if t.Value == value {
			return t.On
		}
	}
	return false
}

// describe renders a query as text, pointers and all, so two queries compare as
// what they select rather than as where their fields happen to live.
func describe(q core.Query) string {
	return fmt.Sprintf("states=%v ready=%v blocked=%v labels=%v priorities=%v assignee=%s parent=%s text=%q",
		q.States, q.Ready, q.Blocked, q.Labels, q.Priorities, deref(q.Assignee), deref(q.Parent), q.Text)
}

func deref(p *string) string {
	if p == nil {
		return "<any>"
	}
	return fmt.Sprintf("%q", *p)
}
