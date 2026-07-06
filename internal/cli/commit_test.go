package cli_test

import (
	"errors"
	"strings"
	"testing"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
	"beaver/internal/vcs"
)

func TestDoneCommitsWhenEnabled(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	enableCommitOnDone(h)
	fake := &vcs.Fake{Rev: "abc1234"}
	h.VCS = fake
	seed(t, h, "cmt001", "Wire the widget", issue.StateTodo, beavertest.DefaultNow)

	out := h.MustRun("done", "cmt001").Stdout

	if len(fake.Commits) != 1 {
		t.Fatalf("recorded %d commits, want exactly 1: %+v", len(fake.Commits), fake.Commits)
	}
	c := fake.Commits[0]
	if c.Message != "Complete cmt001: Wire the widget" {
		t.Errorf("commit message = %q, want it to name the issue and title", c.Message)
	}
	if len(c.Paths) != 1 || !strings.HasSuffix(c.Paths[0], "cmt001-wire-the-widget.md") {
		t.Errorf("commit paths = %v, want just the issue's canonical file", c.Paths)
	}
	if !strings.Contains(out, "abc1234") {
		t.Errorf("confirmation should report the committed revision:\n%s", out)
	}
}

// Even with an adapter present, an unset commit_on_done means done never commits.
func TestDoneDoesNotCommitByDefault(t *testing.T) {
	h := beavertest.New(t).Init() // default config: commit_on_done unset
	fake := &vcs.Fake{}
	h.VCS = fake
	seed(t, h, "def001", "No commit please", issue.StateTodo, beavertest.DefaultNow)

	r := h.Run("done", "def001")
	if r.Code != 0 {
		t.Fatalf("done exit = %d, want 0\nstderr: %s", r.Code, r.Stderr)
	}
	if len(fake.Commits) != 0 {
		t.Errorf("default done committed %d time(s), want 0", len(fake.Commits))
	}
}

// A VCS is never required: with commit_on_done enabled but no adapter, done still
// succeeds on disk and only warns.
func TestDoneWithoutAdapterIsNonFatal(t *testing.T) {
	h := beavertest.New(t).Init()
	enableCommitOnDone(h)
	h.VCS = nil // no adapter configured
	seed(t, h, "nov001", "Still gets done", issue.StateTodo, beavertest.DefaultNow)

	r := h.Run("done", "nov001")
	if r.Code != 0 {
		t.Fatalf("done with no adapter exit = %d, want 0\nstderr: %s", r.Code, r.Stderr)
	}
	if r.Stderr == "" {
		t.Error("enabling commit_on_done with no adapter should warn, but stderr was empty")
	}
	if shown := h.DecodeJSON(h.MustRun("show", "nov001").Stdout); shown["state"] != "done" {
		t.Errorf("issue state = %v, want done despite the missing adapter", shown["state"])
	}
}

// The file is the source of truth: a failed commit is a warning, not a command
// failure, and the issue is still done on disk.
func TestDoneCommitFailureIsNonFatal(t *testing.T) {
	h := beavertest.New(t).Init()
	enableCommitOnDone(h)
	fake := &vcs.Fake{CommitErr: errors.New("git exploded")}
	h.VCS = fake
	seed(t, h, "err001", "Done anyway", issue.StateTodo, beavertest.DefaultNow)

	r := h.Run("done", "err001")
	if r.Code != 0 {
		t.Fatalf("done with a failing commit exit = %d, want 0 (non-fatal)\nstderr: %s", r.Code, r.Stderr)
	}
	if len(fake.Commits) != 1 {
		t.Errorf("commit should have been attempted once, got %d", len(fake.Commits))
	}
	if !strings.Contains(r.Stderr, "err001") {
		t.Errorf("a failed commit should warn and name the issue:\n%s", r.Stderr)
	}
	if shown := h.DecodeJSON(h.MustRun("show", "err001").Stdout); shown["state"] != "done" {
		t.Errorf("issue state = %v, want done despite the commit failure", shown["state"])
	}
}

// Abandoning or restoring an issue is not a completion, so only done commits.
func TestCancelAndReopenDoNotCommit(t *testing.T) {
	h := beavertest.New(t).Init()
	enableCommitOnDone(h)
	fake := &vcs.Fake{}
	h.VCS = fake
	seed(t, h, "can001", "Abandon this", issue.StateTodo, beavertest.DefaultNow)
	seed(t, h, "reo001", "Bring back", issue.StateDone, beavertest.DefaultNow)

	h.MustRun("cancel", "can001")
	h.MustRun("reopen", "reo001")

	if len(fake.Commits) != 0 {
		t.Errorf("cancel/reopen recorded %d commit(s), want 0 — only done completes", len(fake.Commits))
	}
}

// A redundant done writes nothing, so it must record no commit either.
func TestRedundantDoneDoesNotCommit(t *testing.T) {
	h := beavertest.New(t).Init()
	enableCommitOnDone(h)
	fake := &vcs.Fake{}
	h.VCS = fake
	seed(t, h, "red001", "Already finished", issue.StateDone, beavertest.DefaultNow)

	if r := h.Run("done", "red001"); r.Code != 0 {
		t.Fatalf("redundant done exit = %d, want 0", r.Code)
	}
	if len(fake.Commits) != 0 {
		t.Errorf("a no-op done committed %d time(s), want 0", len(fake.Commits))
	}
}

// done's JSON carries an always-present "commit" key — the commit object when one
// was recorded, null otherwise — so an agent reads the revision from the result
// instead of asking the VCS, and never special-cases a missing key.
func TestDoneJSONCarriesCommit(t *testing.T) {
	h := beavertest.New(t).Init()
	enableCommitOnDone(h)
	h.VCS = &vcs.Fake{Rev: "abc1234"}
	seed(t, h, "cmt001", "Wire the widget", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("done", "cmt001").Stdout)
	commit, ok := out["commit"].(map[string]any)
	if !ok {
		t.Fatalf("done JSON commit = %v, want an object with the revision", out["commit"])
	}
	if commit["revision"] != "abc1234" {
		t.Errorf("commit revision = %v, want abc1234", commit["revision"])
	}

	// A redundant done commits nothing, but the key is still there — null — so the
	// shape is constant.
	redundant := h.DecodeJSON(h.MustRun("done", "cmt001").Stdout)
	if v, present := redundant["commit"]; !present || v != nil {
		t.Errorf("redundant done commit = %v (present=%v), want null", v, present)
	}
}

// With the opt-in off, the commit key is still present (null), so a consumer sees
// one shape regardless of project settings.
func TestDoneJSONCommitNullByDefault(t *testing.T) {
	h := beavertest.New(t).Init() // commit_on_done unset
	h.VCS = &vcs.Fake{}
	seed(t, h, "def001", "No commit please", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("done", "def001").Stdout)
	if v, present := out["commit"]; !present || v != nil {
		t.Errorf("done commit = %v (present=%v), want null when nothing was committed", v, present)
	}
}

// enableCommitOnDone turns the project-level opt-in on.
func enableCommitOnDone(h *beavertest.Harness) {
	h.WriteFile("config.yml", "format_version: 1\ncommit_on_done: true\n")
}
