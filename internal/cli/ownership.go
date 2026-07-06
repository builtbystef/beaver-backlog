package cli

import (
	"fmt"
	"strings"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
)

// Ownership is advisory coordination, not a lock: claim reserves an issue for the
// current actor, assign delegates it, release clears it, and start begins work,
// auto-claiming an unowned issue. The guard behind claim and start refuses an issue
// held by a different actor unless --force, but is best-effort only — concurrent
// claims on two branches surface as a VCS merge conflict on the `assignee:` line
// rather than silent double-ownership.

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
// steals it. claim and start share it so both guard ownership identically.
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
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	// One snapshot serves the reference, the not-ready warning, and the JSON
	// readiness view. Starting changes only this issue's own state, never a
	// dependency's, so the pre-write snapshot also serves the post-start view:
	// For is handed the updated issue.
	snap, err := st.Snapshot()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	iss, path, code := resolveRef(env, snap, pos[0])
	if code != exitOK {
		return code
	}

	if iss.State == issue.StateDone || iss.State == issue.StateCancelled {
		errf(env, "%s is %s; reopen it first to start it", iss.ID, iss.State)
		return exitUsage
	}

	// Resolve the actor only after the ref and state checks pass, so a bad ref or
	// a closed issue never triggers an interactive identity prompt.
	me, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	decision := decideClaim(iss.Assignee, me.name, *forceFlag)
	if decision == claimRefuse {
		errf(env, "%s is claimed by %s; use --force to steal it", iss.ID, iss.Assignee)
		return exitUsage
	}

	rel := issue.NewRelations(snap.Issues())

	// The dependency warning fires only when work actually begins (todo moving to
	// in-progress) and is non-fatal; the same facts reach an agent through the
	// JSON relationships below.
	if iss.State == issue.StateTodo {
		if blockers := rel.BlockedOn(iss); len(blockers) > 0 {
			errf(env, "warning: %s is not ready: waiting on %s. Starting anyway.", iss.ID, formatBlockers(blockers))
		}
	}

	prev := iss.Assignee // previous owner, for a steal message
	stateChanged := iss.State != issue.StateInProgress
	claimSet := decision == claimSets

	// A pure no-op — already in-progress and already the actor's — is not
	// rewritten, so updated is not churned.
	humanLine := fmt.Sprintf("%s is already in progress", iss.ID)
	if stateChanged || claimSet {
		iss.State = issue.StateInProgress
		if claimSet {
			iss.Assignee = me.name
		}
		iss.Updated = env.Clock.Now().UTC().Truncate(time.Second)
		if _, err := st.Update(path, iss); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		humanLine = startLine(iss.ID, me.name, prev, stateChanged, claimSet)
	}

	// JSON additionally carries the derived readiness view, so an agent sees in
	// the start result whether the work it just began was blocked and on what.
	if format == output.Human {
		fmt.Fprintln(env.Stdout, humanLine)
		return exitOK
	}
	if err := output.WriteIssueWithRelationship(env.Stdout, iss, rel.For(iss), output.JSON); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
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

// startLine renders start's confirmation, covering the state move and any claim
// or steal.
func startLine(id, actor, prev string, stateChanged, claimSet bool) string {
	head := fmt.Sprintf("%s is already in progress", id)
	if stateChanged {
		head = "Started " + id
	}
	if !claimSet {
		return head
	}
	if prev != "" {
		return fmt.Sprintf("%s (claimed for %s, taken from %s)", head, actor, prev)
	}
	return fmt.Sprintf("%s (claimed for %s)", head, actor)
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
