package core

// This file holds deletion: removing an issue's file outright.

import "github.com/builtbystef/beaver-backlog/internal/issue"

// Deleted is a deletion's result: the issue that was removed, the file it was
// removed from (which a caller may want to name), and the files the scan
// skipped.
type Deleted struct {
	Issue    issue.Issue
	Path     string
	Warnings []Warning
}

// Delete removes the issue ref names from the store outright: the hard delete
// for junk, as distinct from cancelling it, which keeps the file as an abandoned
// record. The store keeps no other copy, so the operator's version control is
// the only undo.
func (s *Service) Delete(ref string) (Deleted, error) {
	snap, warnings, err := s.scan()
	if err != nil {
		return Deleted{Warnings: warnings}, err
	}
	iss, path, err := resolve(snap, ref)
	if err != nil {
		return Deleted{Warnings: warnings}, err
	}
	if err := s.store.Delete(path); err != nil {
		return Deleted{Warnings: warnings}, err
	}
	return Deleted{Issue: iss, Path: path, Warnings: warnings}, nil
}
