package cli

// This file holds the plumbing every command shares: list-flag helpers, opening
// the core, error-to-exit-code mapping, warning rendering, path prettifying, and
// output-format resolution.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// csvList collects a repeatable, comma-separated string flag into an ordered
// slice, e.g. `--label a,b --label c` yields [a, b, c].
type csvList struct{ values []string }

func (l *csvList) String() string { return strings.Join(l.values, ",") }

func (l *csvList) Set(v string) error {
	l.values = append(l.values, splitCSV(v)...)
	return nil
}

// splitCSV splits one value on commas into its trimmed, non-empty parts.
func splitCSV(v string) []string {
	var out []string
	for part := range strings.SplitSeq(v, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// open builds the core service every handler works through, resolving the store
// from the working directory. The env's core options travel with it, so the
// harness's fake clock and ID source reach the application the same way the real
// ones do. Callers map the error with coreError.
func open(env Env) (*core.Service, error) {
	return core.Open(env.WorkDir, env.CoreOptions...)
}

// coreError maps a failure from the core onto this CLI's diagnostic and exit
// code. The reference a failure is about travels in the error itself, so a
// command that resolved several — create, with its edges — still reports the one
// at fault.
func coreError(env Env, err error) int {
	// Both ref errors unwrap to ErrNotFound, so they must be matched before the
	// generic not-found branch swallows them.
	var ambiguous *core.AmbiguousRefError
	var unknown *core.UnknownRefError
	var invalid *core.ValidationError
	switch {
	case errors.Is(err, core.ErrNoStore):
		errf(env, "not a Beaver Backlog store; run `beaver init`")
		return exitNotFound
	case errors.As(err, &ambiguous):
		reportAmbiguous(env, ambiguous)
		return exitNotFound
	case errors.As(err, &unknown):
		errf(env, "no issue found matching %q", unknown.Ref)
		return exitNotFound
	case errors.Is(err, core.ErrNotFound):
		// A not-found that names no reference: there is nothing to quote back.
		errf(env, "no issue found")
		return exitNotFound
	case errors.As(err, &invalid):
		// Input the core refuses describes an issue that cannot exist: the
		// invocation was wrong, not the store.
		errf(env, "%v", invalid)
		return exitUsage
	default:
		errf(env, "%v", err)
		return exitError
	}
}

// reportAmbiguous explains that a reference names several issues and lists them
// so the user can pick one by its unique ID.
func reportAmbiguous(env Env, e *core.AmbiguousRefError) {
	errf(env, "%q is the slug of %d issues; use a full ID:", e.Ref, len(e.Matches))
	for _, iss := range e.Matches {
		// OneLine keeps a hand-edited multi-line title from breaking the
		// one-line-per-candidate listing.
		fmt.Fprintf(env.Stderr, "  %s  %s\n", iss.ID, output.OneLine(iss.Title))
	}
}

// warnSkipped reports the files a core call skipped as invalid. Warnings go to
// stderr, never stdout, so one cannot corrupt the JSON an agent parses.
func warnSkipped(env Env, warnings []core.Warning) {
	for _, w := range warnings {
		errf(env, "skipping invalid issue %s: %v", relPath(env.WorkDir, w.Path), w.Err)
	}
}

// relPath renders path relative to workDir when it sits inside it, falling back
// to the absolute path otherwise.
func relPath(workDir, path string) string {
	if rel, err := filepath.Rel(workDir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// resolveFormat picks the output format for a command, using the same agent
// registry that drives identity resolution to detect a machine consumer.
func resolveFormat(env Env, override string) (output.Format, error) {
	_, isAgent := knownAgent(env.Getenv)
	return output.Resolve(override, env.StdoutIsTTY, isAgent)
}
