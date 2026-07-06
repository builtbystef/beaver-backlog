package cli

import (
	"fmt"

	"beaver/internal/issue"
	"beaver/internal/store"
)

// commitCompletion records the just-written issue file(s) as one atomic commit
// when commit_on_done is enabled and a VCS adapter is present, returning the
// new revision. It never fails the command and returns "" when no commit was
// made: the default (disabled) is silent, and an opt-in that cannot be honored
// is a non-fatal stderr warning — the issue is done either way.
func commitCompletion(env Env, st *store.Store, iss issue.Issue, paths []string) string {
	cfg, err := st.Config()
	if err != nil {
		errf(env, "%s is done, but its project config could not be read to commit it: %v", iss.ID, err)
		return ""
	}
	if !cfg.CommitOnDone {
		return "" // the default: commit nothing
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

// commitMessage is the subject of a completion commit: the issue's id and
// title, so the commit points back at the issue it completed.
func commitMessage(iss issue.Issue) string {
	return fmt.Sprintf("Complete %s: %s", iss.ID, iss.Title)
}

// commitPaths is the set of files a completion commit stages: the issue's
// canonical file, plus the old path when the write canonicalized a drifted
// filename, so the rename is recorded as one change. The surviving file comes
// first.
func commitPaths(newPath, oldPath string) []string {
	if oldPath == "" || oldPath == newPath {
		return []string{newPath}
	}
	return []string{newPath, oldPath}
}
