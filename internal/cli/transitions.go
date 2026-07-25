package cli

import (
	"errors"
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// verb is a lifecycle transition as this CLI presents it: the state it sets and
// the wording for each way the core can answer. Which states may reach the
// target, and whether a call changes anything, is the core's business.
type verb struct {
	name   string      // command name, used in usage diagnostics
	target issue.State // state a successful transition sets

	did     string // human confirmation on a real transition; one %s (id)
	already string // human line when already at target; one %s (id)
	reject  string // stderr guidance when the current state forbids it; two %s (id, current state)
}

var (
	verbDone = verb{
		name:    "done",
		target:  issue.StateDone,
		did:     "Marked %s done",
		already: "%s is already done",
		reject:  "%s is %s; reopen it first to mark it done",
	}
	verbCancel = verb{
		name:    "cancel",
		target:  issue.StateCancelled,
		did:     "Cancelled %s",
		already: "%s is already cancelled",
		reject:  "%s is %s; reopen it first to cancel it",
	}
	verbReopen = verb{
		name:    "reopen",
		target:  issue.StateTodo,
		did:     "Reopened %s",
		already: "%s is already todo",
		reject:  "%s is %s, not closed; reopen only restores done or cancelled issues",
	}
)

func cmdDone(env Env, args []string) int   { return runTransition(env, args, verbDone) }
func cmdCancel(env Env, args []string) int { return runTransition(env, args, verbCancel) }
func cmdReopen(env Env, args []string) int { return runTransition(env, args, verbReopen) }

// runTransition is the shared engine behind done, cancel, and reopen: it parses
// the invocation, asks the core to move the issue to the verb's target state,
// and renders whichever of the three answers came back — moved, already there,
// or refused by the current state.
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

	svc, err := open(env)
	if err != nil {
		return coreError(env, ref, err)
	}
	out, err := svc.Transition(ref, v.target)
	warnSkipped(env, out.Warnings)
	if err != nil {
		var illegal *core.IllegalTransitionError
		if errors.As(err, &illegal) {
			errf(env, v.reject, illegal.ID, illegal.From)
			return exitUsage
		}
		return coreError(env, ref, err)
	}

	// An unchanged outcome is the idempotent no-op: the issue was already at the
	// target, so nothing was written and `updated` stands.
	line := fmt.Sprintf(v.already, out.Issue.ID)
	if out.Changed {
		line = fmt.Sprintf(v.did, out.Issue.ID)
	}
	return reportIssue(env, format, out.Issue, line)
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
