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

// writeTime is when the tests below pretend writes happen — well after the
// instant seeded issues carry, so a bumped `updated` is unmistakable and a
// stray write cannot hide behind an unchanged timestamp.
var writeTime = fixedTime.Add(48 * time.Hour)

func TestTransitionAppliesTheLegalityTable(t *testing.T) {
	cases := []struct {
		from  issue.State
		to    issue.State
		legal bool
	}{
		{issue.StateTodo, issue.StateDone, true},
		{issue.StateInProgress, issue.StateDone, true},
		{issue.StateTodo, issue.StateCancelled, true},
		{issue.StateInProgress, issue.StateCancelled, true},
		{issue.StateDone, issue.StateTodo, true},
		{issue.StateCancelled, issue.StateTodo, true},
		{issue.StateCancelled, issue.StateDone, false}, // closed: reopen it first
		{issue.StateDone, issue.StateCancelled, false}, // closed: reopen it first
		{issue.StateInProgress, issue.StateTodo, false},
	}
	for _, c := range cases {
		t.Run(string(c.from)+"-to-"+string(c.to), func(t *testing.T) {
			root := newStore(t)
			seed(t, root, withState(mkIssue("iss001", "Some work"), c.from))
			before := fileOf(t, root, "iss001", "Some work")

			out, err := openAt(t, root).Transition("iss001", c.to)
			if !c.legal {
				var illegal *core.IllegalTransitionError
				if !errors.As(err, &illegal) {
					t.Fatalf("Transition %s → %s = %v, want *IllegalTransitionError", c.from, c.to, err)
				}
				if illegal.ID != "iss001" || illegal.From != c.from || illegal.To != c.to {
					t.Errorf("error = %+v, want iss001 from %s to %s", illegal, c.from, c.to)
				}
				if after := fileOf(t, root, "iss001", "Some work"); after != before {
					t.Errorf("a refused transition rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
				}
				return
			}
			if err != nil {
				t.Fatalf("Transition %s → %s: %v", c.from, c.to, err)
			}
			if !out.Changed {
				t.Error("Changed = false, want true for a state that moved")
			}
			if out.Issue.State != c.to {
				t.Errorf("state = %s, want %s", out.Issue.State, c.to)
			}
			if !out.Issue.Updated.Equal(writeTime) {
				t.Errorf("updated = %s, want it bumped to %s", out.Issue.Updated, writeTime)
			}
			// The move is on disk, not just in the returned value.
			detail, err := open(t, root).Get("iss001")
			if err != nil {
				t.Fatalf("Get after Transition: %v", err)
			}
			if detail.Issue.State != c.to || !detail.Issue.Updated.Equal(writeTime) {
				t.Errorf("persisted state/updated = %s/%s, want %s/%s",
					detail.Issue.State, detail.Issue.Updated, c.to, writeTime)
			}
		})
	}
}

// A transition moves the state and nothing else: the assignee is kept as the
// record of who did the work, so a closed issue still names them.
func TestTransitionKeepsTheAssignee(t *testing.T) {
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

// A transition to the state an issue already holds is an idempotent no-op: it
// reports the issue unchanged and leaves the file — and its `updated` — alone.
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

func TestStartRefusesAClosedIssue(t *testing.T) {
	for _, state := range []issue.State{issue.StateDone, issue.StateCancelled} {
		t.Run(string(state), func(t *testing.T) {
			root := newStore(t)
			seed(t, root, withState(mkIssue("iss001", "Closed"), state))
			before := fileOf(t, root, "iss001", "Closed")

			_, err := openAt(t, root).Start("iss001", "alice", false)
			var illegal *core.IllegalTransitionError
			if !errors.As(err, &illegal) {
				t.Fatalf("Start of a %s issue = %v, want *IllegalTransitionError", state, err)
			}
			if illegal.From != state || illegal.To != issue.StateInProgress {
				t.Errorf("error = %+v, want from %s to in-progress", illegal, state)
			}
			if after := fileOf(t, root, "iss001", "Closed"); after != before {
				t.Errorf("a refused start rewrote the %s file", state)
			}
		})
	}
}

// Unmet dependencies are data, never a refusal: starting blocked work is
// sometimes the right call, so the work begins and the caller is handed what it
// began in spite of — including a dangling reference no issue answers.
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
// had its turn when it started, so a later start reports none — while the
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
