package issue_test

import (
	"slices"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// rel builds a Relations index from issues described inline.
func rel(issues ...issue.Issue) *issue.Relations { return issue.NewRelations(issues) }

// dep builds a dependent issue: id, state, and the ids it depends on.
func dep(id string, state issue.State, deps ...string) issue.Issue {
	return issue.Issue{ID: id, Title: id, State: state, DependsOn: deps}
}

// BlockedOn returns exactly the not-done dependencies, in stored order: a
// present target carries its state, a missing one is flagged.
func TestBlockedOnClassifiesEachDependency(t *testing.T) {
	r := rel(
		dep("done00", issue.StateDone),
		dep("todo00", issue.StateTodo),
		dep("prog00", issue.StateInProgress),
		dep("cncl00", issue.StateCancelled),
		dep("waiter", issue.StateTodo, "done00", "todo00", "prog00", "cncl00", "gone00"),
	)
	waiter := dep("waiter", issue.StateTodo, "done00", "todo00", "prog00", "cncl00", "gone00")

	got := r.BlockedOn(waiter)
	want := []issue.Blocker{
		{ID: "todo00", State: issue.StateTodo},
		{ID: "prog00", State: issue.StateInProgress},
		{ID: "cncl00", State: issue.StateCancelled},
		{ID: "gone00", Missing: true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("BlockedOn = %+v\nwant %+v (done dependency dropped, order preserved)", got, want)
	}
	// Only the cancelled dependency counts as un-satisfiable-on-its-own.
	if !want[2].Cancelled() || want[0].Cancelled() || want[3].Cancelled() {
		t.Errorf("Cancelled() should be true only for the cancelled target: %+v", want)
	}
}

// A dependency listed twice is one edge and appears once in BlockedOn.
func TestBlockedOnCountsADuplicatedEdgeOnce(t *testing.T) {
	waiter := dep("waiter", issue.StateTodo, "todo00", "todo00", "gone00", "gone00")
	r := rel(dep("todo00", issue.StateTodo), waiter)
	got := r.BlockedOn(waiter)
	want := []issue.Blocker{
		{ID: "todo00", State: issue.StateTodo},
		{ID: "gone00", Missing: true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("BlockedOn = %+v\nwant %+v (each duplicated edge counted once)", got, want)
	}
}

// A dependency is satisfied only by done: a target in any other state, or
// missing, keeps the todo dependent out of the ready set.
func TestOnlyDoneSatisfiesADependency(t *testing.T) {
	for _, depState := range []issue.State{issue.StateTodo, issue.StateInProgress, issue.StateCancelled} {
		r := rel(dep("target", depState), dep("waiter", issue.StateTodo, "target"))
		if r.Ready(dep("waiter", issue.StateTodo, "target")) {
			t.Errorf("dependent is ready with a %q dependency; only done should satisfy", depState)
		}
	}
	// A missing target is unmet too.
	rMissing := rel(dep("waiter", issue.StateTodo, "target"))
	if rMissing.Ready(dep("waiter", issue.StateTodo, "target")) {
		t.Error("dependent is ready with a missing dependency; a dangling ref is unmet")
	}
	// Flip the one satisfying case: a done target makes it ready.
	rDone := rel(dep("target", issue.StateDone), dep("waiter", issue.StateTodo, "target"))
	if !rDone.Ready(dep("waiter", issue.StateTodo, "target")) {
		t.Error("dependent with a done dependency should be ready")
	}
}

// Ready is todo-only: starting or finishing an issue takes it out of the ready
// set even though nothing blocks it.
func TestReadyIsTodoWithNoBlockers(t *testing.T) {
	r := rel()
	cases := map[issue.State]bool{
		issue.StateTodo:       true,
		issue.StateInProgress: false,
		issue.StateDone:       false,
		issue.StateCancelled:  false,
	}
	for state, want := range cases {
		if got := r.Ready(dep("solo00", state)); got != want {
			t.Errorf("Ready(state=%q, no deps) = %v, want %v", state, got, want)
		}
	}
}

// A cancelled dependency leaves the dependent stuck, distinct from an ordinary
// not-yet-done blocker.
func TestStuckIsACancelledDependency(t *testing.T) {
	stuck := dep("waiter", issue.StateTodo, "target")

	rCancel := rel(dep("target", issue.StateCancelled), stuck)
	if rCancel.Ready(stuck) || !rCancel.Blocked(stuck) || !rCancel.Stuck(stuck) {
		t.Errorf("cancelled dep: ready=%v blocked=%v stuck=%v, want false/true/true",
			rCancel.Ready(stuck), rCancel.Blocked(stuck), rCancel.Stuck(stuck))
	}
	// An ordinary not-done dependency is blocked but not stuck: it can still clear.
	rTodo := rel(dep("target", issue.StateTodo), stuck)
	if rTodo.Stuck(stuck) {
		t.Error("a todo dependency is not stuck; only cancellation makes it un-clearable")
	}
	// A missing dependency blocks but is not stuck.
	rMissing := rel(stuck)
	if !rMissing.Blocked(stuck) || rMissing.Stuck(stuck) {
		t.Errorf("missing dep: blocked=%v stuck=%v, want true/false",
			rMissing.Blocked(stuck), rMissing.Stuck(stuck))
	}
}

// Blocks is the reverse of depends_on and Children the reverse of parent, each
// derived by scanning and sorted.
func TestInverseEdgesAreDerived(t *testing.T) {
	target := dep("target", issue.StateTodo)
	a := issue.Issue{ID: "aaa111", State: issue.StateTodo, DependsOn: []string{"target"}}
	b := issue.Issue{ID: "bbb222", State: issue.StateTodo, DependsOn: []string{"target"}}
	child1 := issue.Issue{ID: "chi111", State: issue.StateTodo, Parent: "epic00"}
	child2 := issue.Issue{ID: "chi222", State: issue.StateTodo, Parent: "epic00"}
	epic := dep("epic00", issue.StateTodo)
	r := rel(target, a, b, child1, child2, epic)

	if got := r.Blocks(target); !slices.Equal(got, []string{"aaa111", "bbb222"}) {
		t.Errorf("Blocks(target) = %v, want the two dependents sorted", got)
	}
	if got := r.Blocks(a); len(got) != 0 {
		t.Errorf("Blocks(a) = %v, want none (nothing depends on a)", got)
	}
	if got := r.Children(epic); !slices.Equal(got, []string{"chi111", "chi222"}) {
		t.Errorf("Children(epic) = %v, want the two sub-issues sorted", got)
	}
	if got := r.Children(target); len(got) != 0 {
		t.Errorf("Children(target) = %v, want none", got)
	}
}

// For assembles the whole snapshot, consistent with the individual predicates.
func TestForAssemblesTheSnapshot(t *testing.T) {
	waiter := dep("waiter", issue.StateTodo, "cncl00")
	r := rel(dep("cncl00", issue.StateCancelled), waiter, issue.Issue{ID: "dep111", State: issue.StateTodo, DependsOn: []string{"waiter"}})

	got := r.For(waiter)
	if got.Ready || !got.Blocked || !got.Stuck {
		t.Errorf("For(stuck waiter): ready=%v blocked=%v stuck=%v, want false/true/true", got.Ready, got.Blocked, got.Stuck)
	}
	if len(got.BlockedOn) != 1 || got.BlockedOn[0].ID != "cncl00" {
		t.Errorf("BlockedOn = %+v, want the single cancelled dependency", got.BlockedOn)
	}
	if !slices.Equal(got.Blocks, []string{"dep111"}) {
		t.Errorf("Blocks = %v, want [dep111] (the issue that depends on waiter)", got.Blocks)
	}
}
