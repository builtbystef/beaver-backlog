package cli

import (
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/output"
)

// cmdDelete removes an issue's file outright — the hard delete for junk, as
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

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	if err := st.Delete(path); err != nil {
		errf(env, "%v", err)
		return exitError
	}

	if format == output.JSON {
		if err := output.WriteJSON(env.Stdout, map[string]any{"id": iss.ID, "title": iss.Title, "deleted": true}); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintf(env.Stdout, "Deleted %s  %s\n  removed %s\n", iss.ID, output.OneLine(iss.Title), relPath(env.WorkDir, path))
	return exitOK
}
