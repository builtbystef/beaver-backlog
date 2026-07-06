package cli

import (
	"github.com/builtbystef/busy-beaver/internal/issue"
	"github.com/builtbystef/busy-beaver/internal/output"
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
	// store, so one snapshot answers both rather than scanning the files twice.
	snap, err := st.Snapshot()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	iss, _, code := resolveRef(env, snap, ref)
	if code != exitOK {
		return code
	}

	// Enrich the view with derived relationship facts: what this issue waits
	// on, whether it is ready/blocked/stuck, and the inverse edges (what it
	// blocks, its children) that are never stored.
	rel := issue.NewRelations(snap.Issues()).For(iss)

	if err := output.WriteIssueWithRelationship(env.Stdout, iss, rel, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}
