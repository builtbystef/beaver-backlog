package cli

import (
	"fmt"
)

// cmdEdit opens an issue's raw file in $EDITOR for freeform hand-editing, then
// asks the core to read the result back. The file is deliberately not
// re-serialized: a hand-edit is honored verbatim, and any filename drift it
// leaves is lint for doctor, not a failure. A non-interactive invocation refuses
// instead of hanging; an edit that leaves the file invalid is reported and the
// file left as saved.
func cmdEdit(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "edit")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "edit requires an issue reference: beaver edit <ref>")
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
	target, err := svc.Editable(pos[0])
	warnSkipped(env, target.Warnings)
	if err != nil {
		return coreError(env, err)
	}

	// Refuse when the session cannot host an editor, so a piped or agent run
	// gets a clear error instead of a blocked process.
	if err := editorGate(env); err != nil {
		errf(env, "cannot edit %s: %v", target.Issue.ID, err)
		return exitError
	}
	if err := env.Edit(target.Path); err != nil {
		errf(env, "editor failed: %v", err)
		return exitError
	}

	// On an invalid result, say what is wrong and leave the file alone for the
	// human to fix or re-edit.
	edited, err := svc.Reread(target.Path)
	if err != nil {
		errf(env, "%s is no longer a valid issue after editing: %v", target.Issue.ID, err)
		return exitError
	}
	return reportIssue(env, format, edited, fmt.Sprintf("Edited %s", edited.ID))
}
