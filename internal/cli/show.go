package cli

import (
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// cmdShow renders one issue resolved from a reference, together with the
// derived relationship facts the core computes for it: what it waits on,
// whether it is ready/blocked/stuck, and the inverse edges (what it blocks, its
// children) that are never stored.
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

	svc, err := open(env)
	if err != nil {
		return coreError(env, ref, err)
	}
	detail, err := svc.Get(ref)
	// The scan's warnings stand on their own: a skipped file is worth reporting
	// whether or not the reference then resolved.
	warnSkipped(env, detail.Warnings)
	if err != nil {
		return coreError(env, ref, err)
	}

	if err := output.WriteIssueWithRelationship(env.Stdout, detail.Issue, detail.Relationship, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}
