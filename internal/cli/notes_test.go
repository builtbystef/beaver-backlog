package cli_test

import (
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/busy-beaver/internal/beavertest"
	"github.com/builtbystef/busy-beaver/internal/issue"
)

// The entry is exposed as a structured note, not just spliced into the body text.
func TestNoteAppendsAttributedTimestampedEntry(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Some work", issue.StateTodo, beavertest.DefaultNow)
	h.Clock.Advance(time.Hour) // the note (and the updated bump) take this later time

	out := h.DecodeJSON(h.MustRun("note", "iss001", "tried X; handing back", "--as", "alice").Stdout)

	notes := notesOf(t, out)
	if len(notes) != 1 {
		t.Fatalf("want 1 note, got %d: %v", len(notes), out["notes"])
	}
	n := notes[0]
	if n["author"] != "alice" {
		t.Errorf("note author = %v, want alice (the resolved actor)", n["author"])
	}
	if n["text"] != "tried X; handing back" {
		t.Errorf("note text = %v, want the appended text", n["text"])
	}
	if n["time"] != "2026-06-27T19:30:00Z" {
		t.Errorf("note time = %v, want the clock at note time", n["time"])
	}
	if out["updated"] != "2026-06-27T19:30:00Z" {
		t.Errorf("updated = %v, want bumped to the note time", out["updated"])
	}
	if out["created"] != "2026-06-27T18:30:00Z" {
		t.Errorf("created = %v, want unchanged by a note", out["created"])
	}
	if out["state"] != "todo" {
		t.Errorf("state = %v, want unchanged todo (a note does not move state)", out["state"])
	}
}

