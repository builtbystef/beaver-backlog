package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
	"github.com/builtbystef/beaver-backlog/internal/store"
)

// Ownership is advisory coordination, not a lock: claim reserves an issue for the
// current actor, assign delegates it, release clears it, and start begins work,
// auto-claiming an unowned issue. The guard behind claim and start refuses an issue
// held by a different actor unless --force, but is best-effort only — concurrent
// claims on two branches surface as a VCS merge conflict on the `assignee:` line
// rather than silent double-ownership. Start's guard is the core's; claim keeps
// its own until the command surface consolidates.

// claimDecision is the ownership guard's ruling on an issue's current assignee
// against the acting actor.
type claimDecision int

const (
	claimOwn    claimDecision = iota // already the actor's — a no-op, whatever --force says
	claimSets                        // set the actor as assignee: an unowned claim or a --force steal
	claimRefuse                      // held by a different actor and no --force to steal
)

// decideClaim applies the ownership guard: re-claiming one's own issue is a no-op,
// an unowned issue is claimed, and another actor's issue is refused unless force
// steals it. It is claim's copy of the rule the core applies to start, kept here
// until the command surface consolidates and claim goes away.
func decideClaim(current, actor string, force bool) claimDecision {
	switch {
	case current == actor:
		return claimOwn
	case current == "":
		return claimSets
	case force:
		return claimSets
	default:
		return claimRefuse
	}
}

// cmdClaim makes the current actor an issue's assignee, leaving the state
// untouched. It refuses an issue held by a different actor unless --force steals
// it; re-claiming one's own is an idempotent no-op that never rewrites the file.
func cmdClaim(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "claim")
	asFlag := fs.String("as", "", "claim as this actor (overrides identity detection)")
	forceFlag := fs.Bool("force", false, "steal an issue already claimed by another actor")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "claim requires an issue reference: beaver claim <ref>")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}
	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	// Resolve the actor only after the reference is known good, so a typo'd ref
	// fails fast without triggering an interactive identity prompt.
	me, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	switch decideClaim(iss.Assignee, me.name, *forceFlag) {
	case claimRefuse:
		errf(env, "%s is claimed by %s; use --force to steal it", iss.ID, iss.Assignee)
		return exitUsage
	case claimOwn:
		// An idempotent no-op that leaves the file (and updated) untouched.
		return reportIssue(env, format, iss, fmt.Sprintf("%s is already yours", iss.ID))
	}

	prev := iss.Assignee // "" for a fresh claim; the previous owner for a --force steal
	iss.Assignee = me.name
	line := fmt.Sprintf("Claimed %s for %s", iss.ID, me.name)
	if prev != "" {
		line = fmt.Sprintf("Claimed %s for %s (taken from %s)", iss.ID, me.name, prev)
	}
	return writeAndReport(env, st, path, iss, format, line)
}

// cmdAssign delegates an issue to a named actor. It carries no ownership guard —
// reassigning already-owned work is the point of delegation. State is untouched;
// assigning to the current assignee is an idempotent no-op.
func cmdAssign(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "assign")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 2 {
		errf(env, "assign requires an issue reference and an actor: beaver assign <ref> <actor>")
		return exitUsage
	}
	assignee := strings.TrimSpace(pos[1])
	if assignee == "" {
		errf(env, "assign requires a non-empty actor: beaver assign <ref> <actor>")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}
	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	if iss.Assignee == assignee {
		return reportIssue(env, format, iss, fmt.Sprintf("%s is already assigned to %s", iss.ID, assignee))
	}
	iss.Assignee = assignee
	return writeAndReport(env, st, path, iss, format, fmt.Sprintf("Assigned %s to %s", iss.ID, assignee))
}

// cmdRelease clears an issue's assignee, returning it to unowned. No guard, no
// state change; releasing an already-unassigned issue is an idempotent no-op.
func cmdRelease(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "release")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "release requires an issue reference: beaver release <ref>")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}
	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	if iss.Assignee == "" {
		return reportIssue(env, format, iss, fmt.Sprintf("%s has no assignee", iss.ID))
	}
	prev := iss.Assignee
	iss.Assignee = ""
	return writeAndReport(env, st, path, iss, format, fmt.Sprintf("Released %s (was %s)", iss.ID, prev))
}

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

// writeAndReport bumps `updated`, writes the modified issue back, and renders the
// result. No-op paths report without calling it, so `updated` is churned only on
// a real change.
func writeAndReport(env Env, st *store.Store, path string, iss issue.Issue, format output.Format, humanLine string) int {
	iss.Updated = env.Clock.Now().UTC().Truncate(time.Second)
	if _, err := st.Update(path, iss); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return reportIssue(env, format, iss, humanLine)
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
