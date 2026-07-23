package cli_test

import (
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestShowResolvesByIDSlugAndName(t *testing.T) {
	h := beavertest.New(t).Init()
	// A real create, so the ID is genuine; then reference it three ways.
	id := h.DecodeJSON(h.MustRun("create", "Wire up the resolver").Stdout)["id"].(string)
	const slug = "wire-up-the-resolver"

	byID := h.DecodeJSON(h.MustRun("show", id).Stdout)["id"]
	bySlug := h.DecodeJSON(h.MustRun("show", slug).Stdout)["id"]
	byName := h.DecodeJSON(h.MustRun("show", id+"-"+slug).Stdout)["id"]

	if byID != id || bySlug != id || byName != id {
		t.Errorf("resolved ids: id=%v slug=%v name=%v, want all %q", byID, bySlug, byName, id)
	}
}

func TestUnknownRefIsNotFound(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "aaa111", "Something", issue.StateTodo, beavertest.DefaultNow)

	r := h.Run("show", "nope")
	if r.Code != 3 {
		t.Errorf("unknown ref exit = %d, want 3 (not-found)", r.Code)
	}
	if r.Stdout != "" {
		t.Errorf("unknown ref wrote stdout: %q", r.Stdout)
	}
	if !strings.Contains(r.Stderr, "nope") {
		t.Errorf("stderr should name the ref:\n%s", r.Stderr)
	}
}

// A slug several issues share names no single issue, so it does not resolve — but
// the error lists the candidate IDs (id-sorted, so deterministic) to pick from.
func TestSharedSlugListsCandidates(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "dup222", "Fix bug", issue.StateTodo, beavertest.DefaultNow)
	seed(t, h, "dup111", "Fix bug", issue.StateTodo, beavertest.DefaultNow)

	r := h.Run("show", "fix-bug")
	if r.Code != 3 {
		t.Errorf("shared slug exit = %d, want 3 (not-found)", r.Code)
	}
	if r.Stdout != "" {
		t.Errorf("shared slug wrote stdout: %q", r.Stdout)
	}
	for _, want := range []string{"fix-bug", "dup111", "dup222", "full ID"} {
		if !strings.Contains(r.Stderr, want) {
			t.Errorf("stderr missing %q:\n%s", want, r.Stderr)
		}
	}
	if strings.Index(r.Stderr, "dup111") > strings.Index(r.Stderr, "dup222") {
		t.Errorf("candidates not sorted by id:\n%s", r.Stderr)
	}

	// Each remains reachable by its unique ID.
	if got := h.DecodeJSON(h.MustRun("show", "dup111").Stdout)["id"]; got != "dup111" {
		t.Errorf("show dup111 = %v, want dup111", got)
	}
}

// Slug resolution and not-found reporting live in the shared resolver, not in any
// one command, so every ref-taking verb must show both.
func TestEveryRefCommandUsesSharedResolver(t *testing.T) {
	for _, verb := range []string{"show", "done", "cancel", "reopen"} {
		t.Run(verb, func(t *testing.T) {
			// reopen acts on a closed issue; the others on an open one.
			state := issue.StateTodo
			if verb == "reopen" {
				state = issue.StateDone
			}

			// A slug resolves, so the command succeeds.
			h := beavertest.New(t).Init()
			seed(t, h, "solo11", "Lone issue", state, beavertest.DefaultNow)
			if r := h.Run(verb, "lone-issue"); r.Code != 0 {
				t.Errorf("%s by slug exit = %d, want 0\nstderr: %s", verb, r.Code, r.Stderr)
			}

			// An unknown ref is not-found for every verb.
			if r := h.Run(verb, "zzzzzz"); r.Code != 3 {
				t.Errorf("%s of an unknown ref exit = %d, want 3\nstderr: %s", verb, r.Code, r.Stderr)
			}
		})
	}
}
