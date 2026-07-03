// Package vcs is Busy Beaver's version-control port and its reference adapters.
//
// Busy Beaver never requires a version-control system (ADR 0006). When it does talk
// to one — to seed a human's identity now, and to record work as commits later —
// it does so only through the port defined here: an interface the core depends on,
// with concrete adapters plugged in behind it (ADR 0007). Git is the reference
// adapter Busy Beaver ships; third parties can implement the same port for other
// systems. A nil port is a valid, fully supported state meaning "no adapter":
// Busy Beaver then operates on files alone.
//
// This slice defines only the identity capability. The commit capability, and the
// opt-in commit-per-issue it enables, land with g2h6cs.
package vcs

import (
	"os/exec"
	"strings"
)

// Port is the version-control port: the operations Busy Beaver may ask of a
// version-control system. It is deliberately small and grows one capability at a
// time as features land (commit is next). Every VCS access in the core goes
// through this interface, so Git — or any future adapter — is the single seam
// where a concrete system plugs in.
type Port interface {
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
}

// Git is the reference VCS adapter: it drives the `git` command-line tool. It
// shells out rather than linking a Git library, keeping the dependency at "git is
// on PATH" and matching how humans and agents already drive the repository. This
// slice implements only Identity; Commit arrives with g2h6cs.
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
// error return stays nil in these ordinary cases, honoring the port's contract
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

// Fake is an in-memory Port for tests: it reports a fixed identity without
// touching a real repository, so identity-resolution tests stay deterministic and
// need no git. The zero Fake reports no identity (found=false), standing in for
// "no VCS configured".
type Fake struct {
	Name  string // the actor name to report as the VCS identity
	Found bool   // whether an identity is available
	Err   error  // an optional error to simulate an interrogation failure
}

// Identity returns the fake's configured values verbatim.
func (f Fake) Identity() (string, bool, error) { return f.Name, f.Found, f.Err }
