package issue

import (
	"strings"
	"testing"
	"time"
)

var noteT1 = time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
var noteT2 = time.Date(2026, 6, 27, 19, 5, 0, 0, time.UTC)

// AppendNote on an empty body creates the section and the first entry.
func TestAppendNoteCreatesSectionOnEmptyBody(t *testing.T) {
	got := AppendNote("", Note{Author: "stefan", Time: noteT1, Text: "first"})
	want := "## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nfirst"
	if got != want {
		t.Errorf("AppendNote on empty body =\n%q\nwant\n%q", got, want)
	}
}

// A body with a description keeps it verbatim and appends the notes section after it.
func TestAppendNotePreservesDescription(t *testing.T) {
	desc := "## What to build\n\nDo the thing.\n"
	got := AppendNote(desc, Note{Author: "stefan", Time: noteT1, Text: "started"})

	if !strings.HasPrefix(got, "## What to build\n\nDo the thing.") {
		t.Errorf("description not preserved at the head:\n%q", got)
	}
	if !strings.Contains(got, "\n\n## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nstarted") {
		t.Errorf("notes section not appended after the description:\n%q", got)
	}
}

// A second note appends under the existing section without a second header,
// one blank line between entries.
func TestAppendNoteAppendsUnderExistingSection(t *testing.T) {
	body := AppendNote("Desc.", Note{Author: "stefan", Time: noteT1, Text: "one"})
	body = AppendNote(body, Note{Author: "claude", Time: noteT2, Text: "two"})

	if n := strings.Count(body, NotesHeading); n != 1 {
		t.Errorf("want exactly one %q header, got %d:\n%s", NotesHeading, n, body)
	}
	firstAt := strings.Index(body, "one")
	secondAt := strings.Index(body, "two")
	if firstAt < 0 || secondAt < 0 || firstAt > secondAt {
		t.Errorf("entries not in append order:\n%s", body)
	}
	if !strings.Contains(body, "one\n\n**claude** — 2026-06-27T19:05:00Z\n\ntwo") {
		t.Errorf("second entry not separated from the first by one blank line:\n%q", body)
	}
}

// What AppendNote writes, ParseNotes reads back: author, time, and text, in
// order, across multiple entries.
func TestAppendParseRoundTrip(t *testing.T) {
	body := ""
	body = AppendNote(body, Note{Author: "stefan", Time: noteT1, Text: "tried X;\nsee commit abc"})
	body = AppendNote(body, Note{Author: "claude", Time: noteT2, Text: "picked it up"})

	notes := ParseNotes(body)
	if len(notes) != 2 {
		t.Fatalf("ParseNotes returned %d entries, want 2:\n%s", len(notes), body)
	}
	if notes[0].Author != "stefan" || !notes[0].Time.Equal(noteT1) || notes[0].Text != "tried X;\nsee commit abc" {
		t.Errorf("entry 0 = %+v, want stefan/%v with multi-line text", notes[0], noteT1)
	}
	if notes[1].Author != "claude" || !notes[1].Time.Equal(noteT2) || notes[1].Text != "picked it up" {
		t.Errorf("entry 1 = %+v, want claude/%v/\"picked it up\"", notes[1], noteT2)
	}
}

// No notes section at all yields nil, never a panic or a phantom entry.
func TestParseNotesNoSection(t *testing.T) {
	if notes := ParseNotes("Just a description, no notes here."); notes != nil {
		t.Errorf("ParseNotes without a section = %v, want nil", notes)
	}
	if notes := ParseNotes(""); notes != nil {
		t.Errorf("ParseNotes of empty body = %v, want nil", notes)
	}
}

// An empty notes section (header, nothing under it) has the section but no entries.
func TestParseNotesEmptySection(t *testing.T) {
	if notes := ParseNotes("Desc.\n\n## Notes\n"); len(notes) != 0 {
		t.Errorf("empty section parsed %d entries, want 0", len(notes))
	}
}

