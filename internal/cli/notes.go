package cli

import (
	"fmt"
	"strings"
)

// cmdNote appends an attributed, timestamped entry under the issue body's
// "## Notes" section, creating the section on the first note. Notes are
// append-only — every call writes and bumps `updated` — and are allowed on an
// issue in any state, including closed ones.
func cmdNote(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "note")
	asFlag := fs.String("as", "", "attribute the note to this actor (overrides identity detection)")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 2 {
		errf(env, "note requires an issue reference and text: beaver note <ref> \"<text>\"")
		return exitUsage
	}
	text := strings.TrimSpace(pos[1])
	if text == "" {
		errf(env, "note text must not be empty")
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

	// The core takes the actor as a value, so identity is resolved before the
	// call — after the store is found, so no interactive prompt fires outside a
	// store.
	me, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	out, err := svc.Note(pos[0], me.name, text)
	warnSkipped(env, out.Warnings)
	if err != nil {
		return coreError(env, err)
	}
	return reportIssue(env, format, out.Issue, fmt.Sprintf("Added note to %s as %s", out.Issue.ID, me.name))
}
