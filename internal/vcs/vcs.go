// Package vcs is Busy Beaver's version-control integration. All VCS access
// goes through the System interface; Git is the reference adapter. A nil
// System is a valid, fully supported "no adapter" state in which Busy Beaver
// operates on files alone — no VCS is ever required.
package vcs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// System is the set of operations Busy Beaver may ask of a version-control
// system — the single seam where a concrete adapter plugs in.
type System interface {
	// Identity reports the actor name the VCS is configured with (for Git, the
	// user.name setting). It only seeds the interactive identity confirmation —
	// never a non-interactive actor, since an agent inherits the human's VCS
	// config and would claim work under the human's name. found is false when no
	// configured name is reachable; a non-nil error is reserved for an unexpected
	// failure interrogating a VCS that is present.
	Identity() (name string, found bool, err error)

	// Commit records the given paths as one atomic commit described by message
	// and returns the new commit's short revision. The commit is scoped to
	// exactly paths, leaving other staged or working-tree changes out and
	// undisturbed. A non-nil error means the commit could not be made; callers
	// treat that as non-fatal, since the already-written issue file is the
	// source of truth.
	Commit(paths []string, message string) (revision string, err error)
}

// Git is the reference VCS adapter. It shells out to the `git` command-line
// tool rather than linking a library, keeping the dependency at "git is on PATH".
type Git struct {
	// Dir is the directory git runs in; git's usual local→global→system config
	// cascade applies relative to it. Empty means the process's working directory.
	Dir string
}

// Identity reads `git config user.name`. An unset name, a missing repository,
// or a missing git binary all report found=false with a nil error — no seed
// available, not a failure.
func (g Git) Identity() (string, bool, error) {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = g.Dir
	out, err := cmd.Output()
	if err != nil {
		// Unset key, not a repo, or git not installed — all "no seed".
		return "", false, nil
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", false, nil
	}
	return name, true, nil
}

// Commit stages the given paths and records them as one atomic commit with
// message, returning the new commit's short revision. It commits only those
// pathspecs (git's --only mode), so unrelated staged or working-tree changes
// are neither committed nor disturbed. A vanished, never-tracked path is
// skipped rather than failing the commit; on an empty repository the commit
// becomes the initial one.
func (g Git) Commit(paths []string, message string) (string, error) {
	staged := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, err := g.run("add", "--", p); err != nil {
			if pathExists(p) {
				return "", fmt.Errorf("staging %s: %w", p, err)
			}
			continue // a vanished, never-tracked path: nothing to record
		}
		staged = append(staged, p)
	}
	if len(staged) == 0 {
		return "", errors.New("no paths to commit")
	}
	if _, err := g.run(append([]string{"commit", "--only", "-m", message, "--"}, staged...)...); err != nil {
		return "", fmt.Errorf("recording commit: %w", err)
	}
	rev, err := g.run("rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("reading commit revision: %w", err)
	}
	return strings.TrimSpace(rev), nil
}

// run executes a git subcommand in the adapter's directory, returning its
// stdout on success and, on failure, an error carrying git's own stderr.
func (g Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return stdout.String(), nil
}

// pathExists reports whether p is on disk, distinguishing a real staging
// failure from the benign case of a vanished, never-tracked path.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Fake is an in-memory System for tests: it reports a fixed identity and
// records the commits asked of it, touching no real repository. Only *Fake
// satisfies System; the zero *Fake reports no identity while still recording
// commits.
type Fake struct {
	Name  string // the actor name to report as the VCS identity
	Found bool   // whether an identity is available
	Err   error  // an optional error to simulate an interrogation failure

	// Rev is the revision Commit returns on success; when empty a fixed
	// stand-in is returned so callers always see a non-empty revision.
	Rev string
	// CommitErr, when non-nil, is returned from Commit to simulate a failed commit.
	CommitErr error
	// Commits records every Commit call in order.
	Commits []FakeCommit
}

// FakeCommit is one recorded Commit call.
type FakeCommit struct {
	Paths   []string
	Message string
}

// Identity returns the fake's configured values verbatim.
func (f *Fake) Identity() (string, bool, error) { return f.Name, f.Found, f.Err }

// Commit records the call, then returns the configured error or revision. The
// call is recorded even on a simulated failure, so a test can assert the
// commit was attempted.
func (f *Fake) Commit(paths []string, message string) (string, error) {
	f.Commits = append(f.Commits, FakeCommit{Paths: append([]string(nil), paths...), Message: message})
	if f.CommitErr != nil {
		return "", f.CommitErr
	}
	if f.Rev == "" {
		return "fakerev", nil
	}
	return f.Rev, nil
}
