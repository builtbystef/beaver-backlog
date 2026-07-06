package cli

import (
	"errors"
	"fmt"
	"strings"

	"beaver/internal/issue"
	"beaver/internal/store"
)

// resolver is the one capability resolveRef needs: turning a reference into an
// issue. Both *store.Store (a fresh scan per call) and *store.Snapshot (one scan
// answering many lookups) provide it, so a command picks whichever fits how often
// it asks.
type resolver interface {
	Resolve(ref string) (issue.Issue, string, error)
}

// resolveRef turns a user reference into a single issue through the store's shared
// resolver (store.Resolve) — the one path every issue-addressing command uses, so
// show, done, cancel, and reopen all accept the same references: a full ID, its
// slug, or an "<id>-<slug>" name, stale or canonical (ADR 0002). It maps a
// resolution failure to the right diagnostic and stable exit code and returns that
// code; on success it returns the issue, its path, and exitOK. Callers do:
//
//	iss, path, code := resolveRef(env, st, ref)
//	if code != exitOK {
//		return code
//	}
func resolveRef(env Env, st resolver, ref string) (iss issue.Issue, path string, code int) {
	iss, path, err := st.Resolve(ref)

	// SharedSlugError Unwraps to ErrNotFound, so it must be checked before the
	// generic not-found branch below would swallow it.
	var shared *store.SharedSlugError
	switch {
	case err == nil:
		return iss, path, exitOK
	case errors.As(err, &shared):
		reportSharedSlug(env, shared)
		return issue.Issue{}, "", exitNotFound
	case errors.Is(err, store.ErrNotFound):
		errf(env, "no issue found matching %q", ref)
		return issue.Issue{}, "", exitNotFound
	default:
		errf(env, "%v", err)
		return issue.Issue{}, "", exitError
	}
}

// resolveEdges turns the raw references from a relationship flag (--depends-on)
// into canonical issue ids, routing each through resolveRef so an edge accepts an
// id, a slug, or an <id>-<slug> name — the same references every other command
// takes. It dedupes by resolved id while preserving first-seen order, so repeats
// and two references to one issue collapse to a single stored edge. On the first
// reference that does not resolve it reports the failure (via resolveRef) and
// returns that non-OK code, so create refuses to store an edge to an issue that is
// not there: a typo fails fast instead of persisting as a dangling reference.
func resolveEdges(env Env, st resolver, refs []string) (ids []string, code int) {
	seen := make(map[string]bool)
	for _, ref := range refs {
		iss, _, code := resolveRef(env, st, ref)
		if code != exitOK {
			return nil, code
		}
		if !seen[iss.ID] {
			seen[iss.ID] = true
			ids = append(ids, iss.ID)
		}
	}
	return ids, exitOK
}

// reportSharedSlug explains that a slug names several issues and lists them, each
// as "id  title", so the user can pick one by its unique ID. store.Resolve already
// sorted the matches by ID, so the listing is deterministic. It counts as a
// not-found (there is no single issue), so the caller still exits exitNotFound.
func reportSharedSlug(env Env, e *store.SharedSlugError) {
	errf(env, "%q is the slug of %d issues; use a full ID:", e.Slug, len(e.Matches))
	for _, iss := range e.Matches {
		fmt.Fprintf(env.Stderr, "  %s  %s\n", iss.ID, flattenLine(iss.Title))
	}
}

// flattenLine collapses any newlines or tabs in s to single spaces so a
// hand-edited multi-line title cannot break the one-line-per-candidate listing.
func flattenLine(s string) string {
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ").Replace(s)
}
