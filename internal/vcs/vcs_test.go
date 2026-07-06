package vcs_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"beaver/internal/vcs"
)

// The Git adapter reads user.name from the repository it points at; git is
// isolated from the machine's real config so only the local identity is in play.
func TestGitIdentityReadsUserName(t *testing.T) {
	dir := gitRepo(t)
	run(t, dir, "config", "user.name", "Ada Lovelace")

	name, found, err := vcs.Git{Dir: dir}.Identity()
	if err != nil {
		t.Fatalf("Identity: unexpected error: %v", err)
	}
	if !found || name != "Ada Lovelace" {
		t.Errorf("Identity = (%q, %v), want (\"Ada Lovelace\", true)", name, found)
	}
}

// An unset user.name is reported as "no seed" (found=false), not an error.
func TestGitIdentityUnsetIsNotFound(t *testing.T) {
	dir := gitRepo(t) // user.name deliberately left unset

	name, found, err := vcs.Git{Dir: dir}.Identity()
	if err != nil {
		t.Fatalf("Identity: unexpected error: %v", err)
	}
	if found || name != "" {
		t.Errorf("Identity = (%q, %v), want (\"\", false)", name, found)
	}
}

// A Dir that is not a git repository yields no identity rather than an error.
func TestGitIdentityOutsideRepoIsNotFound(t *testing.T) {
	requireGit(t)
	isolateGit(t)

	_, found, err := vcs.Git{Dir: t.TempDir()}.Identity()
	if err != nil {
		t.Fatalf("Identity: unexpected error: %v", err)
	}
	if found {
		t.Error("expected no identity outside a repo with isolated config")
	}
}

// Commit records exactly the paths it is given, leaving unrelated staged and
// working-tree changes out of the commit and undisturbed.
func TestGitCommitIsScopedAndAtomic(t *testing.T) {
	dir := gitRepo(t)
	identify(t, dir)
	writeFile(t, dir, "issue.md", "todo\n")
	writeFile(t, dir, "unrelated.txt", "base\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "seed")

	// The issue change to commit, plus unrelated noise that must NOT ride along.
	writeFile(t, dir, "issue.md", "done\n")
	writeFile(t, dir, "unrelated.txt", "changed\n")
	run(t, dir, "add", "unrelated.txt") // pre-staged unrelated change

	rev, err := vcs.Git{Dir: dir}.Commit([]string{filepath.Join(dir, "issue.md")}, "mark issue done")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if rev == "" {
		t.Error("Commit returned an empty revision")
	}

	if got := gitOut(t, dir, "log", "-1", "--format=%s"); got != "mark issue done" {
		t.Errorf("commit subject = %q, want %q", got, "mark issue done")
	}
	if got := gitOut(t, dir, "show", "--name-only", "--format=", "HEAD"); got != "issue.md" {
		t.Errorf("commit touched %q, want only issue.md", got)
	}
	if got := gitOut(t, dir, "show", "HEAD:issue.md"); got != "done" {
		t.Errorf("committed issue content = %q, want the done version", got)
	}
	// The unrelated staged change is untouched: still staged, still uncommitted.
	if got := gitOut(t, dir, "diff", "--cached", "--name-only"); got != "unrelated.txt" {
		t.Errorf("staged-but-uncommitted files = %q, want the unrelated change left intact", got)
	}
}

// On a repository with no commits yet, Commit creates the initial commit
// rather than failing.
func TestGitCommitCreatesInitialCommit(t *testing.T) {
	dir := gitRepo(t)
	identify(t, dir)
	writeFile(t, dir, "first.md", "content\n")

	if _, err := (vcs.Git{Dir: dir}).Commit([]string{filepath.Join(dir, "first.md")}, "first"); err != nil {
		t.Fatalf("Commit on an unborn HEAD: %v", err)
	}
	if got := gitOut(t, dir, "log", "--format=%s"); got != "first" {
		t.Errorf("history = %q, want a single root commit %q", got, "first")
	}
}

// Passing both the surviving path and a removed one records a rename
// atomically (the deletion is staged too); a never-tracked, absent path is
// tolerated rather than failing the commit.
func TestGitCommitRecordsRenameAndToleratesUntrackedGhost(t *testing.T) {
	dir := gitRepo(t)
	identify(t, dir)
	writeFile(t, dir, "old.md", "v1\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "seed")

	// Simulate a filename canonicalization: old removed, new written.
	if err := os.Remove(filepath.Join(dir, "old.md")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "new.md", "v2\n")

	newP := filepath.Join(dir, "new.md")
	oldP := filepath.Join(dir, "old.md")
	ghost := filepath.Join(dir, "never-existed.md") // never tracked, absent on disk
	if _, err := (vcs.Git{Dir: dir}).Commit([]string{newP, oldP, ghost}, "canonicalize"); err != nil {
		t.Fatalf("Commit with a rename and a ghost path: %v", err)
	}
	// The tree holds the new file and not the old one.
	if got := gitOut(t, dir, "ls-tree", "-r", "--name-only", "HEAD"); got != "new.md" {
		t.Errorf("tree after rename commit = %q, want only new.md", got)
	}
}

// The Fake records what it is asked to commit, returns its configured
// revision, and records the attempt even when configured to fail.
func TestFakeCommit(t *testing.T) {
	f := &vcs.Fake{Rev: "abc1234"}
	rev, err := f.Commit([]string{"a.md", "b.md"}, "msg")
	if err != nil || rev != "abc1234" {
		t.Fatalf("Commit = (%q, %v), want (\"abc1234\", nil)", rev, err)
	}
	if len(f.Commits) != 1 || f.Commits[0].Message != "msg" || len(f.Commits[0].Paths) != 2 {
		t.Errorf("recorded commit = %+v, want one call with two paths and message %q", f.Commits, "msg")
	}

	failing := &vcs.Fake{CommitErr: errTest}
	if _, err := failing.Commit([]string{"x"}, "m"); err == nil {
		t.Error("Commit with CommitErr set returned no error")
	}
	if len(failing.Commits) != 1 {
		t.Errorf("a failed commit should still be recorded as attempted, got %d", len(failing.Commits))
	}
}

var errTest = errors.New("simulated commit failure")

// The Fake returns its configured values verbatim, and the zero value stands in
// for "no VCS configured".
func TestFake(t *testing.T) {
	if name, found, err := (&vcs.Fake{Name: "claude", Found: true}).Identity(); name != "claude" || !found || err != nil {
		t.Errorf("configured Fake = (%q, %v, %v)", name, found, err)
	}
	if _, found, _ := (&vcs.Fake{}).Identity(); found {
		t.Error("zero Fake should report found=false")
	}
}

// gitRepo makes an isolated, initialized git repository under a temp dir.
func gitRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	isolateGit(t)
	dir := t.TempDir()
	run(t, dir, "init")
	return dir
}

// isolateGit points git's global and system config at throwaway locations, so
// the machine's real ~/.gitconfig cannot leak a user.name into assertions.
func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(t.TempDir(), "gitconfig"))
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed; skipping the reference-adapter test")
	}
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// identify configures the committer identity git requires to make a commit.
func identify(t *testing.T, dir string) {
	t.Helper()
	run(t, dir, "config", "user.name", "Test Committer")
	run(t, dir, "config", "user.email", "test@example.com")
}

// gitOut runs a git command and returns its trimmed stdout, failing the test
// on error.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// writeFile writes content to name within dir, creating parents as needed.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
