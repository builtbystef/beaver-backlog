package issue

import (
	"regexp"
	"strings"
	"time"
)

// Notes are Busy Beaver's coordination journal: flat, append-only, attributed,
// timestamped entries kept in a conventioned section of an issue's Markdown body
// (ADR 0012), the record of a human↔agent handoff ("tried X, see commit abc;
// handing back"). Because the body is human-owned and preserved verbatim (ADR
// 0014), notes are a *convention* layered over free-form Markdown, not a separate
// store: `beaver note` appends an entry here, `show` prints the body (notes and
// all), and the JSON view parses this section back into structured entries. The two
// exported functions below are the write and read halves of that convention, and
// they agree on one format so a note round-trips.
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
//
//	**claude** — 2026-06-27T19:05:00Z
//
//	Picked it up — the guard was inverted. See commit abc123.

// NotesHeading is the Markdown header that introduces the notes section. A
// well-formed body is the description followed by this appended section, so it is
// the last thing in the body; both AppendNote and ParseNotes key off it.
const NotesHeading = "## Notes"

// Note is one entry in the log: who wrote it, when (UTC), and its free text.
type Note struct {
	Author string
	Time   time.Time
	Text   string
}

// noteAttribution matches an entry's attribution line, "**<author>** — <timestamp>",
// capturing the author (group 1) and the raw timestamp (group 2). It is the boundary
// between entries. The separator is written as an em-dash, but an en-dash or an ASCII
// hyphen is accepted too, since a human may hand-author an entry in this section (the
// body is theirs). A match alone does not make a boundary: ParseNotes also requires
// the captured timestamp to parse, so ordinary prose that merely looks bold-ish stays
// note text instead of being mistaken for a new entry.
var noteAttribution = regexp.MustCompile(`^\*\*(.+?)\*\*[ \t]+[—–-][ \t]+(.+?)[ \t]*$`)

// AppendNote returns body with n appended as a new entry, creating the notes section
// when the body has none. Everything already in the body is preserved verbatim; only
// the new entry — and, on the first note, the "## Notes" header — is added. The entry
// is rendered in the exact shape ParseNotes reads back, one blank line separating it
// from whatever precedes it, so the log stays uniformly spaced however many notes it
// holds.
func AppendNote(body string, n Note) string {
	entry := renderNote(n)
	// Trim only the trailing whitespace so the join is clean; the body's own content
	// (the description, earlier notes) is otherwise untouched.
	base := strings.TrimRight(body, " \t\r\n")
	switch {
	case hasNotesSection(base):
		// The section — and any earlier entries — already exist. A note only ever
		// appends, and the section is the body's last, so the new entry goes at the end.
		return base + "\n\n" + entry
	case base == "":
		return NotesHeading + "\n\n" + entry
	default:
		return base + "\n\n" + NotesHeading + "\n\n" + entry
	}
}

// renderNote formats one entry: the attribution line, a blank line, then the text
// with trailing whitespace trimmed so entries pack uniformly.
func renderNote(n Note) string {
	return "**" + n.Author + "** — " + n.Time.UTC().Format(time.RFC3339) +
		"\n\n" + strings.TrimRight(n.Text, " \t\r\n")
}

// ParseNotes extracts the structured entries from body's notes section, in file
// order (which, since notes only append, is chronological). It returns nil when the
// body has no notes section.
//
// The parse is best-effort by design: the body is human-owned free-form Markdown
// (ADR 0014), so ParseNotes recognizes the entries the convention writes and treats
// anything else inside the section as the current entry's text. An entry begins at an
// attribution line whose timestamp parses; any text before the first such line — an
// empty section, or one a hand-edit has mangled — simply yields no entries there.
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
				flush() // close the previous entry before starting this one
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

// hasNotesSection reports whether body already contains the notes header line.
func hasNotesSection(body string) bool {
	_, ok := notesRegion(body)
	return ok
}

// notesRegion returns the text after the first "## Notes" header line — where the
// entries live — and whether such a header was found. The returned region is only
// ever read (parsed), never written back, so reconstructing it line-wise is safe.
func notesRegion(body string) (string, bool) {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == NotesHeading {
			return strings.Join(lines[i+1:], "\n"), true
		}
	}
	return "", false
}
