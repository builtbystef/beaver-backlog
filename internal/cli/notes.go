package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/issue"
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

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}
	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	// Resolve the actor only once the reference is known good, so a typo'd ref
	// fails fast without triggering an interactive identity prompt.
	me, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	now := env.Clock.Now().UTC().Truncate(time.Second)
	iss.Body = issue.AppendNote(iss.Body, issue.Note{Author: me.name, Time: now, Text: text})
	iss.Updated = now
	if _, err := st.Update(path, iss); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return reportIssue(env, format, iss, fmt.Sprintf("Added note to %s as %s", iss.ID, me.name))
}
