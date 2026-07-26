package core

// This file holds the hand-editing seam: handing an issue's raw file to an
// interface that lets a human edit it directly, and reading back what they
// saved. It exists only for as long as an interface offers that — the editor
// machinery goes away with the command surface it belongs to — so nothing else
// in the core hands out a path to write through.

import "github.com/builtbystef/beaver-backlog/internal/issue"

// Editable is an issue together with the file it lives in, for an interface that
// hands that file to a human. The path is where the edit lands, so what comes
// back is read from the same file — see Reread.
type Editable struct {
	Issue    issue.Issue
	Path     string
	Warnings []Warning
}

// Editable resolves ref to the issue and the file it was read from. It is the
// one read that hands out a path, because a hand-edit happens in the file itself
// rather than through an operation the core could apply.
func (s *Service) Editable(ref string) (Editable, error) {
	snap, warnings, err := s.scan()
	if err != nil {
		return Editable{Warnings: warnings}, err
	}
	iss, path, err := resolve(snap, ref)
	if err != nil {
		return Editable{Warnings: warnings}, err
	}
	return Editable{Issue: iss, Path: path, Warnings: warnings}, nil
}

// Reread reads back the issue at path after a hand-edit, applying the same
// usable-issue contract every scan applies and returning why it is not one when
// it is not. The file is deliberately not rewritten: a hand-edit is honored
// verbatim, and any filename drift it leaves is doctor's lint rather than a
// failure.
func (s *Service) Reread(path string) (issue.Issue, error) {
	return s.store.Read(path)
}
