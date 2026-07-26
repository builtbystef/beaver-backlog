package core

// This file holds issue creation: the draft an interface hands in, the rules a
// draft must satisfy, and collision-safe id minting.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/store"
)

// Draft is an issue an interface asks the core to create. DependsOn and Parent
// hold references in any form Get accepts; they are stored as canonical ids, so
// no edge rests on a slug a later retitling would break.
type Draft struct {
	Title     string
	Body      string
	Labels    []string
	Priority  issue.Priority
	DependsOn []string // references to the issues this one waits on
	Parent    string   // reference to the issue this one is a sub-issue of
}

// Created is a creation's result: the issue as written, the file it landed in —
// which a caller may want to name — and the files the scan skipped.
type Created struct {
	Issue    issue.Issue
	Path     string
	Warnings []Warning
}

// ValidationError reports input the core refused because it does not describe
// an issue that can exist. It names the field and what is wrong with it, so a
// caller can phrase the refusal in its own words.
type ValidationError struct {
	Field   string // the field at fault
	Problem string // what is wrong with it
}

func (e *ValidationError) Error() string { return e.Field + " " + e.Problem }

// Create mints the issue a draft describes: it validates the draft, resolves
// the relationship references to ids, draws a collision-safe id, and writes the
// file with created and updated stamped from the same instant. A reference that
// names no issue — or several — refuses the whole creation before anything is
// written, so a typo never persists as a dangling edge.
func (s *Service) Create(d Draft) (Created, error) {
	if strings.TrimSpace(d.Title) == "" {
		return Created{}, &ValidationError{Field: "title", Problem: "must not be empty"}
	}
	if !d.Priority.Valid() {
		return Created{}, &ValidationError{
			Field:   "priority",
			Problem: fmt.Sprintf("%q is not a level (want one of: urgent, high, medium, low)", d.Priority),
		}
	}
	snap, warnings, err := s.scan()
	if err != nil {
		return Created{Warnings: warnings}, err
	}

	// Resolve the edges before minting anything, so an unresolvable reference
	// fails with no file written and no id spent.
	deps, err := resolveAll(snap, d.DependsOn)
	if err != nil {
		return Created{Warnings: warnings}, err
	}
	var parent string
	if ref := strings.TrimSpace(d.Parent); ref != "" {
		p, _, err := resolve(snap, ref)
		if err != nil {
			return Created{Warnings: warnings}, err
		}
		parent = p.ID
	}

	id, err := s.mintID(snap)
	if err != nil {
		return Created{Warnings: warnings}, err
	}
	now := s.now()
	iss := issue.Issue{
		ID:        id,
		Title:     strings.TrimSpace(d.Title),
		State:     issue.StateTodo,
		Priority:  d.Priority,
		Labels:    dedupe(d.Labels), // nil when none, so the field marshals away
		DependsOn: deps,
		Parent:    parent,
		Created:   now,
		Updated:   now,
		Body:      d.Body,
	}
	path, err := s.store.Write(iss)
	if err != nil {
		return Created{Warnings: warnings}, err
	}
	return Created{Issue: iss, Path: path, Warnings: warnings}, nil
}

// mintID draws a fresh id from the service's source, retrying on the rare
// collision with an issue the store already holds. The bound guards against a
// pathological generator, not a full store.
func (s *Service) mintID(snap *store.Snapshot) (string, error) {
	for range 100 {
		id := s.newID()
		if !snap.IDTaken(id) {
			return id, nil
		}
	}
	return "", errors.New("could not generate a unique issue ID")
}

// resolveAll turns relationship references into canonical ids, deduping by
// resolved id while preserving first-seen order, so several references to one
// issue collapse into a single edge.
func resolveAll(snap *store.Snapshot, refs []string) ([]string, error) {
	seen := make(map[string]bool, len(refs))
	var ids []string
	for _, ref := range refs {
		iss, _, err := resolve(snap, ref)
		if err != nil {
			return nil, err
		}
		if !seen[iss.ID] {
			seen[iss.ID] = true
			ids = append(ids, iss.ID)
		}
	}
	return ids, nil
}

// dedupe returns in with later duplicates dropped, preserving first-seen order,
// and nil when nothing is left — the shape an absent frontmatter list takes.
func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
