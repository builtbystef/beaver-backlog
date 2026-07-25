package core_test

import (
	"slices"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestListWithoutFiltersReturnsEveryState(t *testing.T) {
	root := newStore(t)
	seedAllStates(t, root)

	got := listIDs(t, root, core.Query{})
	if want := []string{"cncl44", "done33", "prog22", "todo11"}; !slices.Equal(got, want) {
		t.Errorf("unfiltered list = %v, want every issue %v", got, want)
	}
}

func TestListFiltersByState(t *testing.T) {
	root := newStore(t)
	seedAllStates(t, root)

	for _, c := range []struct {
		state issue.State
		want  string
	}{
		{issue.StateTodo, "todo11"},
		{issue.StateInProgress, "prog22"},
		{issue.StateDone, "done33"},
		{issue.StateCancelled, "cncl44"},
	} {
		got := listIDs(t, root, core.Query{States: []issue.State{c.state}})
		if !slices.Equal(got, []string{c.want}) {
			t.Errorf("states=[%s] = %v, want [%s]", c.state, got, c.want)
		}
	}
}

// A state set is a union, so one query can ask for the open work.
func TestListStateSetMatchesAnyMember(t *testing.T) {
	root := newStore(t)
	seedAllStates(t, root)

	got := listIDs(t, root, core.Query{States: []issue.State{issue.StateTodo, issue.StateInProgress}})
	if want := []string{"prog22", "todo11"}; !slices.Equal(got, want) {
		t.Errorf("open states = %v, want %v", got, want)
	}
}

// Readiness is the dependency question the files do not answer: a todo issue
// whose only dependency is done is ready; the same issue is not ready while
// that dependency is anything else, cancelled included.
func TestListReadySelectsTodoWithEveryDependencyDone(t *testing.T) {
	for _, c := range []struct {
		depState  issue.State
		wantReady bool
	}{
		{issue.StateDone, true},
		{issue.StateCancelled, false},
		{issue.StateTodo, false},
		{issue.StateInProgress, false},
	} {
		root := newStore(t)
		seed(t, root, withState(mkIssue("dep111", "Dependency"), c.depState))
		seed(t, root, withDeps(mkIssue("wtr222", "Waiter"), "dep111"))

		got := listIDs(t, root, core.Query{Ready: true})
		want := []string{"dep111"} // the dependency itself is ready only while todo
		if c.depState != issue.StateTodo {
			want = nil
		}
		if c.wantReady {
			want = append(want, "wtr222")
		}
		if !slices.Equal(got, want) {
			t.Errorf("ready with a %s dependency = %v, want %v", c.depState, got, want)
		}
	}
}

// A dependency the store does not hold is unmet like any other, so a dangling
// reference leaves its dependent out of the ready queue rather than through it.
func TestListReadyTreatsAMissingDependencyAsUnmet(t *testing.T) {
	root := newStore(t)
	seed(t, root, withDeps(mkIssue("wtr222", "Waiter"), "gone99"))

	if got := listIDs(t, root, core.Query{Ready: true}); len(got) != 0 {
		t.Errorf("ready = %v, want nothing: the dependency is missing, not met", got)
	}
}

// Blocked is the ready queue's other half — unstarted work that cannot begin —
// so it is scoped to todo, and started work stays out however blocked it is.
func TestListBlockedSelectsOnlyTodo(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("dep111", "Dependency"), issue.StateCancelled))
	seed(t, root, withDeps(mkIssue("wtr222", "Waiter"), "dep111"))
	seed(t, root, withState(withDeps(mkIssue("run333", "Runner"), "dep111"), issue.StateInProgress))

	got := listIDs(t, root, core.Query{Blocked: true})
	if want := []string{"wtr222"}; !slices.Equal(got, want) {
		t.Errorf("blocked = %v, want %v (the in-progress issue is not unstarted work)", got, want)
	}
}

// Ready and blocked are complementary halves, so asking for both selects nothing
// rather than quietly favouring one.
func TestListReadyAndBlockedTogetherSelectNothing(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("dep111", "Dependency"), issue.StateTodo))
	seed(t, root, withDeps(mkIssue("wtr222", "Waiter"), "dep111"))

	if got := listIDs(t, root, core.Query{Ready: true, Blocked: true}); len(got) != 0 {
		t.Errorf("ready and blocked = %v, want nothing", got)
	}
}

func TestListLabelsMatchAll(t *testing.T) {
	root := newStore(t)
	seed(t, root, withLabels(mkIssue("both11", "Both"), "bug", "urgent"))
	seed(t, root, withLabels(mkIssue("one222", "One"), "bug"))
	seed(t, root, mkIssue("none33", "None"))

	if got := listIDs(t, root, core.Query{Labels: []string{"bug"}}); !slices.Equal(got, []string{"both11", "one222"}) {
		t.Errorf("label bug = %v, want both labelled issues", got)
	}
	if got := listIDs(t, root, core.Query{Labels: []string{"bug", "urgent"}}); !slices.Equal(got, []string{"both11"}) {
		t.Errorf("labels bug+urgent = %v, want only the issue carrying both", got)
	}
}

