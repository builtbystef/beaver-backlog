package cli

import (
	"beaver/internal/issue"
	"beaver/internal/output"
)

// cmdShow renders one issue resolved from a reference.
func cmdShow(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "show")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "show requires an issue reference: beaver show <ref>")
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

	// show both resolves one issue and derives its relationships over the whole
	// store, so it takes one snapshot and asks it twice rather than scanning the
	// files for each question.
	snap, err := st.Snapshot()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	iss, _, code := resolveRef(env, snap, ref)
	if code != exitOK {
		return code
	}

	// Enrich the view with the derived relationship facts show is the natural home
	// for: what this issue is waiting on, whether it is ready/blocked/stuck, and the
	// inverse edges (what it blocks, its children) that are never stored (ADR 0011).
	// Deriving them needs the whole store, so index the snapshot; the resolved
	// issue is among them.
	rel := issue.NewRelations(snap.Issues()).For(iss)

	if err := output.WriteIssueWithRelationship(env.Stdout, iss, rel, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}