// Re-reading through show proves the note round-trips through the on-disk file,
// not just the in-memory issue.
func TestNotePreservesBodyAndCustomAndAppendsSection(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/pre111-keep-me.md", `---
id: pre111
title: Keep me
state: todo
sprint: 7
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

Original **description** stays.
`)
	h.MustRun("note", "pre111", "first observation", "--as", "alice")

	shown := h.DecodeJSON(h.MustRun("show", "pre111").Stdout)
	body, _ := shown["body"].(string)
	if !strings.Contains(body, "Original **description** stays.") {
		t.Errorf("description not preserved in the body:\n%q", body)
	}
	if !strings.Contains(body, "## Notes") || !strings.Contains(body, "first observation") {
		t.Errorf("notes section/entry not appended to the body:\n%q", body)
	}
	if custom, _ := shown["custom"].(map[string]any); custom["sprint"] != float64(7) {
		t.Errorf("custom field sprint not preserved through the note write: %v", shown["custom"])
	}
	if notes := notesOf(t, shown); len(notes) != 1 || notes[0]["author"] != "alice" || notes[0]["text"] != "first observation" {
		t.Errorf("structured notes after re-read = %v, want one by alice", shown["notes"])
	}
}

// That the second note sees the first proves the first was persisted and parsed
// back from disk.
func TestNoteIsAppendOnlyAcrossMultipleNotes(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Handoff", issue.StateTodo, beavertest.DefaultNow)

	h.MustRun("note", "iss001", "first", "--as", "alice")
	h.Clock.Advance(time.Hour)
	out := h.DecodeJSON(h.MustRun("note", "iss001", "second", "--as", "bob").Stdout)

	notes := notesOf(t, out)
	if len(notes) != 2 {
		t.Fatalf("want 2 notes after two note commands, got %d: %v", len(notes), out["notes"])
	}
	if notes[0]["author"] != "alice" || notes[0]["text"] != "first" {
		t.Errorf("first entry = %v, want alice/first (a later note must not overwrite it)", notes[0])
	}
	if notes[1]["author"] != "bob" || notes[1]["text"] != "second" {
		t.Errorf("second entry = %v, want bob/second", notes[1])
	}
	if body, _ := out["body"].(string); strings.Count(body, "## Notes") != 1 {
		t.Errorf("want exactly one Notes section (the second note appends under the first):\n%s", body)
	}
}

func TestNoteRendersInHumanShow(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)
	h.MustRun("note", "iss001", "handing back to you", "--as", "alice")

	out := h.MustRun("show", "iss001").Stdout
	for _, want := range []string{"Notes", "alice", "handing back to you"} {
		if !strings.Contains(out, want) {
			t.Errorf("human show missing %q:\n%s", want, out)
		}
	}
}

// With no --as, the shared identity chain honors BUSY_BEAVER_ACTOR (the agent/CI override).
func TestNoteAttributesThroughIdentityChain(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Env["BUSY_BEAVER_ACTOR"] = "ci-bot"
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("note", "iss001", "from CI").Stdout)
	if notes := notesOf(t, out); len(notes) != 1 || notes[0]["author"] != "ci-bot" {
		t.Errorf("note author = %v, want ci-bot resolved from BUSY_BEAVER_ACTOR", out["notes"])
	}
}

// A for-the-record note on a closed issue is legitimate; the log never gates on
// lifecycle.
func TestNoteAllowedOnClosedIssue(t *testing.T) {
	for _, st := range []issue.State{issue.StateDone, issue.StateCancelled} {
		t.Run(string(st), func(t *testing.T) {
			h := beavertest.New(t).Init()
			seed(t, h, "iss001", "Closed", st, beavertest.DefaultNow)

			r := h.Run("note", "iss001", "for the record", "--as", "alice")
			if r.Code != 0 {
				t.Fatalf("note on a %s issue exit = %d, want 0 (notes never gate on state)", st, r.Code)
			}
			out := h.DecodeJSON(r.Stdout)
			if out["state"] != string(st) {
				t.Errorf("state = %v, want unchanged %s", out["state"], st)
			}
			if notes := notesOf(t, out); len(notes) != 1 {
				t.Errorf("want the note appended to the closed issue, got %v", out["notes"])
			}
		})
	}
}

func TestNoteHumanConfirmationLine(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)

	out := h.MustRun("note", "iss001", "a quick note", "--as", "alice").Stdout
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("human note output should not be JSON:\n%s", out)
	}
	if !strings.Contains(out, "iss001") || !strings.Contains(strings.ToLower(out), "note") {
		t.Errorf("confirmation line should name the issue and mention the note:\n%s", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("confirmation line should name the attributed actor:\n%s", out)
	}
}

func TestNotePersistsToDiskInFormat(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)
	h.MustRun("note", "iss001", "durable", "--as", "alice")

	raw := h.ReadFile("issues/" + h.IssueFiles()[0])
	for _, want := range []string{"## Notes", "**alice** —", "2026-06-27T18:30:00Z", "durable"} {
		if !strings.Contains(raw, want) {
			t.Errorf("on-disk file missing %q:\n%s", want, raw)
		}
	}
}

func TestNoteUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "x", issue.StateTodo, beavertest.DefaultNow)
	before := h.ReadFile("issues/" + h.IssueFiles()[0])

	cases := [][]string{
		{"note"},                                    // missing ref and text
		{"note", "iss001"},                          // missing text
		{"note", "iss001", "   "},                   // blank text
		{"note", "iss001", "hi", "extra"},           // too many args
		{"note", "iss001", "hi", "--format", "xml"}, // invalid format
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
	if after := h.ReadFile("issues/" + h.IssueFiles()[0]); after != before {
		t.Errorf("a rejected note modified the issue file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestNoteNotFoundAndNoStore(t *testing.T) {
	h := beavertest.New(t).Init()
	if r := h.Run("note", "zzzzzz", "hi"); r.Code != 3 {
		t.Errorf("note on a missing issue exit = %d, want 3 (not-found)", r.Code)
	}

	noStore := beavertest.New(t) // no init
	r := noStore.Run("note", "zzzzzz", "hi")
	if r.Code != 3 {
		t.Errorf("note without a store exit = %d, want 3 (not-found)", r.Code)
	}
	if !strings.Contains(r.Stderr, "init") {
		t.Errorf("note without a store should suggest init:\n%s", r.Stderr)
	}
}

// notesOf extracts the structured notes array from a decoded issue result.
func notesOf(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	raw, ok := out["notes"].([]any)
	if !ok {
		t.Fatalf("result has no notes array: %v", out["notes"])
	}
	notes := make([]map[string]any, len(raw))
	for i, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			t.Fatalf("note %d is not a JSON object: %v", i, r)
		}
		notes[i] = m
	}
	return notes
}
