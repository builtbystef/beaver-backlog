package core

// This file holds the coordination log: appending an attributed, timestamped
// note to an issue.

import (
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// Note appends an entry to the issue ref names, attributed to actor and stamped
// with the moment it is recorded, the same instant the write records as
// `updated`, so the log and the file agree on when it happened. Notes are
// allowed on an issue in any state, closed ones included: a for-the-record note
// on finished work is legitimate, and the log never gates on lifecycle.
//
// A note is never a no-op. Every call writes, because an entry is a record of a
// moment rather than a state to converge on: two identical notes are two
// notes.
func (s *Service) Note(ref, actor, text string) (Outcome, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Outcome{}, &ValidationError{Field: "note text", Problem: "must not be empty"}
	}
	if strings.TrimSpace(actor) == "" {
		return Outcome{}, &ValidationError{Field: "note author", Problem: "must not be empty"}
	}
	snap, warnings, err := s.scan()
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	iss, path, err := resolve(snap, ref)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}

	now := s.now()
	previous := iss
	iss.Body = issue.AppendNote(iss.Body, issue.Note{Author: actor, Time: now, Text: text})
	iss, err = s.writeAt(path, iss, now)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	return Outcome{Issue: iss, Previous: previous, Changed: true, Warnings: warnings}, nil
}
