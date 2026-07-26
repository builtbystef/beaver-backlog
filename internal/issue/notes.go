package issue

import (
	"regexp"
	"strings"
	"time"
)

// Notes are flat, append-only, attributed, timestamped entries kept in a
// conventioned section of an issue's Markdown body. Because the body is
// human-owned and preserved verbatim, notes are a convention layered over
// free-form Markdown, not a separate store; AppendNote and ParseNotes are the
// write and read halves of that convention, and they agree on one format so a
// note round-trips.
//
// The format is a "## Notes" section whose entries each open with an attribution
// line — "**<author>** — <timestamp>" — followed by the note text, up to the next
// attribution line or the end of the body:
//
//	## Notes
//
//	**stefan** — 2026-06-27T18:30:00Z
//
//	Tried the obvious fix; the parser still chokes on empty input. Handing back.

// NotesHeading is the Markdown header that introduces the notes section, the
// last section of a well-formed body.
const NotesHeading = "## Notes"

// Note is one entry in the log: who wrote it, when (UTC), and its free text.
type Note struct {
	Author string
	Time   time.Time
	Text   string
}

// noteAttribution matches an entry's attribution line, "**<author>** — <timestamp>",
// capturing the author and the raw timestamp. An en-dash or ASCII hyphen is
// accepted alongside the em-dash, since a human may hand-author an entry. A match
// alone does not make a boundary: ParseNotes also requires the timestamp to
// parse, so prose that merely looks bold-ish stays note text.
var noteAttribution = regexp.MustCompile(`^\*\*(.+?)\*\*[ \t]+[—–-][ \t]+(.+?)[ \t]*$`)

// AppendNote returns body with n appended as a new entry, creating the notes
// section when the body has none. Everything already in the body is preserved
// verbatim; the entry is rendered in the exact shape ParseNotes reads back.
func AppendNote(body string, n Note) string {
	entry := renderNote(n)
	// Trim only trailing whitespace so the join is clean; the body's own
	// content is otherwise untouched.
	base := strings.TrimRight(body, " \t\r\n")
	switch {
	case hasNotesSection(base):
		// The notes section is the body's last, so the entry goes at the end.
		return base + "\n\n" + entry
	case base == "":
		return NotesHeading + "\n\n" + entry
	default:
		return base + "\n\n" + NotesHeading + "\n\n" + entry
	}
}

// renderNote formats one entry: the attribution line, a blank line, then the
// text with trailing whitespace trimmed so entries pack uniformly.
func renderNote(n Note) string {
	return "**" + n.Author + "** — " + n.Time.UTC().Format(time.RFC3339) +
		"\n\n" + strings.TrimRight(n.Text, " \t\r\n")
}

// SetDescription returns body with everything above the notes section replaced
// by description, leaving the notes section byte-identical. A body with no notes
// section is description alone: the whole body was the description.
//
// The log is not the writer's to rewrite — an entry is another actor's words —
// so a caller replacing what an issue says never touches what was said about it.
func SetDescription(body, description string) string {
	start, ok := notesStart(body)
	if !ok {
		return description
	}
	notes := body[start:]
	// Trim only where the join happens, so the description meets the heading with
	// the one blank line every other writer leaves.
	desc := strings.TrimRight(description, " \t\r\n")
	if desc == "" {
		return notes
	}
	return desc + "\n\n" + notes
}

// ParseNotes extracts the structured entries from body's notes section, in file
// order. It returns nil when the body has no notes section.
//
// The parse is best-effort because the body is human-owned free-form Markdown:
// an entry begins only at an attribution line whose timestamp parses, anything
// else inside the section is treated as the current entry's text, and text
// before the first entry yields no entries.
func ParseNotes(body string) []Note {
	region, ok := notesRegion(body)
	if !ok {
		return nil
	}

	var notes []Note
	var cur *Note
	var text []string
	flush := func() {
		if cur != nil {
			cur.Text = strings.TrimSpace(strings.Join(text, "\n"))
			notes = append(notes, *cur)
		}
		cur, text = nil, nil
	}

	for line := range strings.SplitSeq(region, "\n") {
		if m := noteAttribution.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			if ts, err := parseTime(strings.TrimSpace(m[2])); err == nil {
				flush()
				cur = &Note{Author: strings.TrimSpace(m[1]), Time: ts}
				continue
			}
		}
		if cur != nil {
			text = append(text, line)
		}
	}
	flush()
	return notes
}

func hasNotesSection(body string) bool {
	_, ok := notesRegion(body)
	return ok
}

// notesRegion returns the text after the first "## Notes" header line, and
// whether such a header was found.
func notesRegion(body string) (string, bool) {
	start, ok := notesStart(body)
	if !ok {
		return "", false
	}
	// Everything after the header line is the section's entries.
	_, entries, _ := strings.Cut(body[start:], "\n")
	return entries, true
}

// notesStart returns the offset at which the notes section begins — the first
// byte of its "## Notes" header line — and whether the body has one. It is an
// offset rather than a line index so a caller can keep the section's bytes
// exactly as they are.
func notesStart(body string) (int, bool) {
	for pos := 0; pos <= len(body); {
		end := strings.IndexByte(body[pos:], '\n')
		line := body[pos:]
		if end >= 0 {
			line = body[pos : pos+end]
		}
		if strings.TrimSpace(line) == NotesHeading {
			return pos, true
		}
		if end < 0 {
			break
		}
		pos += end + 1
	}
	return 0, false
}
