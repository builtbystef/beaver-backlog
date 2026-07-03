package vcs_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"beaver/internal/vcs"
)

// The Git adapter reads user.name from the repository it points at. The test is
// hermetic: it isolates git from the machine's real global/system config so the
// only identity in play is the one it sets locally.
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

// With no user.name set (and global/system config isolated away), git exits
// non-zero; the adapter reports this as "no seed", not an error.
func TestGitIdentityUnsetIsNotFound(t *testing.T) {
	dir := gitRepo(t) // a repo, but user.name deliberately left unset

	name, found, err := vcs.Git{Dir: dir}.Identity()
	if err != nil {
		t.Fatalf("Identity: unexpected error: %v", err)
	}
	if found || name != "" {
		t.Errorf("Identity = (%q, %v), want (\"\", false)", name, found)
	}
}

// A Dir that is not a git repository (and with global config isolated) yields no
// identity rather than an error — the graceful "no VCS to seed from" path.
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

// The Fake returns its configured values verbatim, and the zero value stands in
// for "no VCS configured".
func TestFake(t *testing.T) {
	if name, found, err := (vcs.Fake{Name: "claude", Found: true}).Identity(); name != "claude" || !found || err != nil {
		t.Errorf("configured Fake = (%q, %v, %v)", name, found, err)
	}
	if _, found, _ := (vcs.Fake{}).Identity(); found {
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

// isolateGit points git's global and system config at throwaway locations for the
// duration of the test, so the machine's real ~/.gitconfig cannot leak a
// user.name into these assertions.
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
