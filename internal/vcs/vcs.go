// Package vcs is Busy Beaver's version-control integration and its reference adapters.
//
// Busy Beaver never requires a version-control system (ADR 0006). When it does talk
// to one — to seed a human's identity, and to record work as commits — it does so
// only through the System interface defined here: an interface the core depends on,
// with concrete adapters plugged in behind it — the ports-and-adapters design of
// ADR 0007. Git is the reference adapter Busy Beaver ships; third parties can
// implement the same interface for other systems. A nil System is a valid, fully
// supported state meaning "no adapter": Busy Beaver then operates on files alone.
//
// The integration spans two capabilities: reporting the configured identity, and
// recording work as a commit — the seam the opt-in commit-per-issue feature drives.
// Both remain optional; a nil System is still a fully supported "no adapter" state
// in which Busy Beaver operates on files alone.
package vcs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// System is Busy Beaver's version-control integration: the operations Busy Beaver may
// ask of a version-control system. It is deliberately small and grows one capability
// at a time as features land. Every VCS access in the core goes through this
// interface, so Git — or any future adapter — is the single seam where a concrete
// system plugs in.
type System interface {
	// Identity reports the actor name the VCS is configured with — for Git, the
	// `user.name` setting. Busy Beaver uses it only to seed the interactive
	// confirmation that establishes a human's saved identity (ADR 0008, ADR 0010);
	// it is never adopted as an actor non-interactively, because a coding agent
	// inherits the human's VCS config and would otherwise claim work under the
	// human's name.
	//
	// found is false when the adapter can reach no configured name — no VCS
	// present, or none set — in which case identity resolution falls through to a
	// prompt. A non-nil error is reserved for an unexpected failure interrogating a
	// VCS that is present; adapters fold the ordinary "nothing configured" case
	// into found=false rather than an error.
	Identity() (name string, found bool, err error)

	// Commit records the given paths as one atomic commit described by message and
	// returns the new commit's short revision. It is how Busy Beaver optionally drives
	// a VCS to record work — one completed issue as one commit (ADR 0007) — and is
	// invoked only when a project opts in; by default Busy Beaver commits nothing
	// (ADR 0006).
	//
	// The commit is scoped to exactly paths: the adapter stages those paths (adding
	// any not yet tracked, and staging the deletion of one a rename removed) and
	// commits only them, leaving any other staged or working-tree changes out of the
	// commit and undisturbed — so the result is a clean per-issue commit even when
	// the working tree is dirty. A non-nil error means the commit could not be made;
	// the caller treats that as non-fatal, since the issue file is already written
	// and is the source of truth (ADR 0006), and surfaces it as a warning.
	Commit(paths []string, message string) (revision string, err error)
}

// Git is the reference VCS adapter: it drives the `git` command-line tool. It
// shells out rather than linking a Git library, keeping the dependency at "git is
// on PATH" and matching how humans and agents already drive the repository. It
// implements the full interface: Identity reads the configured name, Commit records
// an atomic commit.
type Git struct {
	// Dir is the directory git runs in — the project working directory. git reads
	// its usual local→global→system config cascade relative to Dir, so a
	// repo-local user.name wins while a global one still seeds when Dir is outside
	// any repository. An empty Dir uses the process's working directory.
	Dir string
}

// Identity reads `git config user.name`. An unset name (git exits non-zero with no
// output) or a missing git binary is not a failure here — it just means "no seed
// available", reported as found=false so resolution falls through to a prompt. The
// error return stays nil in these ordinary cases, honoring the interface's contract
// that err signals only the unexpected.
func (g Git) Identity() (string, bool, error) {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = g.Dir
	out, err := cmd.Output()
	if err != nil {
		// Unset key, not a git repo, or git not installed — all "no seed", not a
		// hard failure the caller must surface.
		return "", false, nil
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", false, nil
	}
	return name, true, nil
}

// Commit stages the given paths and records them as one atomic commit with message,
// returning the new commit's short revision (e.g. "a1b2c3d").
//
// The commit is deliberately scoped to exactly paths. Each is staged on its own —
// `git add` picks up a new, modified, or (for a canonicalized rename) deleted
// tracked file — and the commit is a partial commit (git's --only mode) over just
// those pathspecs, so unrelated staged or working-tree changes are neither
// committed nor disturbed. A path that no longer exists and was never tracked
// matches nothing, so staging it is a no-op that is skipped rather than failing the
// commit; a real failure to stage a file that is present surfaces as an error. On
// an empty repository the commit is created as the initial one.
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

// run executes a git subcommand in the adapter's directory, returning its standard
// output on success and, on failure, an error carrying git's own stderr so the
// caller can surface a specific reason (Identity keeps its own simpler read because
// it deliberately swallows every failure as "no seed").
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

// pathExists reports whether p is present on disk, distinguishing a real staging
// failure (a file that is there but could not be added) from the benign case of a
// vanished, never-tracked path that git legitimately cannot match.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Fake is an in-memory System for tests: it reports a fixed identity and records the
// commits asked of it, touching no real repository, so tests stay deterministic and
// need no git. Because it records calls, it is used through a pointer — *Fake
// satisfies System, a plain Fake does not — and the zero *Fake reports no identity
// (found=false) while still recording commits, standing in for "no VCS configured".
type Fake struct {
	Name  string // the actor name to report as the VCS identity
	Found bool   // whether an identity is available
	Err   error  // an optional error to simulate an interrogation failure

	// Rev is the revision Commit returns on success; when empty a fixed stand-in is
	// returned so a caller always sees a non-empty revision.
	Rev string
	// CommitErr, when non-nil, is returned from Commit to simulate a failed commit.
	CommitErr error
	// Commits records every Commit call in order, so a test can assert exactly what
	// was committed and with what message.
	Commits []FakeCommit
}

// FakeCommit is one recorded Commit call.
type FakeCommit struct {
	Paths   []string
	Message string
}

// Identity returns the fake's configured values verbatim.
func (f *Fake) Identity() (string, bool, error) { return f.Name, f.Found, f.Err }

// Commit records the call, then returns the configured error if one is set or the
// configured revision otherwise. The call is recorded even on a simulated failure,
// so a test can assert the commit was attempted with the expected paths and message.
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