// The empty Priority is the unprioritized, so it is a value a query can ask for;
// an empty slice is the absent filter.
func TestListFiltersByPriority(t *testing.T) {
	root := newStore(t)
	seed(t, root, withPriority(mkIssue("hgh111", "High"), issue.PriorityHigh))
	seed(t, root, withPriority(mkIssue("low222", "Low"), issue.PriorityLow))
	seed(t, root, mkIssue("non333", "Unprioritized"))

	if got := listIDs(t, root, core.Query{Priorities: []issue.Priority{issue.PriorityHigh}}); !slices.Equal(got, []string{"hgh111"}) {
		t.Errorf("priority high = %v, want [hgh111]", got)
	}
	if got := listIDs(t, root, core.Query{Priorities: []issue.Priority{""}}); !slices.Equal(got, []string{"non333"}) {
		t.Errorf("priority none = %v, want the unprioritized issue", got)
	}
}

// Assignee is a pointer because the unassigned are a legitimate thing to ask
// for: nil is no filter, the empty string is "nobody holds it".
func TestListFiltersByAssignee(t *testing.T) {
	root := newStore(t)
	seed(t, root, withAssignee(mkIssue("mine11", "Mine"), "ada"))
	seed(t, root, withAssignee(mkIssue("their2", "Theirs"), "grace"))
	seed(t, root, mkIssue("free33", "Unassigned"))

	if got := listIDs(t, root, core.Query{Assignee: ptr("ada")}); !slices.Equal(got, []string{"mine11"}) {
		t.Errorf("assignee ada = %v, want [mine11]", got)
	}
	if got := listIDs(t, root, core.Query{Assignee: ptr("")}); !slices.Equal(got, []string{"free33"}) {
		t.Errorf("assignee \"\" = %v, want the unassigned issue", got)
	}
}

// Every active dimension applies at once; an issue must satisfy all of them.
func TestListCombinesFilters(t *testing.T) {
	root := newStore(t)
	seed(t, root, withAssignee(withLabels(withPriority(mkIssue("hit111", "Hit"), issue.PriorityHigh), "bug"), "ada"))
	seed(t, root, withAssignee(withLabels(withPriority(mkIssue("mis222", "Wrong label"), issue.PriorityHigh), "chore"), "ada"))
	seed(t, root, withAssignee(withPriority(mkIssue("mis333", "Wrong state"), issue.PriorityHigh), "ada"))

	q := core.Query{
		States:     []issue.State{issue.StateTodo},
		Labels:     []string{"bug"},
		Priorities: []issue.Priority{issue.PriorityHigh},
		Assignee:   ptr("ada"),
	}
	if got := listIDs(t, root, q); !slices.Equal(got, []string{"hit111"}) {
		t.Errorf("combined filters = %v, want [hit111]", got)
	}
}

// The order is priority first, then oldest first, then the id — the last so
// issues minted at the same instant still come back reproducibly.
func TestListOrdersByPriorityThenAgeThenID(t *testing.T) {
	root := newStore(t)
	seed(t, root, withPriority(atTime(mkIssue("low111", "Low but oldest"), fixedTime), issue.PriorityLow))
	seed(t, root, withPriority(atTime(mkIssue("urg222", "Urgent and newest"), fixedTime.Add(3*time.Hour)), issue.PriorityUrgent))
	seed(t, root, atTime(mkIssue("non333", "Unprioritized"), fixedTime))
	seed(t, root, withPriority(atTime(mkIssue("zzz444", "Medium, later id"), fixedTime.Add(time.Hour)), issue.PriorityMedium))
	seed(t, root, withPriority(atTime(mkIssue("aaa555", "Medium, same instant"), fixedTime.Add(time.Hour)), issue.PriorityMedium))

	got := listIDs(t, root, core.Query{})
	want := []string{"urg222", "aaa555", "zzz444", "low111", "non333"}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// seedAllStates writes one issue in each state, deliberately misaligning id order
// with creation order so a test cannot pass on the store's file order alone.
func seedAllStates(t *testing.T, root string) {
	t.Helper()
	seed(t, root, withState(atTime(mkIssue("cncl44", "Dropped"), fixedTime.Add(1*time.Minute)), issue.StateCancelled))
	seed(t, root, withState(atTime(mkIssue("done33", "Shipped"), fixedTime.Add(2*time.Minute)), issue.StateDone))
	seed(t, root, withState(atTime(mkIssue("prog22", "Underway"), fixedTime.Add(3*time.Minute)), issue.StateInProgress))
	seed(t, root, withState(atTime(mkIssue("todo11", "Waiting"), fixedTime.Add(4*time.Minute)), issue.StateTodo))
}

// listIDs runs a query against the store at root and returns the ids it selected,
// in the order the core put them in.
func listIDs(t *testing.T, root string, q core.Query) []string {
	t.Helper()
	listing, err := open(t, root).List(q)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return ids(listing.Issues)
}

func withLabels(iss issue.Issue, labels ...string) issue.Issue {
	iss.Labels = labels
	return iss
}

func withPriority(iss issue.Issue, p issue.Priority) issue.Issue {
	iss.Priority = p
	return iss
}

func withAssignee(iss issue.Issue, actor string) issue.Issue {
	iss.Assignee = actor
	return iss
}

func atTime(iss issue.Issue, t time.Time) issue.Issue {
	iss.Created, iss.Updated = t, t
	return iss
}

func ptr(s string) *string { return &s }
