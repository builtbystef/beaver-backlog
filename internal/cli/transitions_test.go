package cli_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

func TestTransitionsSetStateAndBumpUpdated(t *testing.T) {
	const created = "2026-06-27T18:30:00Z" // beavertest.DefaultNow
	const bumped = "2026-06-29T18:30:00Z"  // DefaultNow + 48h
	later := beavertest.DefaultNow.Add(48 * time.Hour)

	cases := []struct {
		verb string
		from issue.State
		want string
	}{
		{"done", issue.StateTodo, "done"},
		{"done", issue.StateInProgress, "done"},
		{"cancel", issue.StateTodo, "cancelled"},
		{"cancel", issue.StateInProgress, "cancelled"},
		{"reopen", issue.StateDone, "todo"},
		{"reopen", issue.StateCancelled, "todo"},
	}
	for _, c := range cases {
		t.Run(c.verb+"-from-"+string(c.from), func(t *testing.T) {
			h := beavertest.New(t).Init()
			seed(t, h, "iss001", "Some work", c.from, beavertest.DefaultNow)
			h.Clock.Set(later)

			out := h.DecodeJSON(h.MustRun(c.verb, "iss001").Stdout)
			if out["state"] != c.want {
				t.Errorf("state = %v, want %v", out["state"], c.want)
			}
			if out["updated"] != bumped {
				t.Errorf("updated = %v, want bumped to %v", out["updated"], bumped)
			}
			if out["created"] != created {
				t.Errorf("created = %v, want unchanged %v", out["created"], created)
			}

			// The change is on disk, not just echoed: re-read it through show.
			shown := h.DecodeJSON(h.MustRun("show", "iss001").Stdout)
			if shown["state"] != c.want || shown["updated"] != bumped {
				t.Errorf("persisted state/updated = %v/%v, want %v/%v", shown["state"], shown["updated"], c.want, bumped)
			}
		})
	}
}

// Cancelled is terminal but not completed: the issue stays readable and listed,
// never deleted.
func TestCancelIsDistinctFromDoneAndReadable(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "cnc111", "Abandon this", issue.StateTodo, beavertest.DefaultNow)
	seed(t, h, "don222", "Finish this", issue.StateTodo, beavertest.DefaultNow)

	cancelled := h.DecodeJSON(h.MustRun("cancel", "cnc111").Stdout)
	done := h.DecodeJSON(h.MustRun("done", "don222").Stdout)
	if cancelled["state"] != "cancelled" || done["state"] != "done" {
		t.Fatalf("states = %v / %v, want cancelled / done", cancelled["state"], done["state"])
	}
	if cancelled["state"] == done["state"] {
		t.Error("cancel and done produced the same state; they must be distinct")
	}

	shown := h.DecodeJSON(h.MustRun("show", "cnc111").Stdout)
	if shown["state"] != "cancelled" || shown["title"] != "Abandon this" {
		t.Errorf("cancelled issue is not readable as expected: %v", shown)
	}
	if got := listIDs(t, h, "--state", "cancelled"); !slices.Equal(got, []string{"cnc111"}) {
		t.Errorf("cancelled list = %v, want [cnc111]", got)
	}
}

