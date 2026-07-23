package cli

// This file holds the plumbing every command shares: list-flag helpers, store
// discovery, error-to-exit-code mapping, path prettifying, and output-format
// resolution.

import (
	"errors"
	"path/filepath"
	"strings"

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
