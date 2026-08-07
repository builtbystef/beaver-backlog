package cli

import (
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// verb is a lifecycle transition as this CLI presents it: the state it sets and
// the wording for each way the core can answer. Every move is legal — whether a
// call changes anything is the core's business.
type verb struct {
	name   string      // command name, used in usage diagnostics
	target issue.State // state a successful transition sets

	did     string // human confirmation on a real transition; one %s (id)
	already string // human line when already at target; one %s (id)
}

var (
	verbDone = verb{
		name:    "done",
		target:  issue.StateDone,
		did:     "Marked %s done",
		already: "%s is already done",
	}
	verbCancel = verb{
		name:    "cancel",
		target:  issue.StateCancelled,
		did:     "Cancelled %s",
		already: "%s is already cancelled",
	}
	verbReopen = verb{
		name:    "reopen",
		target:  issue.StateTodo,
		did:     "Reopened %s",
		already: "%s is already todo",
	}
)

func cmdDone(env Env, args []string) int   { return runTransition(env, args, verbDone) }
func cmdCancel(env Env, args []string) int { return runTransition(env, args, verbCancel) }
func cmdReopen(env Env, args []string) int { return runTransition(env, args, verbReopen) }

// runTransition is the shared engine behind done, cancel, and reopen: it parses
// the invocation, asks the core to move the issue to the verb's target state,
// and renders whichever answer came back — moved, or already there.
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
		return coreError(env, err)
	}
	out, err := svc.Transition(ref, v.target)
	warnSkipped(env, out.Warnings)
	if err != nil {
		return coreError(env, err)
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
