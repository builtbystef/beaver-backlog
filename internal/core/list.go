package core

// This file holds the query engine: which issues a Query selects, and the order
// a listing comes back in.

import (
	"slices"
	"sort"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// Query selects a subset of the store. Every field is a refinement that stacks
// with the others: an unset field constrains nothing, and an issue must satisfy
// every active one. Ready and Blocked are the two halves of the unstarted work
// over the dependency graph, so setting both selects nothing.
type Query struct {
	States     []issue.State    // match any of these states; empty matches every state
	Ready      bool             // only ready issues: todo with every dependency done
	Blocked    bool             // only blocked issues: todo with an unmet dependency
	Labels     []string         // only issues carrying every one of these labels
	Priorities []issue.Priority // match any of these; the empty Priority matches the unprioritized, an empty slice matches every priority
	Assignee   *string          // only issues assigned to this actor; the empty string matches the unassigned, nil matches every issue
}

// Listing is a query's result: the matching issues in display order, and the
// files the scan skipped.
type Listing struct {
	Issues   []issue.Issue
	Warnings []Warning
}

// List returns the issues q selects, ordered by priority (urgent first,
// unprioritized last), then oldest first, with the ID as a tiebreak. The slice
// is never nil, so an empty result stays distinguishable from a failure.
func (s *Service) List(q Query) (Listing, error) {
	snap, warnings, err := s.scan()
	if err != nil {
		return Listing{Warnings: warnings}, err
	}
	all := snap.Issues()
	rel := issue.NewRelations(all)
	issues := make([]issue.Issue, 0, len(all))
	for _, iss := range all {
		if q.matches(iss, rel) {
			issues = append(issues, iss)
		}
	}
	sortIssues(issues)
	return Listing{Issues: issues, Warnings: warnings}, nil
}

// matches reports whether iss satisfies every active dimension of q, with rel
// answering the derived questions the issue file does not store.
func (q Query) matches(iss issue.Issue, rel *issue.Relations) bool {
	if len(q.States) > 0 && !slices.Contains(q.States, iss.State) {
		return false
	}
	if q.Ready && !rel.Ready(iss) {
		return false
	}
	// Blocked is scoped to todo because it answers "what unstarted work is
	// waiting"; Get still reports a blocked in-progress issue as blocked.
	if q.Blocked && (iss.State != issue.StateTodo || !rel.Blocked(iss)) {
		return false
	}
	if len(q.Priorities) > 0 && !slices.Contains(q.Priorities, iss.Priority) {
		return false
	}
	if q.Assignee != nil && iss.Assignee != *q.Assignee {
		return false
	}
	return hasAllLabels(iss.Labels, q.Labels)
}

// hasAllLabels reports whether have carries every label in want (AND semantics).
// An empty want matches everything.
func hasAllLabels(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := make(map[string]bool, len(have))
	for _, l := range have {
		set[l] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

// sortIssues orders issues by priority (urgent first, unprioritized last), then
// oldest first, with the ID as a tiebreak so issues minted at the same instant
// still sort reproducibly.
func sortIssues(issues []issue.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if ra, rb := a.Priority.Rank(), b.Priority.Rank(); ra != rb {
			return ra < rb
		}
		if !a.Created.Equal(b.Created) {
			return a.Created.Before(b.Created)
		}
		return a.ID < b.ID
	})
}
