package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// Start is where ownership and the lifecycle meet: it begins work and claims the
// issue in one move, and it carries the only ownership guard left on the command
// surface. Assignment everywhere else is an unguarded update of the assignee
// field, because it is advisory coordination rather than a lock — the guard here
// is best-effort too, since concurrent claims on two branches surface as a merge
// conflict on the `assignee:` line rather than as silent double-ownership.

// cmdStart moves an issue to in-progress, auto-claiming an unowned one for the
// current actor and refusing one held by a different actor unless --force steals
// it. A closed issue is refused outright (reopen it first). Unmet dependencies
// produce a warning, never a refusal — starting blocked work is sometimes the
// right call.
func cmdStart(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "start")
	asFlag := fs.String("as", "", "start as this actor (overrides identity detection)")
	forceFlag := fs.Bool("force", false, "steal an issue already claimed by another actor")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "start requires an issue reference: beaver start <ref>")
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

	// The core takes the actor as a value, so identity is resolved before the
	// call — after the store is found, so no interactive prompt fires outside a
	// store.
	me, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	out, err := svc.Start(ref, me.name, *forceFlag)
	warnSkipped(env, out.Warnings)
	if err != nil {
		return startError(env, err)
	}

	// The dependency warning is non-fatal, and the core reports the dependencies
	// only when work actually begins; the same facts reach an agent through the
	// JSON relationships below.
	if len(out.UnmetDependencies) > 0 {
		errf(env, "warning: %s is not ready: waiting on %s. Starting anyway.",
			out.Issue.ID, formatBlockers(out.UnmetDependencies))
	}

	// JSON additionally carries the derived readiness view, so an agent sees in
	// the start result whether the work it just began was blocked and on what.
	if format == output.Human {
		fmt.Fprintln(env.Stdout, startLine(out))
		return exitOK
	}
	if err := output.WriteIssueWithRelationship(env.Stdout, out.Issue, out.Relationship, output.JSON); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}

// startError maps start's own refusals onto its diagnostics: a closed issue must
// be reopened first, and an issue another actor holds needs --force. Anything
// else is a failure every command reports the same way.
func startError(env Env, err error) int {
	var illegal *core.IllegalTransitionError
	var claimed *core.ClaimedError
	switch {
	case errors.As(err, &illegal):
		errf(env, "%s is %s; reopen it first to start it", illegal.ID, illegal.From)
		return exitUsage
	case errors.As(err, &claimed):
		errf(env, "%s is claimed by %s; use --force to steal it", claimed.ID, claimed.By)
		return exitUsage
	default:
		return coreError(env, err)
	}
}

// startLine renders start's confirmation from what the core actually did: the
// before-and-after pair says whether the state moved and whether the issue
// changed hands, so a pure no-op reports itself as one.
func startLine(out core.StartOutcome) string {
	head := fmt.Sprintf("%s is already in progress", out.Issue.ID)
	if out.Previous.State != out.Issue.State {
		head = "Started " + out.Issue.ID
	}
	prev := out.Previous.Assignee
	if prev == out.Issue.Assignee {
		return head
	}
	if prev != "" {
		return fmt.Sprintf("%s (claimed for %s, taken from %s)", head, out.Issue.Assignee, prev)
	}
	return fmt.Sprintf("%s (claimed for %s)", head, out.Issue.Assignee)
}

// formatBlockers renders unmet dependencies as a comma-joined list of
// "<id> (<state>)", or "<id> (missing)" for a dangling reference.
func formatBlockers(blockers []issue.Blocker) string {
	parts := make([]string, len(blockers))
	for i, b := range blockers {
		if b.Missing {
			parts[i] = b.ID + " (missing)"
		} else {
			parts[i] = fmt.Sprintf("%s (%s)", b.ID, b.State)
		}
	}
	return strings.Join(parts, ", ")
}
