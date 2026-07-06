package cli

import (
	"fmt"
)

// cmdEdit opens an issue's raw file in $EDITOR for freeform hand-editing, then
// re-validates the result. It edits the file in place — the file is the single
// source of truth (ADR 0001), so what the human saves is what the issue becomes —
// and deliberately does not re-serialize it: a hand-edit is honored verbatim, and
// the untidiness it may leave (a filename that no longer matches a changed title)
// is lint for doctor, not a failure (ADR 0005). A non-interactive invocation
// refuses instead of hanging; an edit that leaves the file unparseable or invalid
// is reported, and the file is left as saved for the human to fix or re-edit.
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

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	// Refuse before touching the editor when the session cannot host one, so a
	// piped or agent run gets a clear error instead of a blocked process.
	if err := editorGate(env); err != nil {
		errf(env, "cannot edit %s: %v", iss.ID, err)
		return exitError
	}
	if err := env.Edit(path); err != nil {
		errf(env, "editor failed: %v", err)
		return exitError
	}

	// Re-read the file the human just saved through the store's usable-issue
	// contract. On a valid result, report the re-read issue (which reflects any
	// edited fields); on an invalid one, say what is wrong and leave the file alone.
	edited, err := st.Read(path)
	if err != nil {
		errf(env, "%s is no longer a valid issue after editing: %v", iss.ID, err)
		return exitError
	}
	return reportIssue(env, format, edited, fmt.Sprintf("Edited %s", edited.ID))
}
