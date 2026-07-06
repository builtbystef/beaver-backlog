package cli

import (
	"fmt"
	"strings"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
)

// Ownership is advisory coordination, not a lock (ADR 0009). The single `assignee`
// field is the primitive: claim reserves an issue for the current actor, assign
// delegates it to a named one, release clears it, and start begins work —
// auto-claiming an unowned issue as it moves to in-progress. The guard behind claim
// and start is best-effort against the local working tree: it refuses an issue
// already held by a *different* actor unless --force, but it is never a true lock,
// so genuinely concurrent claims on two branches surface as a VCS merge conflict on
// the `assignee:` line rather than silent double-ownership.

// claimDecision is how the ownership guard resolves an issue's current assignee
// against the acting actor: leave it (already theirs), set it (unowned, or a
// --force steal), or refuse (held by another with no --force).
type claimDecision int

const (
	claimOwn    claimDecision = iota // already the actor's — a no-op, whatever --force says
	claimSets                        // set the actor as assignee: an unowned claim or a --force steal
	claimRefuse                      // held by a different actor and no --force to steal
)

// decideClaim applies the best-effort ownership guard (ADR 0009): re-claiming one's
// own issue is a no-op; an unowned issue is claimed; another actor's issue is
// refused unless force steals it. claim and start share it so both guard ownership
// identically.
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

// cmdClaim makes the current actor an issue's assignee to signal "this is mine",
// leaving the state untouched — claiming reserves, it does not start (ADR 0009).
// The actor is resolved through the one identity chain every attributing command
// uses (--as overrides it). The guard refuses an issue already held by a different
// actor unless --force steals it; re-claiming one's own is an idempotent no-op that
// never rewrites the file.
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

	// Resolve who is acting only once the reference is known good, so a typo'd ref
	// fails fast (exit 3) without triggering an interactive identity prompt.
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
		// Already the actor's: an idempotent no-op that leaves the file (and updated) untouched.
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

// cmdAssign delegates an issue to a named actor — the explicit counterpart to
// claim. Where claim grabs an issue for whoever is acting and guards against
// clobbering another's, assign sets an arbitrary assignee outright and so
// deliberately carries no ownership guard: reassigning already-owned work is the
// whole point of delegation. State is untouched; assigning to the current assignee
// is an idempotent no-op.
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

// cmdRelease clears an issue's assignee, returning it to unowned. Like assign it is
// an explicit administrative action with no guard and no state change; releasing an
// already-unassigned issue is an idempotent no-op.
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

// cmdStart moves an issue to in-progress and begins work on it, folding the
// ownership guard into the state transition. An unowned issue is auto-claimed for
// the current actor as it starts — the only implicit assignment (ADR 0009); an
// issue held by a different actor is refused unless --force steals it; and a closed
// issue is refused outright (reopen it first). start is dependency-aware but never
// gated by it: moving a not-ready todo to in-progress first warns, naming the
// unfinished dependencies, then starts anyway — the same advisory stance as the
// ownership guard and as "blocked is a derived view, never enforced at write time"
// (ADR 0011). Starting blocked work is sometimes the right call (prep work, or
// knowledge the graph lacks), so start informs rather than refuses.
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

	// One snapshot answers everything start asks of the store: the reference, the
	// advisory not-ready warning, and the JSON readiness view — one scan, not one
	// per question (the same pattern show and create use). Starting an issue changes
	// only its own state, never any dependency's, so the pre-write snapshot also
	// serves the post-start view: For is handed the updated issue, so its readiness
	// flags reflect the new state.
	snap, err := st.Snapshot()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	iss, path, code := resolveRef(env, snap, pos[0])
	if code != exitOK {
		return code
	}

	// A closed issue cannot be started; it must be reopened first — the same refusal
	// the transition verbs give, for the same reason: start's sources are the open
	// states (todo, in-progress), so a done or cancelled issue is the rejected one.
	if iss.State == issue.StateDone || iss.State == issue.StateCancelled {
		errf(env, "%s is %s; reopen it first to start it", iss.ID, iss.State)
		return exitUsage
	}

	// Resolve who is acting only now — after the ref and state checks pass — so a bad
	// ref or a closed issue never triggers an interactive identity prompt.
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

	// Advisory dependency warning: only when actually moving a todo to in-progress
	// (the moment work begins). It names the unmet dependencies and is non-fatal —
	// start never refuses on this basis (ADR 0011). This is the human-facing half of
	// the signal; the same facts reach an agent through the JSON relationships below.
	if iss.State == issue.StateTodo {
		if blockers := rel.BlockedOn(iss); len(blockers) > 0 {
			errf(env, "warning: %s is not ready: waiting on %s. Starting anyway.", iss.ID, formatBlockers(blockers))
		}
	}

	prev := iss.Assignee // previous owner, for a steal message
	stateChanged := iss.State != issue.StateInProgress
	claimSet := decision == claimSets

	// A pure no-op — already in-progress and already the actor's — is not rewritten,
	// so updated is not churned (matching the idempotent transitions). Anything that
	// moves the state or the assignee is written with a bumped updated.
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

	// Human output is the concise confirmation line (the warning already spoke).
	// JSON additionally carries the derived readiness view — the same issue+
	// relationships shape show emits — so an agent sees, in the start result itself,
	// whether the work it just began was blocked and on exactly what.
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

// writeAndReport bumps `updated` from the injected clock (matching create and the
// transitions), writes the modified issue back through its resolved path, and
// renders the result. It is the shared write half of the ownership verbs; a no-op
// path reports without ever calling it, so `updated` is churned only on a real
// change. The read-modify-write carries the body and any custom frontmatter keys
// through untouched (ADR 0014).
func writeAndReport(env Env, st *store.Store, path string, iss issue.Issue, format output.Format, humanLine string) int {
	iss.Updated = env.Clock.Now().UTC().Truncate(time.Second)
	if _, err := st.Update(path, iss); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return reportIssue(env, format, iss, humanLine)
}

// startLine renders start's human confirmation, folding together the two things it
// may have done: moved the state (Started, versus already in progress) and
// auto-claimed or stolen the issue.
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

// formatBlockers renders unmet dependencies for the start warning as a comma-joined
// list of "<id> (<state>)", or "<id> (missing)" for a dangling reference — the same
// vocabulary show uses for what an issue is waiting on.
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
