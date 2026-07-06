package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"beaver/internal/issue"
	"beaver/internal/store"
)

// This file holds the editor plumbing edit and interactive create share. Both
// editing paths hand a raw issue file to the user's $EDITOR and then re-validate
// what came back (ADR 0005): the file is the issue, so freeform hand-editing is a
// first-class way to change one, guarded only by "the result must still be a
// usable issue."

// editorGate reports why the current session cannot host an interactive editor, or
// nil when it can. A full-screen editor needs a terminal on both ends — spawning
// one against a pipe or an agent would block forever on input that never comes — so
// edit and interactive create refuse rather than hang (the same interactivity line
// ADR 0010 draws for identity setup), and both need an editor actually wired in.
func editorGate(env Env) error {
	switch {
	case !env.StdinIsTTY || !env.StdoutIsTTY:
		return errors.New("an interactive terminal is required")
	case env.Edit == nil:
		return errors.New("no editor is available; set $EDITOR or $VISUAL")
	default:
		return nil
	}
}

// authorInEditor drives interactive create's editor path: it writes the new issue
// as a skeleton file (its machine-owned frontmatter filled — a minted id, todo
// state, timestamps, and any --depends-on/--parent edges — with an empty title for
// the human to supply), opens $EDITOR on it, and reads the result back. It enforces
// the one input create fundamentally needs — a non-empty title — on top of the
// store's usable-issue validation, and canonicalizes the filename once the title
// (and slug) are set. Any failure after the skeleton is written cleans it out of
// the issues directory, so an abandoned or invalid authoring never leaves a
// half-formed issue in the store: only a good result is returned, at its canonical
// path. The cleanup distinguishes an untouched skeleton (deleted — nothing was
// lost) from one the human typed into (stashed under .beaver/drafts — their words
// are never discarded, the same recovery contract git's COMMIT_EDITMSG offers).
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
	// Until a good result is imported at its canonical name, the skeleton is junk in
	// the issues directory: on any early (failure) return, delete it if the human
	// never changed it, or stash it as a draft if they did. The success path sets
	// committed first — by then the canonicalizing write below has already renamed
	// or replaced the skeleton, so it must be left alone.
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
	// The id is machine-owned frontmatter (ADR 0014): create minted it, and the
	// canonicalizing write below trusts it to name the file. An id rewritten in the
	// editor is refused — were it another issue's id, the write would land on that
	// issue's canonical name and silently replace it, the one data loss no Busy
	// Beaver write is ever allowed to commit. The authoring is stashed as a draft
	// like any other failed result.
	if edited.ID != seed.ID {
		errf(env, "create minted the id %s; it cannot be changed in the editor (the file now says %s)", seed.ID, edited.ID)
		return issue.Issue{}, "", exitError
	}
	if strings.TrimSpace(edited.Title) == "" {
		errf(env, "create needs a title; none was set in the editor")
		return issue.Issue{}, "", exitError
	}

	// The title — and so the slug — has almost certainly changed from the empty
	// skeleton, so rewrite at the canonical <id>-<slug> name and drop the skeleton
	// file, the same drift-fixing write the transitions use.
	path, err := st.Update(skeleton, edited)
	if err != nil {
		errf(env, "%v", err)
		return issue.Issue{}, "", exitError
	}
	committed = true
	return edited, path, exitOK
}

// abandonSkeleton cleans up after a failed interactive authoring. A skeleton the
// human never changed (they quit without writing, or the editor failed) is plain
// junk and is deleted. One they typed into holds their work: it is stashed under
// .beaver/drafts — out of the scanned issue set, so the store stays clean, but
// never deleted — and the draft's location is reported so the words are one
// copy-paste from a retry. If even the stash fails, the file is left where it is:
// a half-formed issue doctor will flag beats destroying what the human wrote.
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
