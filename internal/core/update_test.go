package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// One call reaches every field an update can touch, so a caller changing six
// things at once pays for one write and one `updated` bump.
func TestUpdateSetsEveryField(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("dep111", "A prerequisite"))
	seed(t, root, mkIssue("par222", "The epic"))
	seed(t, root, mkIssue("iss001", "Old title"))

	out, err := openAt(t, root).Update("iss001", core.Changes{
		Title:        new("New title"),
		Body:         new("The description."),
		Assignee:     new("alice"),
		Priority:     new(issue.PriorityHigh),
		AddLabels:    []string{"docs", "chore"},
		AddDependsOn: []string{"a-prerequisite"}, // any reference form Get accepts
		Parent:       new("par222"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !out.Changed {
		t.Error("Changed = false, want true")
	}
	got := out.Issue
	if got.ID != "iss001" {
		t.Errorf("id = %q, want iss001: an update never re-identifies an issue", got.ID)
	}
	if got.Title != "New title" || got.Body != "The description." {
		t.Errorf("title/body = %q/%q, want the new ones", got.Title, got.Body)
	}
	if got.Assignee != "alice" || got.Priority != issue.PriorityHigh {
		t.Errorf("assignee/priority = %q/%q, want alice/high", got.Assignee, got.Priority)
	}
	if !slices.Equal(got.Labels, []string{"docs", "chore"}) {
		t.Errorf("labels = %v, want [docs chore]", got.Labels)
	}
	// The reference was a slug; what is stored is the id it names.
	if !slices.Equal(got.DependsOn, []string{"dep111"}) {
		t.Errorf("depends_on = %v, want [dep111]", got.DependsOn)
	}
	if got.Parent != "par222" {
		t.Errorf("parent = %q, want par222", got.Parent)
	}
	if !got.Updated.Equal(writeTime) {
		t.Errorf("updated = %s, want it bumped to %s", got.Updated, writeTime)
	}
	if !out.Previous.Updated.Equal(fixedTime) || out.Previous.Title != "Old title" {
		t.Errorf("previous = %+v, want the issue as it stood before", out.Previous)
	}

	detail, err := open(t, root).Get("iss001")
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if detail.Issue.Title != "New title" || detail.Issue.Assignee != "alice" {
		t.Errorf("persisted = %q/%q, want the new title and assignee", detail.Issue.Title, detail.Issue.Assignee)
	}
}

// A nil pointer is not an empty value: a caller that names one field says
// nothing about the rest.
func TestUpdateLeavesUnnamedFieldsAlone(t *testing.T) {
	root := newStore(t)
	seeded := withLabels(withPriority(withAssignee(withParent(
		withDeps(withBody(mkIssue("iss001", "Keep me"), "Body text."), "dep111"),
		"par222"), "bob"), issue.PriorityLow), "keep")
	seed(t, root, seeded)

	out, err := openAt(t, root).Update("iss001", core.Changes{Assignee: new("alice")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := out.Issue
	if got.Title != "Keep me" || got.Body != out.Previous.Body || got.Priority != issue.PriorityLow {
		t.Errorf("title/body/priority = %q/%q/%q, want them untouched", got.Title, got.Body, got.Priority)
	}
	if !slices.Equal(got.Labels, []string{"keep"}) || !slices.Equal(got.DependsOn, []string{"dep111"}) {
		t.Errorf("labels/depends_on = %v/%v, want them untouched", got.Labels, got.DependsOn)
	}
	if got.Parent != "par222" || got.Assignee != "alice" {
		t.Errorf("parent/assignee = %q/%q, want par222/alice", got.Parent, got.Assignee)
	}
}

// The empty string is how a caller says "no one" and "no parent"; the same
// pointer set to a value would have meant the opposite.
func TestUpdateClearsWithAnEmptyValue(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("par222", "The epic"))
	seed(t, root, withAssignee(withParent(mkIssue("iss001", "Owned child"), "par222"), "bob"))

	out, err := openAt(t, root).Update("iss001", core.Changes{
		Assignee: new(""),
		Parent:   new(""),
		Priority: new(issue.Priority("")),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Issue.Assignee != "" || out.Issue.Parent != "" || out.Issue.Priority != "" {
		t.Errorf("assignee/parent/priority = %q/%q/%q, want all cleared",
			out.Issue.Assignee, out.Issue.Parent, out.Issue.Priority)
	}
	if out.Previous.Assignee != "bob" {
		t.Errorf("previous assignee = %q, want bob", out.Previous.Assignee)
	}
}

// Labels and dependencies are sets, not lists: a caller adds and removes
// entries without having to know, or resend, the ones it does not name, and a
// value in both sets loses.
func TestUpdateAppliesSetsWithRemovalWinning(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("dep111", "First prerequisite"))
	seed(t, root, mkIssue("dep222", "Second prerequisite"))
	seed(t, root, withDeps(withLabels(mkIssue("iss001", "Some work"), "docs", "chore"), "dep111"))

	out, err := openAt(t, root).Update("iss001", core.Changes{
		AddLabels:       []string{"bug", "chore", "docs"}, // chore and docs are already there
		RemoveLabels:    []string{"docs", "bug"},          // removal wins over the add of bug
		AddDependsOn:    []string{"dep222", "second-prerequisite"},
		RemoveDependsOn: []string{"dep111"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !slices.Equal(out.Issue.Labels, []string{"chore"}) {
		t.Errorf("labels = %v, want [chore]: docs removed, bug removed despite the add", out.Issue.Labels)
	}
	// Two references to one issue collapse into a single edge.
	if !slices.Equal(out.Issue.DependsOn, []string{"dep222"}) {
		t.Errorf("depends_on = %v, want [dep222]", out.Issue.DependsOn)
	}
}

// A dangling edge names an issue no scan can find, so a removal takes the
// reference as written when nothing answers it. Otherwise the only way out of
// a dependency on a deleted issue would be to hand-edit the file.
func TestUpdateRemovesADanglingDependency(t *testing.T) {
	root := newStore(t)
	seed(t, root, withDeps(mkIssue("iss001", "Some work"), "gone99"))

	out, err := openAt(t, root).Update("iss001", core.Changes{RemoveDependsOn: []string{"gone99"}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(out.Issue.DependsOn) != 0 {
		t.Errorf("depends_on = %v, want the dangling edge dropped", out.Issue.DependsOn)
	}
}

// An addition is strict where a removal is forgiving: a reference that names no
// issue refuses the whole update, so a typo never persists as a dangling edge.
func TestUpdateRefusesAnUnresolvableReference(t *testing.T) {
	cases := map[string]core.Changes{
		"depends_on": {AddDependsOn: []string{"gone99"}},
		"parent":     {Parent: new("gone99")},
	}
	for field, changes := range cases {
		t.Run(field, func(t *testing.T) {
			root := newStore(t)
			seed(t, root, mkIssue("iss001", "Some work"))
			before := fileOf(t, root, "iss001", "Some work")

			_, err := openAt(t, root).Update("iss001", changes)
			var unknown *core.UnknownRefError
			if !errors.As(err, &unknown) {
				t.Fatalf("Update with an unknown %s = %v, want *UnknownRefError", field, err)
			}
			if unknown.Ref != "gone99" {
				t.Errorf("Ref = %q, want the reference at fault", unknown.Ref)
			}
			if after := fileOf(t, root, "iss001", "Some work"); after != before {
				t.Error("a refused update rewrote the file")
			}
		})
	}
}

// The worked example of the net-change rule: adding and removing the same label
// on an unlabelled issue asks for nothing, so nothing is written and `updated`
// keeps the moment of the last real edit.
func TestUpdateWithNoNetChangeWritesNothing(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Some work"))
	before := fileOf(t, root, "iss001", "Some work")

	out, err := openAt(t, root).Update("iss001", core.Changes{
		AddLabels:    []string{"a"},
		RemoveLabels: []string{"a"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Changed {
		t.Error("Changed = true, want false: the adds and removes cancelled out")
	}
	if !out.Issue.Updated.Equal(fixedTime) {
		t.Errorf("updated = %s, want unchanged %s (no bump on a no-op)", out.Issue.Updated, fixedTime)
	}
	if after := fileOf(t, root, "iss001", "Some work"); after != before {
		t.Errorf("a no-op update rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// Setting a field to what it already holds is the same no-op, whichever field
// it is.
func TestUpdateToTheCurrentValuesWritesNothing(t *testing.T) {
	root := newStore(t)
	seed(t, root, withLabels(withAssignee(mkIssue("iss001", "Some work"), "alice"), "docs"))
	before := fileOf(t, root, "iss001", "Some work")

	out, err := openAt(t, root).Update("iss001", core.Changes{
		Title:     new("Some work"),
		Assignee:  new("alice"),
		AddLabels: []string{"docs"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Changed {
		t.Error("Changed = true, want false: every value was already what was asked for")
	}
	if after := fileOf(t, root, "iss001", "Some work"); after != before {
		t.Error("a no-op update rewrote the file")
	}
}

// The file name mirrors the title, so retitling moves the issue to the name its
// new slug implies, and leaves no second file behind under the old one.
func TestUpdateRenamesTheFileOnANewTitle(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Old title"))

	out, err := openAt(t, root).Update("iss001", core.Changes{Title: new("  A better title  ")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Issue.Title != "A better title" {
		t.Errorf("title = %q, want it trimmed", out.Issue.Title)
	}
	if exists(t, root, "iss001-old-title.md") {
		t.Error("the file under the old slug is still there")
	}
	if !exists(t, root, "iss001-a-better-title.md") {
		t.Error("the issue was not written under its new slug")
	}
	if detail, err := open(t, root).Get("iss001"); err != nil {
		t.Errorf("Get by id after a retitle: %v", err)
	} else if detail.Issue.ID != "iss001" {
		t.Errorf("id = %q, want iss001: the id survives a retitle", detail.Issue.ID)
	}
}

// A description is the writer's; the notes are other actors' words. Replacing
// one never touches the other.
func TestUpdateBodyPreservesTheNotes(t *testing.T) {
	notes := issue.NotesHeading + "\n\n**bob** — 2026-06-27T18:30:00Z\n\nTried the obvious fix."
	root := newStore(t)
	seed(t, root, withBody(mkIssue("iss001", "Some work"), "The old description.\n\n"+notes))

	out, err := openAt(t, root).Update("iss001", core.Changes{Body: new("The new description.")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !strings.HasPrefix(out.Issue.Body, "The new description.\n\n"+issue.NotesHeading) {
		t.Errorf("body =\n%q\nwant it to open with the new description", out.Issue.Body)
	}
	// Byte-identical from the heading on, whatever the store's round-trip did to
	// the body's edges.
	if got, want := section(out.Issue.Body), section(out.Previous.Body); got != want {
		t.Errorf("notes section =\n%q\nwant\n%q", got, want)
	}
	if entries := issue.ParseNotes(out.Issue.Body); len(entries) != 1 || entries[0].Author != "bob" {
		t.Errorf("notes = %+v, want bob's single entry intact", entries)
	}
}

// With no notes section the whole body is the description, so it is replaced
// outright.
func TestUpdateBodyReplacesAWholeBodyWithoutNotes(t *testing.T) {
	root := newStore(t)
	seed(t, root, withBody(mkIssue("iss001", "Some work"), "The old description."))

	out, err := openAt(t, root).Update("iss001", core.Changes{Body: new("The new description.")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Issue.Body != "The new description." {
		t.Errorf("body = %q, want the new description alone", out.Issue.Body)
	}
}

// An issue cannot wait on itself, directly or through others: the edge that
// would close the loop is refused before anything is written.
func TestUpdateRefusesADependencyCycle(t *testing.T) {
	cases := map[string]struct{ ref, want string }{
		"through another issue": {"iss001", "dep111, iss001"},
		"on itself":             {"dep111", "dep111"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := newStore(t)
			seed(t, root, mkIssue("dep111", "The prerequisite"))
			seed(t, root, withDeps(mkIssue("iss001", "The dependent"), "dep111"))
			before := fileOf(t, root, "dep111", "The prerequisite")

			_, err := openAt(t, root).Update("dep111", core.Changes{AddDependsOn: []string{c.ref}})
			var cycle *core.CycleError
			if !errors.As(err, &cycle) {
				t.Fatalf("Update closing a cycle = %v, want *CycleError", err)
			}
			if cycle.Field != "depends_on" || strings.Join(cycle.Cycle, ", ") != c.want {
				t.Errorf("error = %+v, want depends_on and the cycle %s", cycle, c.want)
			}
			if after := fileOf(t, root, "dep111", "The prerequisite"); after != before {
				t.Error("a refused update rewrote the file")
			}
		})
	}
}

// Hierarchy is a tree: an issue cannot end up its own ancestor either.
func TestUpdateRefusesAParentCycle(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("par222", "The epic"))
	seed(t, root, withParent(mkIssue("kid333", "The sub-issue"), "par222"))

	_, err := openAt(t, root).Update("par222", core.Changes{Parent: new("kid333")})
	var cycle *core.CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("Update closing a parent cycle = %v, want *CycleError", err)
	}
	if cycle.Field != "parent" {
		t.Errorf("Field = %q, want parent", cycle.Field)
	}
}

// Only a cycle the change introduces refuses it. One that arrived some other
// way, by a merge or a hand-edit, is doctor's to report, and refusing every edit
// to an issue caught in one would leave no way to edit it back out.
func TestUpdateEditsAnIssueAlreadyInACycle(t *testing.T) {
	root := newStore(t)
	seed(t, root, withDeps(mkIssue("aaa111", "One side"), "bbb222"))
	seed(t, root, withDeps(mkIssue("bbb222", "The other"), "aaa111"))
	seed(t, root, mkIssue("ccc333", "Unrelated"))

	svc := openAt(t, root)
	if _, err := svc.Update("aaa111", core.Changes{AddDependsOn: []string{"ccc333"}}); err != nil {
		t.Fatalf("Update of an issue already in a cycle = %v, want success", err)
	}
	out, err := svc.Update("aaa111", core.Changes{RemoveDependsOn: []string{"bbb222"}})
	if err != nil {
		t.Fatalf("Update breaking the cycle: %v", err)
	}
	if !slices.Equal(out.Issue.DependsOn, []string{"ccc333"}) {
		t.Errorf("depends_on = %v, want the cycle edge gone and the new one kept", out.Issue.DependsOn)
	}
}

// Input that describes an issue which cannot exist is refused before the store
// is touched at all.
func TestUpdateRefusesInvalidInput(t *testing.T) {
	cases := map[string]struct {
		changes core.Changes
		field   string
	}{
		"empty title":      {core.Changes{Title: new("   ")}, "title"},
		"unknown priority": {core.Changes{Priority: new(issue.Priority("urgentish"))}, "priority"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := newStore(t)
			seed(t, root, mkIssue("iss001", "Some work"))
			before := fileOf(t, root, "iss001", "Some work")

			_, err := openAt(t, root).Update("iss001", c.changes)
			var invalid *core.ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("Update = %v, want *ValidationError", err)
			}
			if invalid.Field != c.field {
				t.Errorf("Field = %q, want %q", invalid.Field, c.field)
			}
			if after := fileOf(t, root, "iss001", "Some work"); after != before {
				t.Error("a refused update rewrote the file")
			}
		})
	}
}

func TestUpdateOfAnUnknownRefIsNotFound(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Only issue"))

	_, err := openAt(t, root).Update("nope", core.Changes{Assignee: new("alice")})
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Update of an unknown ref = %v, want ErrNotFound", err)
	}
}

func TestParsePriority(t *testing.T) {
	cases := map[string]issue.Priority{
		"urgent":  issue.PriorityUrgent,
		"high":    issue.PriorityHigh,
		"medium":  issue.PriorityMedium,
		"  low  ": issue.PriorityLow,
		"none":    "", // the word that clears it
		"":        "",
	}
	for in, want := range cases {
		got, err := core.ParsePriority(in)
		if err != nil {
			t.Errorf("ParsePriority(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParsePriority(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := core.ParsePriority("urgentish"); err == nil {
		t.Error("ParsePriority of a non-level = nil error, want a refusal naming the levels")
	}
}

// section returns a body's notes section, heading included, or "" when it has
// none.
func section(body string) string {
	if i := strings.Index(body, issue.NotesHeading); i >= 0 {
		return body[i:]
	}
	return ""
}

// exists reports whether the store holds an issue file by that name.
func exists(t *testing.T, root, name string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, ".beaver", "issues", name))
	return err == nil
}
