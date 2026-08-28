package cli

import (
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/output"
)

// cmdDelete removes an issue's file outright: the hard delete for junk, as
// distinct from cancel, which keeps the file as an abandoned record. It does
// not prompt: resolution already demands an exact reference, and a VCS (when
// present) retains the history as the undo.
func cmdDelete(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "delete")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "delete requires an issue reference: beaver delete <ref>")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	svc, err := open(env)
	if err != nil {
		return coreError(env, err)
	}
	deleted, err := svc.Delete(pos[0])
	warnSkipped(env, deleted.Warnings)
	if err != nil {
		return coreError(env, err)
	}

	if format == output.JSON {
		result := map[string]any{"id": deleted.Issue.ID, "title": deleted.Issue.Title, "deleted": true}
		if err := output.WriteJSON(env.Stdout, result); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintf(env.Stdout, "Deleted %s  %s\n  removed %s\n",
		deleted.Issue.ID, output.OneLine(deleted.Issue.Title), relPath(env.WorkDir, deleted.Path))
	return exitOK
}
