package cli

import (
	"fmt"
	"slices"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
)

// verb is a lifecycle transition: it moves an issue to target from any of a
// fixed set of source states, treating already-at-target as an idempotent
// no-op — see classify.
type verb struct {
	name    string        // command name, used in usage diagnostics
	target  issue.State   // state a successful transition sets
	sources []issue.State // states the transition may start from

	did     string // human confirmation on a real transition; one %s (id)
	already string // human line when already at target; one %s (id)
	reject  string // stderr guidance when the current state forbids it; two %s (id, current state)

	// completes marks the verb that finishes an issue (done), which may record
	// the opt-in commit-per-issue.
	completes bool
}

var (
	verbDone = verb{
		name:      "done",
		target:    issue.StateDone,
		sources:   []issue.State{issue.StateTodo, issue.StateInProgress},
		did:       "Marked %s done",
		already:   "%s is already done",
		reject:    "%s is %s; reopen it first to mark it done",
		completes: true,
	}
	verbCancel = verb{
		name:    "cancel",
		target:  issue.StateCancelled,
		sources: []issue.State{issue.StateTodo, issue.StateInProgress},
		did:     "Cancelled %s",
		already: "%s is already cancelled",
		reject:  "%s is %s; reopen it first to cancel it",
	}
	verbReopen = verb{
		name:    "reopen",
		target:  issue.StateTodo,
		sources: []issue.State{issue.StateDone, issue.StateCancelled},
		did:     "Reopened %s",
		already: "%s is already todo",
		reject:  "%s is %s, not closed; reopen only restores done or cancelled issues",
	}
)

func cmdDone(env Env, args []string) int   { return runTransition(env, args, verbDone) }
func cmdCancel(env Env, args []string) int { return runTransition(env, args, verbCancel) }
func cmdReopen(env Env, args []string) int { return runTransition(env, args, verbReopen) }

// transitionKind is the outcome of applying a verb to an issue's current state.
type transitionKind int

const (
	transApply     transitionKind = iota // move the issue to the verb's target state
	transRedundant                       // already at target: an idempotent no-op success
	transReject                          // the current state forbids this verb
)

// classify decides how verb v applies to an issue currently in state cur:
// already-at-target is a redundant no-op, a listed source transitions, and
// anything else is rejected.
func classify(v verb, cur issue.State) transitionKind {
	switch {
	case cur == v.target:
		return transRedundant
	case slices.Contains(v.sources, cur):
		return transApply
	default:
		return transReject
	}
}

// runTransition is the shared engine behind done, cancel, and reopen: it
// resolves the referenced issue and either rewrites it with the new state and a
// bumped `updated`, reports an idempotent no-op, or refuses a transition the
// current state forbids. A refusal never touches the file.
func runTransition(env Env, args []string, v verb) int {
	fs, formatFlag := newFlagSet(env, v.name)
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "%s requires an issue reference: beaver %s <ref>", v.name, v.name)
		return exitUsage
	}
	ref := pos[0]
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	iss, path, code := resolveRef(env, st, ref)
	if code != exitOK {
		return code
	}

	switch classify(v, iss.State) {
	case transReject:
		errf(env, v.reject, iss.ID, iss.State)
		return exitUsage

	case transRedundant:
		// Idempotent no-op: report without rewriting, so `updated` and the file
		// bytes stay untouched. The completing verb still uses the
		// commit-carrying shape so done's JSON is constant across no-op and
		// real transitions.
		line := fmt.Sprintf(v.already, iss.ID)
		if v.completes {
			return reportCompletion(env, format, iss, "", line)
		}
		return reportIssue(env, format, iss, line)
	}

	iss.State = v.target
	iss.Updated = env.Clock.Now().UTC().Truncate(time.Second)
	newPath, err := st.Update(path, iss)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	line := fmt.Sprintf(v.did, iss.ID)
	// The opt-in completion commit runs only after the file is safely written
	// and never fails the command — the issue is done regardless.
	if v.completes {
		rev := commitCompletion(env, st, iss, commitPaths(newPath, path))
		if rev != "" {
			line += fmt.Sprintf(" (committed %s)", rev)
		}
		return reportCompletion(env, format, iss, rev, line)
	}
	return reportIssue(env, format, iss, line)
}

// reportCompletion is reportIssue for the completing verb: the JSON result adds
// an always-present "commit" key (null when no commit was made), so done's JSON
// has one constant shape.
func reportCompletion(env Env, format output.Format, iss issue.Issue, revision, humanLine string) int {
	if format == output.Human {
		fmt.Fprintln(env.Stdout, humanLine)
		return exitOK
	}
	if err := output.WriteIssueWithCommit(env.Stdout, iss, revision, output.JSON); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}

// reportIssue renders a completed command's result: a concise confirmation line
// for a human, or the full resulting issue as JSON for a machine.
func reportIssue(env Env, format output.Format, iss issue.Issue, humanLine string) int {
	if format == output.Human {
		fmt.Fprintln(env.Stdout, humanLine)
		return exitOK
	}
	if err := output.WriteIssue(env.Stdout, iss, output.JSON); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}
