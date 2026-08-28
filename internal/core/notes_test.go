package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The entry is appended in the shape the notes convention reads back, under a
// section the first note creates, and the body it was added to is untouched.
func TestNoteAppendsAnAttributedTimestampedEntry(t *testing.T) {
	root := newStore(t)
	seed(t, root, withBody(mkIssue("iss001", "Some work"), "Original **description** stays.\n"))

	out, err := openAt(t, root).Note("iss001", "alice", "  tried X; handing back  ")
	if err != nil {
		t.Fatalf("Note: %v", err)
	}
	if !out.Changed {
		t.Error("Changed = false, want true: a note always writes")
	}
	if !strings.Contains(out.Issue.Body, "Original **description** stays.") {
		t.Errorf("the description was not preserved:\n%s", out.Issue.Body)
	}
	notes := issue.ParseNotes(out.Issue.Body)
	if len(notes) != 1 {
		t.Fatalf("parsed %d notes, want 1:\n%s", len(notes), out.Issue.Body)
	}
	n := notes[0]
	if n.Author != "alice" {
		t.Errorf("author = %q, want the actor alice", n.Author)
	}
	if n.Text != "tried X; handing back" {
		t.Errorf("text = %q, want it trimmed", n.Text)
	}
	// The log and the file agree on when it happened: one instant, not two
	// readings of the clock.
	if !n.Time.Equal(writeTime) || !out.Issue.Updated.Equal(writeTime) {
		t.Errorf("note time/updated = %s/%s, want both %s", n.Time, out.Issue.Updated, writeTime)
	}
	if !out.Issue.Created.Equal(fixedTime) {
		t.Errorf("created = %s, want unchanged %s (a note is not a creation)", out.Issue.Created, fixedTime)
	}
	if out.Issue.State != issue.StateTodo {
		t.Errorf("state = %s, want unchanged todo (a note does not move state)", out.Issue.State)
	}

	// The entry round-trips through the file, not just the returned issue.
	detail, err := open(t, root).Get("iss001")
	if err != nil {
		t.Fatalf("Get after Note: %v", err)
	}
	if got := issue.ParseNotes(detail.Issue.Body); len(got) != 1 || got[0].Text != n.Text {
		t.Errorf("persisted notes = %v, want the one entry", got)
	}
}

// An entry records a moment rather than a state to converge on, so the same note
// twice is two notes, under one section.
func TestNoteIsNeverANoOp(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Handoff"))
	svc := openAt(t, root)

	if _, err := svc.Note("iss001", "alice", "same words"); err != nil {
		t.Fatalf("first Note: %v", err)
	}
	out, err := svc.Note("iss001", "alice", "same words")
	if err != nil {
		t.Fatalf("second Note: %v", err)
	}
	if !out.Changed {
		t.Error("Changed = false on a repeated note, want true: notes never collapse")
	}
	if got := issue.ParseNotes(out.Issue.Body); len(got) != 2 {
		t.Errorf("parsed %d notes after two calls, want 2:\n%s", len(got), out.Issue.Body)
	}
	if n := strings.Count(out.Issue.Body, issue.NotesHeading); n != 1 {
		t.Errorf("%d notes sections, want exactly 1 (the second entry appends under the first)", n)
	}
}

// A for-the-record note on finished work is legitimate: the log never gates on
// lifecycle.
func TestNoteIsAllowedInEveryState(t *testing.T) {
	for _, state := range []issue.State{issue.StateTodo, issue.StateInProgress, issue.StateDone, issue.StateCancelled} {
		t.Run(string(state), func(t *testing.T) {
			root := newStore(t)
			seed(t, root, withState(mkIssue("iss001", "Whatever state"), state))

			out, err := openAt(t, root).Note("iss001", "alice", "for the record")
			if err != nil {
				t.Fatalf("Note on a %s issue: %v", state, err)
			}
			if out.Issue.State != state {
				t.Errorf("state = %s, want unchanged %s", out.Issue.State, state)
			}
			if len(issue.ParseNotes(out.Issue.Body)) != 1 {
				t.Errorf("the note was not appended to the %s issue:\n%s", state, out.Issue.Body)
			}
		})
	}
}

// An entry with nothing in it, or attributed to nobody, is not a record of
// anything: both are refused with the file untouched.
func TestNoteRequiresTextAndAnActor(t *testing.T) {
	cases := []struct {
		name        string
		actor, text string
		field       string
	}{
		{"blank text", "alice", "   ", "note text"},
		{"no actor", "  ", "some words", "note author"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := newStore(t)
			seed(t, root, mkIssue("iss001", "Some work"))
			before := fileOf(t, root, "iss001", "Some work")

			_, err := openAt(t, root).Note("iss001", c.actor, c.text)
			var invalid *core.ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("Note = %v, want *ValidationError", err)
			}
			if invalid.Field != c.field {
				t.Errorf("refused field = %q, want %q", invalid.Field, c.field)
			}
			if after := fileOf(t, root, "iss001", "Some work"); after != before {
				t.Errorf("a refused note rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestNoteOnAnUnknownRefIsNotFound(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("iss001", "Only issue"))

	if _, err := openAt(t, root).Note("nope", "alice", "hi"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("Note on an unknown ref = %v, want ErrNotFound", err)
	}
}

// withBody gives a seeded issue a description to preserve.
func withBody(iss issue.Issue, body string) issue.Issue {
	iss.Body = body
	return iss
}
