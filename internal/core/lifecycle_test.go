package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/clock"
	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// writeTime is when the tests below pretend writes happen: well after the
// instant seeded issues carry, so a bumped `updated` is unmistakable and a
// stray write cannot hide behind an unchanged timestamp.
var writeTime = fixedTime.Add(48 * time.Hour)

// Every move is legal: any state may reach any transition target. The files
// are hand-editable markdown, so the tools describe changes rather than
// gatekeep them.
func TestTransitionMovesBetweenAnyTwoStates(t *testing.T) {
	states := []issue.State{issue.StateTodo, issue.StateInProgress, issue.StateDone, issue.StateCancelled}
	targets := []issue.State{issue.StateTodo, issue.StateDone, issue.StateCancelled}
	for _, from := range states {
		for _, to := range targets {
			if from == to {
				continue // the idempotent no-op has its own test
			}
			t.Run(string(from)+"-to-"+string(to), func(t *testing.T) {
				root := newStore(t)
				seed(t, root, withState(mkIssue("iss001", "Some work"), from))

				out, err := openAt(t, root).Transition("iss001", to)
				if err != nil {
					t.Fatalf("Transition %s → %s: %v", from, to, err)
				}
				if !out.Changed {
					t.Error("Changed = false, want true for a state that moved")
				}
				if out.Issue.State != to {
					t.Errorf("state = %s, want %s", out.Issue.State, to)
				}
				if !out.Issue.Updated.Equal(writeTime) {
					t.Errorf("updated = %s, want it bumped to %s", out.Issue.Updated, writeTime)
				}
				// The move is on disk, not just in the returned value.
				detail, err := open(t, root).Get("iss001")
				if err != nil {
					t.Fatalf("Get after Transition: %v", err)
				}
				if detail.Issue.State != to || !detail.Issue.Updated.Equal(writeTime) {
					t.Errorf("persisted state/updated = %s/%s, want %s/%s",
						detail.Issue.State, detail.Issue.Updated, to, writeTime)
				}
			})
		}
	}
}

// A transition to a closed state keeps the assignee as the record of who did
// the work, so a closed issue still names them.
func TestTransitionToAClosedStateKeepsTheAssignee(t *testing.T) {
	root := newStore(t)
	seed(t, root, withAssignee(withState(mkIssue("iss001", "Work to finish"), issue.StateInProgress), "alice"))

	out, err := openAt(t, root).Transition("iss001", issue.StateDone)
	if err != nil {
		t.Fatalf("Transition: %v", err)
	}
	if out.Issue.Assignee != "alice" {
		t.Errorf("assignee after done = %q, want alice retained", out.Issue.Assignee)
	}
	detail, err := open(t, root).Get("iss001")
	if err != nil {
		t.Fatalf("Get after Transition: %v", err)
	}
	if detail.Issue.Assignee != "alice" {
		t.Errorf("persisted assignee = %q, want alice", detail.Issue.Assignee)
	}
}

// Entering todo clears the assignee: todo is the unowned pile, and whoever
// picks the issue up next claims it by starting it. This holds wherever the
// issue came from, a closed state or active work.
func TestTransitionToTodoClearsTheAssignee(t *testing.T) {
	for _, from := range []issue.State{issue.StateInProgress, issue.StateDone, issue.StateCancelled} {
		t.Run(string(from), func(t *testing.T) {
			root := newStore(t)
			seed(t, root, withAssignee(withState(mkIssue("iss001", "Owned work"), from), "alice"))

			out, err := openAt(t, root).Transition("iss001", issue.StateTodo)
			if err != nil {
				t.Fatalf("Transition: %v", err)
			}
			if out.Issue.Assignee != "" {
				t.Errorf("assignee after todo = %q, want cleared", out.Issue.Assignee)
			}
			if out.Previous.Assignee != "alice" {
				t.Errorf("previous assignee = %q, want alice (who held it)", out.Previous.Assignee)
			}
			detail, err := open(t, root).Get("iss001")
			if err != nil {
				t.Fatalf("Get after Transition: %v", err)
			}
			if detail.Issue.Assignee != "" {
				t.Errorf("persisted assignee = %q, want cleared", detail.Issue.Assignee)
			}
		})
	}
}

