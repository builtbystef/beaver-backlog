package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
)

// This file holds the two direct-manipulation verbs — edit and delete — and the
// editor plumbing interactive create shares with edit. Both editing paths hand a
// raw issue file to the user's $EDITOR and then re-validate what came back (ADR
// 0005): the file is the issue, so freeform hand-editing is a first-class way to
// change one, guarded only by "the result must still be a usable issue."

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

// cmdEdit opens an issue's raw file in $EDITOR for freeform hand-editing, then
// re-validates the result. It edits the file in place — the file is the single
// source of truth (ADR 0001), so what the human saves is what the issue becomes —
// and deliberately does not re-serialize it: a hand-edit is honored verbatim, and
// the untidiness it may leave (a filename that no longer matches a changed title)
// is lint for doctor, not a failure (ADR 0005). A non-interactive invocation
// refuses instead of hanging; an edit that leaves the file unparseable or invalid
// is reported, and the file is left as saved for the human to fix or re-edit.
func cmdEdit(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "edit")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "edit requires an issue reference: beaver edit <ref>")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	// Refuse before touching the editor when the session cannot host one, so a
	// piped or agent run gets a clear error instead of a blocked process.
	if err := editorGate(env); err != nil {
		errf(env, "cannot edit %s: %v", iss.ID, err)
		return exitError
	}
	if err := env.Edit(path); err != nil {
		errf(env, "editor failed: %v", err)
		return exitError
	}

	// Re-read the file the human just saved through the store's usable-issue
	// contract. On a valid result, report the re-read issue (which reflects any
	// edited fields); on an invalid one, say what is wrong and leave the file alone.
	edited, err := st.Read(path)
	if err != nil {
		errf(env, "%s is no longer a valid issue after editing: %v", iss.ID, err)
		return exitError
	}
	return reportIssue(env, format, edited, fmt.Sprintf("Edited %s", edited.ID))
}

// cmdDelete removes an issue's file outright — the hard delete of junk that should
// never have existed (a typo, an accidental duplicate), as distinct from cancel,
// which keeps the file as a deliberately-abandoned record (ADR 0004). It does not
// prompt: resolution already demands an exact reference, so a delete names one
// specific issue, and a VCS (when present) retains the history as the undo. After
// it runs the issue is gone from every read path.
func cmdDelete(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "delete")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "delete requires an issue reference: beaver delete <ref>")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	if err := st.Delete(path); err != nil {
		errf(env, "%v", err)
		return exitError
	}

	if format == output.JSON {
		if err := output.WriteJSON(env.Stdout, map[string]any{"id": iss.ID, "title": iss.Title, "deleted": true}); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintf(env.Stdout, "Deleted %s  %s\n  removed %s\n", iss.ID, output.OneLine(iss.Title), relPath(env.WorkDir, path))
	return exitOK
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