// The clock is advanced before the rejected verb so an erroneous write (which
// would bump `updated`) could not slip past the byte-for-byte comparison.
func TestTransitionsRejectNonsensical(t *testing.T) {
	cases := []struct {
		verb string
		from issue.State
	}{
		{"done", issue.StateCancelled},    // a cancelled issue is closed; reopen first
		{"cancel", issue.StateDone},       // a done issue is closed; reopen first
		{"reopen", issue.StateInProgress}, // in-progress is not closed; nothing to reopen
	}
	for _, c := range cases {
		t.Run(c.verb+"-from-"+string(c.from), func(t *testing.T) {
			h := beavertest.New(t).Init()
			seed(t, h, "iss001", "Some work", c.from, beavertest.DefaultNow)
			file := "issues/" + h.IssueFiles()[0]
			before := h.ReadFile(file)
			h.Clock.Advance(time.Hour) // a stray write would change the bytes

			r := h.Run(c.verb, "iss001")
			if r.Code != 2 {
				t.Errorf("%s from %s exit = %d, want 2 (usage)", c.verb, c.from, r.Code)
			}
			if !strings.Contains(r.Stderr, "iss001") {
				t.Errorf("rejection should name the issue:\n%s", r.Stderr)
			}
			if r.Stdout != "" {
				t.Errorf("rejected transition wrote to stdout: %q", r.Stdout)
			}
			if after := h.ReadFile(file); after != before {
				t.Errorf("rejected transition modified the file (corruption):\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestTransitionsRedundantAreIdempotent(t *testing.T) {
	cases := []struct {
		verb  string
		state issue.State
	}{
		{"done", issue.StateDone},
		{"cancel", issue.StateCancelled},
		{"reopen", issue.StateTodo},
	}
	for _, c := range cases {
		t.Run(c.verb+"-on-"+string(c.state), func(t *testing.T) {
			h := beavertest.New(t).Init()
			seed(t, h, "iss001", "Some work", c.state, beavertest.DefaultNow)
			file := "issues/" + h.IssueFiles()[0]
			before := h.ReadFile(file)
			h.Clock.Advance(time.Hour) // a rewrite would bump updated and change bytes

			r := h.Run(c.verb, "iss001")
			if r.Code != 0 {
				t.Errorf("%s on %s exit = %d, want 0 (idempotent)", c.verb, c.state, r.Code)
			}
			out := h.DecodeJSON(r.Stdout)
			if out["state"] != string(c.state) {
				t.Errorf("state = %v, want unchanged %v", out["state"], c.state)
			}
			if out["updated"] != "2026-06-27T18:30:00Z" {
				t.Errorf("updated = %v, want unchanged (no bump on a no-op)", out["updated"])
			}
			if after := h.ReadFile(file); after != before {
				t.Errorf("idempotent no-op rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

// A transition rewrites the whole file, so the body and custom frontmatter keys
// must pass through untouched.
func TestTransitionPreservesBodyAndCustomFields(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/pre111-keep-me.md", `---
id: pre111
title: Keep me
state: todo
sprint: 7
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

The body with **markdown** and a fenced block:

    still here
`)
	h.Clock.Set(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))

	out := h.DecodeJSON(h.MustRun("done", "pre111").Stdout)
	if out["state"] != "done" {
		t.Errorf("state = %v, want done", out["state"])
	}
	if out["updated"] != "2026-07-01T12:00:00Z" {
		t.Errorf("updated = %v, want bumped", out["updated"])
	}

	// Re-read from disk to prove the write persisted, not just that the command
	// echoed the fields.
	shown := h.DecodeJSON(h.MustRun("show", "pre111").Stdout)
	if custom, _ := shown["custom"].(map[string]any); custom["sprint"] != float64(7) {
		t.Errorf("custom field sprint not preserved: %v", shown["custom"])
	}
	if body, _ := shown["body"].(string); !strings.Contains(body, "**markdown**") || !strings.Contains(body, "still here") {
		t.Errorf("body not preserved verbatim: %q", body)
	}
}

// Writing an issue whose filename has drifted (a hand-edit or merge) lands at the
// canonical name and removes the drifted file, so no second file shares the id.
func TestTransitionCanonicalizesDriftedFilename(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/wrong-name.md", `---
id: dft111
title: Drifted name
state: todo
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

body
`)
	h.Clock.Set(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))

	if out := h.DecodeJSON(h.MustRun("done", "dft111").Stdout); out["state"] != "done" {
		t.Fatalf("state = %v, want done", out["state"])
	}
	const want = "dft111-drifted-name.md"
	if files := h.IssueFiles(); !slices.Equal(files, []string{want}) {
		t.Errorf("issue files = %v, want exactly [%s]", files, want)
	}
	if shown := h.DecodeJSON(h.MustRun("show", "dft111").Stdout); shown["state"] != "done" {
		t.Errorf("resolved state = %v, want done", shown["state"])
	}
}

func TestTransitionsHumanOutput(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seed(t, h, "app111", "Do it", issue.StateTodo, beavertest.DefaultNow)
	seed(t, h, "red222", "Already", issue.StateDone, beavertest.DefaultNow)
	seed(t, h, "rej333", "Active", issue.StateInProgress, beavertest.DefaultNow)

	applied := h.MustRun("done", "app111").Stdout
	if strings.HasPrefix(strings.TrimSpace(applied), "{") {
		t.Errorf("expected a human line, got JSON:\n%s", applied)
	}
	if !strings.Contains(applied, "app111") || !strings.Contains(strings.ToLower(applied), "done") {
		t.Errorf("applied line missing id or state:\n%s", applied)
	}

	redundant := h.MustRun("done", "red222").Stdout
	if !strings.Contains(strings.ToLower(redundant), "already") {
		t.Errorf("redundant line should say 'already':\n%s", redundant)
	}

	rej := h.Run("reopen", "rej333")
	if rej.Code != 2 {
		t.Errorf("reopen of an in-progress issue exit = %d, want 2", rej.Code)
	}
	if !strings.Contains(rej.Stderr, "rej333") {
		t.Errorf("rejection should name the issue:\n%s", rej.Stderr)
	}
}

func TestTransitionsUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "x", issue.StateTodo, beavertest.DefaultNow)
	cases := [][]string{
		{"done"},                              // missing ref
		{"cancel", "a", "b"},                  // too many args
		{"reopen"},                            // missing ref
		{"done", "iss001", "--format", "xml"}, // invalid format
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

func TestTransitionsNotFoundAndNoStore(t *testing.T) {
	for _, verb := range []string{"done", "cancel", "reopen"} {
		h := beavertest.New(t).Init()
		if r := h.Run(verb, "zzzzzz"); r.Code != 3 {
			t.Errorf("%s of a missing issue exit = %d, want 3 (not-found)", verb, r.Code)
		}

		noStore := beavertest.New(t) // no init
		r := noStore.Run(verb, "x")
		if r.Code != 3 {
			t.Errorf("%s without a store exit = %d, want 3 (not-found)", verb, r.Code)
		}
		if !strings.Contains(r.Stderr, "init") {
			t.Errorf("%s without a store should suggest init:\n%s", verb, r.Stderr)
		}
	}
}
