package cli

import (
	"errors"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// This file holds the editor plumbing: driving $EDITOR over a file and deciding
// what to do with what comes back. Interactive create's authoring lives here —
// the orchestration is the interface's, while composing, filing, and abandoning
// the authoring are the core's.

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

// authorInEditor drives interactive create: the core composes the draft into a
// skeleton file, $EDITOR opens on it, and the core files what comes back. Any
// failure once the skeleton exists abandons it — an untouched one deleted, one
// typed into stashed as a draft — so only a finished authoring becomes an issue.
func authorInEditor(env Env, svc *core.Service, d core.Draft) (core.Created, int) {
	composed, err := svc.Compose(d)
	warnSkipped(env, composed.Warnings)
	if err != nil {
		return core.Created{}, coreError(env, err)
	}

	// The success path sets filed first: by then the canonicalizing write has
	// already renamed or replaced the skeleton, so it must be left alone.
	filed := false
	defer func() {
		if !filed {
			abandonAuthoring(env, svc, composed)
		}
	}()

	if err := env.Edit(composed.Path); err != nil {
		errf(env, "editor failed: %v", err)
		return core.Created{}, exitError
	}
	created, err := svc.Finish(composed.Path, composed.Issue)
	if err != nil {
		return core.Created{}, authoringError(env, err)
	}
	filed = true
	return created, exitOK
}

// authoringError maps the core's refusals of an authoring onto create's
// diagnostics: a result that is no longer an issue, an id the human rewrote,
// and the title they never supplied. All three leave the authoring unfiled, so
// the caller's cleanup still runs.
func authoringError(env Env, err error) int {
	var unusable *core.UnusableAuthoringError
	var reassigned *core.ReassignedIDError
	var invalid *core.ValidationError
	switch {
	case errors.As(err, &unusable):
		errf(env, "issue is not valid after editing: %v", unusable.Err)
	case errors.As(err, &reassigned):
		errf(env, "create minted the id %s; it cannot be changed in the editor (the file now says %s)",
			reassigned.Minted, reassigned.Found)
	case errors.As(err, &invalid):
		errf(env, "create needs a title; none was set in the editor")
	default:
		errf(env, "%v", err)
	}
	return exitError
}

// abandonAuthoring cleans up after an authoring that never became an issue and
// says where the human's words went. Their draft is never silently discarded:
// if even the stash fails, the file is left in place and named.
func abandonAuthoring(env Env, svc *core.Service, composed core.Created) {
	stashed, err := svc.Abandon(composed.Path, composed.Issue)
	switch {
	case err != nil:
		errf(env, "could not stash your draft (%v); it remains at %s", err, relPath(env.WorkDir, composed.Path))
	case stashed != "":
		errf(env, "your draft is saved at %s", relPath(env.WorkDir, stashed))
	}
}
