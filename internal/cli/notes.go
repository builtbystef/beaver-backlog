package cli

import (
	"fmt"
	"strings"
	"time"

	"beaver/internal/issue"
)

// cmdNote appends an attributed, timestamped entry to an issue's append-only notes
// log — the coordination journal for human↔agent handoff (ADR 0012). The actor is
// resolved through the one identity chain every attributing command uses (--as
// overrides it), the time comes from the injected clock, and the entry is appended
// under the "## Notes" section of the body, which is created on the first note. The
// rest of the body is carried through untouched (ADR 0014).
//
// Notes are append-only: there is no edit, no reply, no no-op. Every call adds a new
// entry, so — unlike the idempotent verbs — note always writes and always bumps
// `updated`. A note is allowed on an issue in any state: a closed issue can still
// receive a for-the-record observation, and the log never gates on lifecycle.
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

	// Resolve who is writing only once the reference is known good, so a typo'd ref
	// fails fast (exit 3) without triggering an interactive identity prompt — the same
	// ordering claim and start use.
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
