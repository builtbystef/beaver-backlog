package cli

import (
	"fmt"
	"slices"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
)

// A verb is a lifecycle transition: it moves an issue to target from any of a
// fixed set of source states. done and cancel close an open issue (todo or
// in-progress); reopen restores a closed one (done or cancelled) to todo. The
// four states split two-open/two-closed, so each verb accepts exactly two
// sources, treats its target as an idempotent no-op, and rejects the one
// remaining state — see classify.
type verb struct {
	name    string        // command name, used in usage diagnostics
	target  issue.State   // state a successful transition sets
	sources []issue.State // states the transition may start from

	did     string // human confirmation on a real transition; one %s (id)
	already string // human line when already at target; one %s (id)
	reject  string // stderr guidance when the current state forbids it; two %s (id, current state)
}

var (
	verbDone = verb{
		name:    "done",
		target:  issue.StateDone,
		sources: []issue.State{issue.StateTodo, issue.StateInProgress},
		did:     "Marked %s done",
		already: "%s is already done",
		reject:  "%s is %s; reopen it first to mark it done",
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

// classify decides how verb v applies to an issue currently in state cur. The
// rule is uniform across all three verbs: already-at-target is a redundant no-op
// (idempotent success, no rewrite); a listed source transitions; anything else is
// a graceful rejection. Because target and sources together cover three of the
// four states, the reject case is exactly the remaining one — the opposite
// terminal state for done/cancel, and in-progress for reopen (reopen restores a
// closed issue; it deliberately will not rewind active work to todo).
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

// runTransition is the shared engine behind done, cancel, and reopen: resolve the
// referenced issue, decide the transition, and either rewrite the file with the
// new state and a bumped `updated`, report an idempotent no-op, or refuse a
// transition the current state forbids. A refusal never touches the file, so a
// nonsensical request cannot corrupt an issue.
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
	format, err := output.Resolve(*formatFlag, env.StdoutIsTTY, env.Getenv)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return storeError(env, err)
	}

	iss, path, code := resolveRef(env, st, ref)
	if code != exitOK {
		return code
	}

	switch classify(v, iss.State) {
	case transReject:
		// A well-formed request that does not apply to this issue's state. It is a
		// usage error (exit 2), not a runtime failure: nothing is written, so the
		// file cannot be corrupted, and the message says how to proceed.
		errf(env, v.reject, iss.ID, iss.State)
		return exitUsage

	case transRedundant:
		// Already in the target state. Idempotent success: report the issue but do
		// not rewrite it, so `updated` and the file bytes stay untouched.
		return reportTransition(env, format, iss, fmt.Sprintf(v.already, iss.ID))
	}

	// transApply: set the new state and bump `updated` from the injected clock,
	// matching create's timestamp handling. The read-modify-write carries the body
	// and any custom frontmatter keys through untouched (ADR 0014).
	iss.State = v.target
	iss.Updated = env.Clock.Now().UTC().Truncate(time.Second)
	if _, err := st.Update(path, iss); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return reportTransition(env, format, iss, fmt.Sprintf(v.did, iss.ID))
}

// reportTransition renders a completed transition: a concise confirmation line
// for a human, or the full resulting issue as JSON for a machine — the same
// per-issue shape create and show emit, so an agent sees the new state directly.
func reportTransition(env Env, format output.Format, iss issue.Issue, humanLine string) int {
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
