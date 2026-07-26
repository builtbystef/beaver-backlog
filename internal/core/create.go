package core

// This file holds issue creation: the draft an interface hands in, the rules a
// draft must satisfy, collision-safe id minting, and the three moves an
// interactive authoring is made of — compose a skeleton, finish it, abandon it.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
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
	return s.create(d)
}

// Compose opens an interactive authoring: it does everything Create does except
// require a title, and what it writes is a skeleton for a human to finish in an
// editor. The id is minted here rather than on the way back, so the id the human
// is shown is the id the issue keeps — see Finish, which holds them to it.
func (s *Service) Compose(d Draft) (Created, error) {
	return s.create(d)
}

// create is Create and Compose's shared body: every creation rule except the
// title, which only a finished issue must satisfy.
func (s *Service) create(d Draft) (Created, error) {
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
		Title:     strings.TrimSpace(d.Title), // empty for a composed skeleton; the human supplies it
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

// UnusableAuthoringError reports that what came back from an authoring is not a
// usable issue at all — unreadable, malformed, or failing validation — so there
// is nothing to file. It carries the underlying reason.
type UnusableAuthoringError struct {
	Path string // the file the authoring was in
	Err  error  // why it is not a usable issue
}

func (e *UnusableAuthoringError) Error() string {
	return fmt.Sprintf("%s is not a usable issue: %v", e.Path, e.Err)
}

func (e *UnusableAuthoringError) Unwrap() error { return e.Err }

// ReassignedIDError reports an authoring that came back under a different id
// than the one composed for it. The id is the machine's: were the new one
// another issue's, filing the authoring at its canonical name would land on that
// issue's file and replace it.
type ReassignedIDError struct {
	Minted string // the id Compose minted
	Found  string // the id the authoring now carries
}

func (e *ReassignedIDError) Error() string {
	return fmt.Sprintf("the composed id %s came back as %s", e.Minted, e.Found)
}

// Finish files the issue a human authored over a composed skeleton: the
// authoring must still be a usable issue, must carry the composed id back
// unchanged, and must have gained a title. It is written at the canonical
// <id>-<slug> name the title now implies, which drops the skeleton's file.
//
// The timestamps are left as the authoring carries them: the issue is the one
// Compose stamped, so finishing it is not a modification of an existing issue.
func (s *Service) Finish(path string, seed issue.Issue) (Created, error) {
	authored, err := s.store.Read(path)
	if err != nil {
		return Created{}, &UnusableAuthoringError{Path: path, Err: err}
	}
	if authored.ID != seed.ID {
		return Created{}, &ReassignedIDError{Minted: seed.ID, Found: authored.ID}
	}
	if strings.TrimSpace(authored.Title) == "" {
		return Created{}, &ValidationError{Field: "title", Problem: "must not be empty"}
	}
	final, err := s.store.Update(path, authored)
	if err != nil {
		return Created{}, err
	}
	return Created{Issue: authored, Path: final}, nil
}

// Abandon disposes of an authoring that never became an issue, so a composed
// skeleton is never left in the issue set. A skeleton the human said nothing new
// in is deleted outright; one they typed into is stashed as a draft and its
// destination returned ("" when it was deleted), because their words are not
// ours to discard.
func (s *Service) Abandon(path string, seed issue.Issue) (stashed string, err error) {
	if s.untouched(path, seed) {
		// Best effort: a skeleton that cannot be removed is junk in the store, not
		// lost work, and doctor reports it.
		s.store.Delete(path)
		return "", nil
	}
	stashed, err = s.store.StashDraft(path)
	if errors.Is(err, os.ErrNotExist) {
		// The authoring is already gone — the editor removed it, or another hand
		// did. There is nothing left to keep and nothing to report.
		return "", nil
	}
	return stashed, err
}

// untouched reports whether the authoring at path still says exactly what the
// skeleton said. It compares the issue the file now holds against the seed
// re-serialized, so an authoring saved unchanged — or changed only in
// formatting — reads as untouched, while one that no longer parses certainly
// does not.
func (s *Service) untouched(path string, seed issue.Issue) bool {
	authored, err := s.store.Read(path)
	if err != nil {
		return false
	}
	current, err := issue.Marshal(authored)
	if err != nil {
		return false
	}
	seeded, err := issue.Marshal(seed)
	if err != nil {
		return false
	}
	return bytes.Equal(current, seeded)
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