// A transition to the state an issue already holds is an idempotent no-op: it
// reports the issue unchanged and leaves the file, and its `updated`, alone.
func TestTransitionToTheCurrentStateWritesNothing(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("iss001", "Some work"), issue.StateDone))
	before := fileOf(t, root, "iss001", "Some work")

	out, err := openAt(t, root).Transition("iss001", issue.StateDone)
	if err != nil {
		t.Fatalf("Transition of an already-done issue: %v", err)
	}
	if out.Changed {
		t.Error("Changed = true, want false: the issue was already done")
	}
	if out.Issue.State != issue.StateDone {
		t.Errorf("state = %s, want done", out.Issue.State)
	}
	if !out.Issue.Updated.Equal(fixedTime) {
		t.Errorf("updated = %s, want unchanged %s (no bump on a no-op)", out.Issue.Updated, fixedTime)
	}
	if after := fileOf(t, root, "iss001", "Some work"); after != before {
		t.Errorf("a no-op transition rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// The assignee clearing belongs to entering todo, not to being asked for it: a
// transition to the state an issue already holds is a pure no-op, so an
// already-todo issue keeps its assignee, since unassigning is update's job, not
// reopen's.
func TestTransitionToTodoOfATodoIssueKeepsTheAssignee(t *testing.T) {
	root := newStore(t)
	seed(t, root, withAssignee(mkIssue("iss001", "Claimed but unstarted"), "alice"))
	before := fileOf(t, root, "iss001", "Claimed but unstarted")

	out, err := openAt(t, root).Transition("iss001", issue.StateTodo)
	if err != nil {
		t.Fatalf("Transition: %v", err)
	}
	if out.Changed || out.Issue.Assignee != "alice" {
		t.Errorf("changed/assignee = %v/%q, want false/alice: a no-op must not unassign", out.Changed, out.Issue.Assignee)
	}
	if after := fileOf(t, root, "iss001", "Claimed but unstarted"); after != before {
		t.Errorf("a no-op transition rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// Beginning work is Start's, not a transition's: it also claims the issue and
// reports what the work waits on.
func TestTransitionRefusesInProgress(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Some work"))

	if _, err := openAt(t, root).Transition("iss001", issue.StateInProgress); !errors.Is(err, core.ErrNotATransition) {
		t.Errorf("Transition to in-progress = %v, want ErrNotATransition", err)
	}
}

func TestTransitionOfAnUnknownRefIsNotFound(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Only issue"))

	if _, err := openAt(t, root).Transition("nope", issue.StateDone); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Transition of an unknown ref = %v, want ErrNotFound", err)
	}
}

func TestStartBeginsWorkAndClaims(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Unowned todo"))

	out, err := openAt(t, root).Start("iss001", "alice", false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !out.Changed {
		t.Error("Changed = false, want true")
	}
	if out.Issue.State != issue.StateInProgress || out.Issue.Assignee != "alice" {
		t.Errorf("state/assignee = %s/%s, want in-progress/alice", out.Issue.State, out.Issue.Assignee)
	}
	if !out.Issue.Updated.Equal(writeTime) {
		t.Errorf("updated = %s, want it bumped to %s", out.Issue.Updated, writeTime)
	}
	// Previous is what the caller compares against to describe what happened.
	if out.Previous.State != issue.StateTodo || out.Previous.Assignee != "" {
		t.Errorf("previous state/assignee = %s/%q, want todo/unowned", out.Previous.State, out.Previous.Assignee)
	}

	detail, err := open(t, root).Get("iss001")
	if err != nil {
		t.Fatalf("Get after Start: %v", err)
	}
	if detail.Issue.State != issue.StateInProgress || detail.Issue.Assignee != "alice" {
		t.Errorf("persisted = %s/%s, want in-progress/alice", detail.Issue.State, detail.Issue.Assignee)
	}
}

// The auto-claim applies even when the state does not move, so an ownerless
// in-progress issue gains an owner.
func TestStartClaimsAnUnownedInProgressIssue(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("iss001", "Ownerless active"), issue.StateInProgress))

	out, err := openAt(t, root).Start("iss001", "alice", false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !out.Changed || out.Issue.Assignee != "alice" || out.Issue.State != issue.StateInProgress {
		t.Errorf("changed/assignee/state = %v/%s/%s, want true/alice/in-progress",
			out.Changed, out.Issue.Assignee, out.Issue.State)
	}
}

// The guard is advisory but real: another actor's issue is refused, and only
// force takes it.
func TestStartGuardsOwnershipUnlessForced(t *testing.T) {
	root := newStore(t)
	seed(t, root, withAssignee(mkIssue("iss001", "Bob's work"), "bob"))
	before := fileOf(t, root, "iss001", "Bob's work")

	_, err := openAt(t, root).Start("iss001", "alice", false)
	var claimed *core.ClaimedError
	if !errors.As(err, &claimed) {
		t.Fatalf("Start of another's issue = %v, want *ClaimedError", err)
	}
	if claimed.ID != "iss001" || claimed.By != "bob" {
		t.Errorf("error = %+v, want iss001 held by bob", claimed)
	}
	if after := fileOf(t, root, "iss001", "Bob's work"); after != before {
		t.Errorf("a refused start rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	out, err := openAt(t, root).Start("iss001", "alice", true)
	if err != nil {
		t.Fatalf("forced Start: %v", err)
	}
	if out.Issue.Assignee != "alice" || out.Issue.State != issue.StateInProgress {
		t.Errorf("after a forced start = %s/%s, want alice/in-progress", out.Issue.Assignee, out.Issue.State)
	}
	if out.Previous.Assignee != "bob" {
		t.Errorf("previous assignee = %q, want bob (the owner it was taken from)", out.Previous.Assignee)
	}
}

func TestStartOnOwnInProgressIssueWritesNothing(t *testing.T) {
	root := newStore(t)
	seed(t, root, withAssignee(withState(mkIssue("iss001", "Already going"), issue.StateInProgress), "alice"))
	before := fileOf(t, root, "iss001", "Already going")

	out, err := openAt(t, root).Start("iss001", "alice", false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if out.Changed {
		t.Error("Changed = true, want false: the issue was already alice's and already in progress")
	}
	if !out.Issue.Updated.Equal(fixedTime) {
		t.Errorf("updated = %s, want unchanged %s (no bump on a no-op)", out.Issue.Updated, fixedTime)
	}
	if after := fileOf(t, root, "iss001", "Already going"); after != before {
		t.Errorf("a no-op start rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// A closed issue is resurrected into active work in one move: starting it
// claims it and puts it in progress, with no reopen dance in between. Work
// begins here too, so its unmet dependencies are reported like any other start.
func TestStartResurrectsAClosedIssue(t *testing.T) {
	for _, state := range []issue.State{issue.StateDone, issue.StateCancelled} {
		t.Run(string(state), func(t *testing.T) {
			root := newStore(t)
			seed(t, root, withState(mkIssue("dep001", "Prerequisite"), issue.StateInProgress))
			seed(t, root, withDeps(withState(mkIssue("iss001", "Closed"), state), "dep001"))

			out, err := openAt(t, root).Start("iss001", "alice", false)
			if err != nil {
				t.Fatalf("Start of a %s issue: %v", state, err)
			}
			if !out.Changed || out.Issue.State != issue.StateInProgress || out.Issue.Assignee != "alice" {
				t.Errorf("changed/state/assignee = %v/%s/%s, want true/in-progress/alice",
					out.Changed, out.Issue.State, out.Issue.Assignee)
			}
			if out.Previous.State != state {
				t.Errorf("previous state = %s, want %s", out.Previous.State, state)
			}
			if len(out.UnmetDependencies) != 1 || out.UnmetDependencies[0].ID != "dep001" {
				t.Errorf("unmet dependencies = %v, want dep001: this call began the work", out.UnmetDependencies)
			}
		})
	}
}

// Unmet dependencies are data, never a refusal: starting blocked work is
// sometimes the right call, so the work begins and the caller is handed what it
// began in spite of, including a dangling reference no issue answers.
func TestStartReportsUnmetDependenciesAndStillBegins(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("dep001", "Prerequisite"), issue.StateInProgress))
	seed(t, root, withDeps(mkIssue("iss001", "The dependent"), "dep001", "gone99"))

	out, err := openAt(t, root).Start("iss001", "alice", false)
	if err != nil {
		t.Fatalf("Start of a not-ready issue = %v, want success (the dependencies are advisory)", err)
	}
	if out.Issue.State != issue.StateInProgress {
		t.Errorf("state = %s, want in-progress (started despite the dependencies)", out.Issue.State)
	}
	if len(out.UnmetDependencies) != 2 {
		t.Fatalf("unmet dependencies = %v, want both dep001 and gone99", out.UnmetDependencies)
	}
	if b := out.UnmetDependencies[0]; b.ID != "dep001" || b.Missing || b.State != issue.StateInProgress {
		t.Errorf("first blocker = %+v, want dep001 in-progress and present", b)
	}
	if b := out.UnmetDependencies[1]; b.ID != "gone99" || !b.Missing {
		t.Errorf("second blocker = %+v, want gone99 reported as missing", b)
	}
	// The same facts reach a caller that renders relationships rather than the
	// warning, computed over the issue as it now stands.
	if !out.Relationship.Blocked || out.Relationship.Ready {
		t.Errorf("relationship blocked/ready = %v/%v, want true/false",
			out.Relationship.Blocked, out.Relationship.Ready)
	}
}

// The dependency report belongs to work beginning: an issue already in progress
// had its turn when it started, so a later start reports none, while the
// relationship view still shows it blocked.
func TestStartReportsNoDependenciesWhenWorkAlreadyBegan(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("dep001", "Prerequisite"), issue.StateInProgress))
	seed(t, root, withDeps(withState(mkIssue("iss001", "The dependent"), issue.StateInProgress), "dep001"))

	out, err := openAt(t, root).Start("iss001", "alice", false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(out.UnmetDependencies) != 0 {
		t.Errorf("unmet dependencies = %v, want none: this call did not begin the work", out.UnmetDependencies)
	}
	if !out.Relationship.Blocked {
		t.Error("relationship blocked = false, want true: the dependency is still unmet")
	}
}

func TestStartOfAnUnknownRefIsNotFound(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Only issue"))

	if _, err := openAt(t, root).Start("nope", "alice", false); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Start of an unknown ref = %v, want ErrNotFound", err)
	}
}

// openAt returns a service whose writes stamp `updated` with writeTime.
func openAt(t *testing.T, dir string) *core.Service {
	t.Helper()
	svc, err := core.Open(dir, core.WithClock(clock.Fixed(writeTime)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return svc
}

// fileOf returns the bytes of an issue's file, so a test can prove a call left
// it untouched rather than merely rewrote it to the same state.
func fileOf(t *testing.T, root, id, title string) string {
	t.Helper()
	path := filepath.Join(root, ".beaver", "issues", issue.FileName(id, issue.Slug(title)))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
