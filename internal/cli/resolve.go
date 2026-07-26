package cli

import (
	"errors"
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
	"github.com/builtbystef/beaver-backlog/internal/store"
)

// resolver turns a reference into an issue. Both *store.Store (a fresh scan per
// call) and *store.Snapshot (one scan answering many lookups) provide it.
type resolver interface {
	Resolve(ref string) (issue.Issue, string, error)
}

// resolveRef turns a user reference — a full ID, a slug, or an "<id>-<slug>"
// name, stale or canonical — into a single issue. It maps a resolution failure
// to a diagnostic and its exit code; on success it returns the issue, its path,
// and exitOK.
func resolveRef(env Env, st resolver, ref string) (iss issue.Issue, path string, code int) {
	iss, path, err := st.Resolve(ref)

	// SharedSlugError unwraps to ErrNotFound, so it must be checked before the
	// generic not-found branch swallows it.
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

// reportSharedSlug explains that a slug names several issues and lists them so
// the user can pick one by its unique ID.
func reportSharedSlug(env Env, e *store.SharedSlugError) {
	errf(env, "%q is the slug of %d issues; use a full ID:", e.Slug, len(e.Matches))
	for _, iss := range e.Matches {
		// OneLine keeps a hand-edited multi-line title from breaking the
		// one-line-per-candidate listing.
		fmt.Fprintf(env.Stderr, "  %s  %s\n", iss.ID, output.OneLine(iss.Title))
	}
}
