package cli

// This file holds the plumbing every command shares: flag-value helpers for the
// repeatable list flags, store discovery wired to the loud-warning contract, the
// error-to-exit-code mapping, path prettifying, and output-format resolution.

import (
	"errors"
	"path/filepath"
	"strings"

	"beaver/internal/output"
	"beaver/internal/store"
)

// csvList collects a repeatable, comma-separated string flag into an ordered
// slice, e.g. `--label a,b --label c` yields [a, b, c]. Each Set splits on commas
// and appends the trimmed, non-empty values in order; any later normalization
// (resolving references to canonical ids, deduping) happens once the values are
// known to be needed. It backs both --depends-on (issue references) and --label
// (free-form tags), which share this exact repeatable/comma-separated shape.
type csvList struct{ values []string }

func (l *csvList) String() string { return strings.Join(l.values, ",") }

func (l *csvList) Set(v string) error {
	l.values = append(l.values, splitCSV(v)...)
	return nil
}

// splitCSV splits one flag value on commas into its trimmed, non-empty parts. It
// is the shared normalizer behind csvList and the positional label arguments the
// label command takes, so `a, b` becomes [a, b] wherever a list of values is
// accepted.
func splitCSV(v string) []string {
	var out []string
	for part := range strings.SplitSeq(v, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// dedupe returns in with later duplicates dropped, preserving first-seen order —
// the canonical form for the multi-valued flags (labels, edges) where a repeat is
// redundant, not meaningful.
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
		errf(env, "not a Busy Beaver store; run `beaver init`")
		return exitNotFound
	}
	errf(env, "%v", err)
	return exitError
}

// discover finds the store from the working directory and wires it to report the
// invalid files it skips as loud stderr warnings (ADR 0005), so every command
// that reads the store degrades gracefully and visibly — it keeps working on the
// valid issues but never hides a broken file or lets it brick the command. It
// returns the same (store, error) as store.Discover; callers map the error with
// storeError.
func discover(env Env) (*store.Store, error) {
	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return nil, err
	}
	st.OnWarn(warnInvalid(env))
	return st, nil
}

// warnInvalid builds the store's warning handler for one command run: it prints
// each skipped file once, loudly, to stderr, naming the file and the specific
// problem (ADR 0005). It writes to stderr, never stdout, so a warning never
// corrupts the JSON an agent parses (ADR 0013). Dedup is by path, so a command
// that scans the store more than once (resolving a reference and then reading
// all) still warns a given file only once.
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

// relPath renders path relative to workDir when it sits inside it, for friendlier
// human output, and falls back to the absolute path otherwise.
func relPath(workDir, path string) string {
	if rel, err := filepath.Rel(workDir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// resolveFormat picks the output format for a command, centralizing agent
// detection: output no longer sniffs the environment itself but is told whether an
// agent is present, using the one registry (knownAgent) that also drives identity
// resolution (ADR 0010, ADR 0013).
func resolveFormat(env Env, override string) (output.Format, error) {
	_, isAgent := knownAgent(env.Getenv)
	return output.Resolve(override, env.StdoutIsTTY, isAgent)
}