// An entry boundary requires a parsable timestamp, so prose that merely looks
// bold-and-dashed stays note text.
func TestParseNotesIgnoresNonTimestampAttributionLikeLines(t *testing.T) {
	body := "## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\n" +
		"Compared **A** — **B** in the table; B wins.\nSecond line."
	notes := ParseNotes(body)
	if len(notes) != 1 {
		t.Fatalf("want a single entry (the prose is not a boundary), got %d:\n%v", len(notes), notes)
	}
	if !strings.Contains(notes[0].Text, "B wins.") || !strings.Contains(notes[0].Text, "Second line.") {
		t.Errorf("prose that looks bold-ish should stay in the entry text: %q", notes[0].Text)
	}
}

// A hand-authored entry using an ASCII hyphen separator still parses.
func TestParseNotesToleratesHyphenSeparator(t *testing.T) {
	body := "## Notes\n\n**alice** - 2026-06-27T18:30:00Z\n\nhand-written note"
	notes := ParseNotes(body)
	if len(notes) != 1 || notes[0].Author != "alice" || notes[0].Text != "hand-written note" {
		t.Errorf("hyphen-separated hand-authored entry did not parse: %+v", notes)
	}
}

// The header is found after the description's own "##" sub-headers, and text
// before the first entry is dropped.
func TestParseNotesFindsSectionAfterOtherHeaders(t *testing.T) {
	body := "## What to build\n\nStuff.\n\n## Acceptance criteria\n\n- [ ] a\n\n" +
		"## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nthe only note"
	notes := ParseNotes(body)
	if len(notes) != 1 || notes[0].Text != "the only note" {
		t.Errorf("ParseNotes = %+v, want a single \"the only note\" entry", notes)
	}
}

// Replacing a description leaves the log alone: the notes section comes through
// byte-identical, however the new description is shaped.
func TestSetDescriptionPreservesTheNotesSection(t *testing.T) {
	notes := "## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nthe only note"
	body := "## What to build\n\nOld stuff.\n\n" + notes

	got := SetDescription(body, "New stuff.\n\n")
	if want := "New stuff.\n\n" + notes; got != want {
		t.Errorf("SetDescription =\n%q\nwant\n%q", got, want)
	}
}

// With no notes section the whole body is the description, so it is replaced
// outright and verbatim, since there is no join to tidy.
func TestSetDescriptionReplacesABodyWithoutNotes(t *testing.T) {
	if got := SetDescription("Old stuff.\n", "New stuff.\n"); got != "New stuff.\n" {
		t.Errorf("SetDescription = %q, want the new description verbatim", got)
	}
}

// An emptied description leaves the notes section as the whole body, with no
// blank lines left where the description was.
func TestSetDescriptionEmptyKeepsOnlyTheNotes(t *testing.T) {
	notes := "## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nthe only note"
	if got := SetDescription("Old stuff.\n\n"+notes, ""); got != notes {
		t.Errorf("SetDescription = %q, want the notes section alone", got)
	}
}

// Description is the half of the body that is about the issue rather than about
// the work on it: everything above the log, and none of it.
func TestDescriptionStopsAtTheNotesSection(t *testing.T) {
	body := "## What to build\n\nOld stuff.\n\n## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nthe only note"

	if got, want := Description(body), "## What to build\n\nOld stuff."; got != want {
		t.Errorf("Description =\n%q\nwant\n%q", got, want)
	}
}

// A body with no log is all description.
func TestDescriptionOfABodyWithoutNotes(t *testing.T) {
	if got, want := Description("Just prose.\n"), "Just prose."; got != want {
		t.Errorf("Description = %q, want %q", got, want)
	}
}

// A body that is nothing but the log has no description.
func TestDescriptionOfNotesAlone(t *testing.T) {
	body := "## Notes\n\n**stefan** — 2026-06-27T18:30:00Z\n\nthe only note"
	if got := Description(body); got != "" {
		t.Errorf("Description = %q, want empty", got)
	}
}
