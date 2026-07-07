package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"github.com/builtbystef/busy-beaver/internal/issue"
	"github.com/builtbystef/busy-beaver/internal/store"
)

// This file holds the editor plumbing edit and interactive create share: both
// hand a raw issue file to the user's $EDITOR and re-validate what came back.

// editorGate reports why the current session cannot host an interactive editor,
// or nil when it can. An editor needs a terminal on both ends — spawning one
// against a pipe would block forever — and an editor actually wired in.
func editorGate(env Env) error {
	switch {
	case !env.StdinIsTTY || !env.StdoutIsTTY:
		return errors.New("an interactive terminal is required")
	case env.Edit == nil:
		return errors.New("no editor is available; set the EDITOR or VISUAL environment variable")
	default:
		return nil
	}
}

// authorInEditor drives interactive create's editor path: it writes the seed
// issue as a skeleton file, opens $EDITOR on it, and reads the result back,
// requiring a non-empty title on top of the store's validation. Any failure
// after the skeleton is written cleans it out of the issues directory, so only
// a good result is returned, at its canonical path.
func authorInEditor(env Env, st *store.Store, seed issue.Issue) (issue.Issue, string, int) {
	skeleton, err := st.Write(seed)
	if err != nil {
		errf(env, "%v", err)
		return issue.Issue{}, "", exitError
	}
	seeded, err := issue.Marshal(seed) // the exact bytes Write just produced
	if err != nil {
		errf(env, "%v", err)
		return issue.Issue{}, "", exitError
	}
	// On any failure return, clean up the skeleton: delete it if untouched,
	// stash it as a draft if the human typed into it. The success path sets
	// committed first — by then the canonicalizing write has already renamed or
	// replaced the skeleton, so it must be left alone.
	committed := false
	defer func() {
		if !committed {
			abandonSkeleton(env, st, skeleton, seeded)
		}
	}()

	if err := env.Edit(skeleton); err != nil {
		errf(env, "editor failed: %v", err)
		return issue.Issue{}, "", exitError
	}
	edited, err := st.Read(skeleton)
	if err != nil {
		errf(env, "issue is not valid after editing: %v", err)
		return issue.Issue{}, "", exitError
	}
	// Refuse an id rewritten in the editor: were it another issue's id, the
	// canonicalizing write below would land on that issue's file and silently
	// replace it. The authoring is stashed as a draft like any other failure.
	if edited.ID != seed.ID {
		errf(env, "create minted the id %s; it cannot be changed in the editor (the file now says %s)", seed.ID, edited.ID)
		return issue.Issue{}, "", exitError
	}
	if strings.TrimSpace(edited.Title) == "" {
		errf(env, "create needs a title; none was set in the editor")
		return issue.Issue{}, "", exitError
	}

	// The title — and so the slug — has almost certainly changed, so rewrite at
	// the canonical <id>-<slug> name and drop the skeleton file.
	path, err := st.Update(skeleton, edited)
	if err != nil {
		errf(env, "%v", err)
		return issue.Issue{}, "", exitError
	}
	committed = true
	return edited, path, exitOK
}

// abandonSkeleton cleans up after a failed interactive authoring: an untouched
// skeleton is deleted, but one the human typed into is stashed under
// .beaver/drafts and its location reported — their words are never discarded.
// If even the stash fails, the file is left in place rather than destroyed.
func abandonSkeleton(env Env, st *store.Store, skeleton string, seeded []byte) {
	current, err := os.ReadFile(skeleton)
	if err != nil || bytes.Equal(current, seeded) {
		st.Delete(skeleton)
		return
	}
	dest, err := st.StashDraft(skeleton)
	if err != nil {
		errf(env, "could not stash your draft (%v); it remains at %s", err, relPath(env.WorkDir, skeleton))
		return
	}
	errf(env, "your draft is saved at %s", relPath(env.WorkDir, dest))
}
