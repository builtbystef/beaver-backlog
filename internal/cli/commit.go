package cli

import (
	"fmt"

	"beaver/internal/issue"
	"beaver/internal/store"
)

// commitCompletion honors the project's opt-in commit-per-issue when an issue is
// completed (ADR 0007). When commit_on_done is enabled and a VCS adapter is present,
// it records the just-written issue file(s) as one atomic commit and returns the new
// revision for the confirmation line.
//
// It is a convenience layered over a store that is already updated, so it never
// fails the command and returns "" whenever no commit was made: the default
// (disabled) is a silent no-op, and an explicit opt-in that cannot be honored — no
// adapter, an unreadable config, or a failed commit — is a loud but non-fatal stderr
// warning. The issue is done either way; the file is the source of truth and
// Busy Beaver never requires a VCS (ADR 0006).
func commitCompletion(env Env, st *store.Store, iss issue.Issue, paths []string) string {
	cfg, err := st.Config()
	if err != nil {
		errf(env, "%s is done, but its project config could not be read to commit it: %v", iss.ID, err)
		return ""
	}
	if !cfg.CommitOnDone {
		return "" // the default: Busy Beaver commits nothing (ADR 0006)
	}
	if env.VCS == nil {
		errf(env, "commit_on_done is enabled but no VCS adapter is configured; %s is done, nothing committed", iss.ID)
		return ""
	}
	rev, err := env.VCS.Commit(paths, commitMessage(iss))
	if err != nil {
		errf(env, "commit_on_done: could not commit %s (%v); it is marked done regardless", iss.ID, err)
		return ""
	}
	return rev
}

// commitMessage is the subject of a completion commit: the issue's id and title, so
// the commit reads clearly in `git log` and points back at the issue it completed.
func commitMessage(iss issue.Issue) string {
	return fmt.Sprintf("Complete %s: %s", iss.ID, iss.Title)
}

// commitPaths is the set of files a completion commit stages: the issue's canonical
// file, plus the old path when the write canonicalized a drifted filename (ADR 0005)
// — so the commit records the rename as one change rather than leaving a stale
// duplicate behind. The surviving file comes first.
func commitPaths(newPath, oldPath string) []string {
	if oldPath == "" || oldPath == newPath {
		return []string{newPath}
	}
	return []string{newPath, oldPath}
}
