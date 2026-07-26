package cli

// This file holds the plumbing every command shares: list-flag helpers, opening
// the core (and the legacy direct store discovery), error-to-exit-code mapping,
// warning rendering, path prettifying, and output-format resolution.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/output"
	"github.com/builtbystef/beaver-backlog/internal/store"
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

// dedupe returns in with later duplicates dropped, preserving first-seen order.
func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// open builds the core service the handlers work through, resolving the store
// from the working directory. The clock and ID source travel with it, so the
// harness's fakes reach the core the same way they reach the CLI. Callers map
// the error with coreError.
func open(env Env) (*core.Service, error) {
	return core.Open(env.WorkDir, core.WithClock(env.Clock), core.WithIDSource(env.NewID))
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

// warnSkipped reports the files a core read skipped as invalid, for a command
// that reads once.
func warnSkipped(env Env, warnings []core.Warning) { warnOnce(env)(warnings) }

// warnOnce builds the warning reporter for one command run. Warnings go to
// stderr, never stdout, so one cannot corrupt the JSON an agent parses, and the
// reporter dedupes by path so a command that goes to the core twice — claim
// reads before it writes — still names a broken file once.
func warnOnce(env Env) func([]core.Warning) {
	reported := make(map[string]bool)
	return func(warnings []core.Warning) {
		for _, w := range warnings {
			if reported[w.Path] {
				continue
			}
			reported[w.Path] = true
			errf(env, "skipping invalid issue %s: %v", relPath(env.WorkDir, w.Path), w.Err)
		}
	}
}

func storeError(env Env, err error) int {
	if errors.Is(err, store.ErrNoStore) {
		errf(env, "not a Beaver Backlog store; run `beaver init`")
		return exitNotFound
	}
	errf(env, "%v", err)
	return exitError
}

// discover finds the store from the working directory and wires it to warn on
// stderr about each invalid file it skips, so commands keep working on the
// valid issues without hiding a broken file. Callers map the error with
// storeError.
func discover(env Env) (*store.Store, error) {
	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return nil, err
	}
	st.OnWarn(warnInvalid(env))
	return st, nil
}

// warnInvalid builds the store's warning handler for one command run. It writes
// to stderr, never stdout, so a warning cannot corrupt the JSON an agent
// parses, and dedupes by path so a command that scans the store more than once
// still warns about a given file only once.
func warnInvalid(env Env) func(store.Warning) {
	seen := make(map[string]bool)
	return func(w store.Warning) {
		if seen[w.Path] {
			return
		}
		seen[w.Path] = true
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
