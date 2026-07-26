package core

// This file holds the multi-field update: the change set an interface hands in,
// the rules for turning it into a new issue, and the net-change rule that keeps
// an update which effectively changes nothing from touching the file.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/store"
)

// Changes is the set of field edits an interface asks the core to apply to one
// issue. A nil pointer leaves its field alone, so a caller names only what it
// means to change; a pointer to the empty string clears Assignee or Parent.
//
// Labels and dependencies arrive as add and remove sets rather than as a
// replacement list, so a caller that knows about one entry never has to send —
// and so never silently drops — the ones it does not. An entry named in both
// sets is removed: a caller that says both has said the entry should be gone.
type Changes struct {
	Title *string
	// Body replaces the description only. An issue's notes are preserved
	// verbatim, whatever the new description says.
	Body *string
	// Assignee is set as given; the empty string returns the issue to unowned.
	// There is no ownership guard here — delegation is the point of assigning,
	// and Start keeps the only guard.
	Assignee *string
	// Priority must be a level or the empty value that clears it.
	Priority *issue.Priority

	AddLabels, RemoveLabels []string
	// The dependency sets hold references in any form Get accepts; they are
	// stored as canonical ids, so no edge rests on a slug a later retitling would
	// break.
	AddDependsOn, RemoveDependsOn []string
	// Parent is a reference resolved the same way; the empty string detaches the
	// issue from its parent.
	Parent *string
}

// CycleError reports a relationship change refused because it would close a
// cycle — an issue that would wait on itself, directly or through others, or a
// sub-issue that would be its own ancestor. It names the field the change was to
// and the issues caught in the loop.
type CycleError struct {
	Field string   // the frontmatter field at fault: depends_on or parent
	Cycle []string // the ids the cycle runs through, sorted
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("%s would create a cycle: %s", e.Field, strings.Join(e.Cycle, ", "))
}

// ParsePriority interprets a priority written as text: each of the four levels
// names itself, and "none" or the empty string mean unprioritized. Anything else
// is refused with an error naming the values that are accepted.
func ParsePriority(s string) (issue.Priority, error) {
	switch p := issue.Priority(strings.TrimSpace(s)); p {
	case issue.PriorityUrgent, issue.PriorityHigh, issue.PriorityMedium, issue.PriorityLow:
		return p, nil
	case "", "none":
		return "", nil
	default:
		return "", fmt.Errorf("invalid priority %q (want one of: urgent, high, medium, low, none)", s)
	}
}

// Update applies a change set to the issue ref names and writes the result. It
// is the one operation behind every field edit, so the rules — what an empty
// value clears, how a set is applied, when a write is skipped — are stated once
// however many fields a caller changes at a time.
//
// An update whose net effect is nothing writes nothing: the result reports
// Changed false and `updated` keeps the moment of the last real edit. A change
// that would close a dependency or parent cycle is refused with a *CycleError
// before anything is written, and an unresolvable reference refuses the whole
// update, so a typo never persists as a dangling edge.
func (s *Service) Update(ref string, c Changes) (Outcome, error) {
	if c.Title != nil && strings.TrimSpace(*c.Title) == "" {
		return Outcome{}, &ValidationError{Field: "title", Problem: "must not be empty"}
	}
	if c.Priority != nil && !c.Priority.Valid() {
		return Outcome{}, &ValidationError{
			Field:   "priority",
			Problem: fmt.Sprintf("%q is not a level (want one of: urgent, high, medium, low)", *c.Priority),
		}
	}

	snap, warnings, err := s.scan()
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	before, path, err := resolve(snap, ref)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}

	next, err := apply(snap, before, c)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	if !differs(before, next) {
		return Outcome{Issue: before, Previous: before, Warnings: warnings}, nil
	}
	if err := checkCycles(snap, before, next); err != nil {
		return Outcome{Warnings: warnings}, err
	}

	// The title travels with the file name, so a retitled issue is written to the
	// name its new slug implies — the store drops the old file, leaving the id
	// untouched and no second copy behind.
	written, err := s.write(path, next)
	if err != nil {
		return Outcome{Warnings: warnings}, err
	}
	return Outcome{Issue: written, Previous: before, Changed: true, Warnings: warnings}, nil
}

// apply builds the issue a change set describes from the issue as it stands,
// resolving every reference against the same scan the caller was resolved
// against. It writes nothing; a refusal here costs no file.
func apply(snap *store.Snapshot, before issue.Issue, c Changes) (issue.Issue, error) {
	next := before
	if c.Title != nil {
		next.Title = strings.TrimSpace(*c.Title)
	}
	if c.Body != nil {
		next.Body = issue.SetDescription(before.Body, *c.Body)
	}
	if c.Assignee != nil {
		next.Assignee = strings.TrimSpace(*c.Assignee)
	}
	if c.Priority != nil {
		next.Priority = *c.Priority
	}
	if len(c.AddLabels) > 0 || len(c.RemoveLabels) > 0 {
		next.Labels = applySet(before.Labels, dedupe(c.AddLabels), dedupe(c.RemoveLabels))
	}
	if len(c.AddDependsOn) > 0 || len(c.RemoveDependsOn) > 0 {
		adds, err := resolveAll(snap, c.AddDependsOn)
		if err != nil {
			return issue.Issue{}, err
		}
		next.DependsOn = applySet(before.DependsOn, adds, removalIDs(snap, c.RemoveDependsOn))
	}
	if c.Parent != nil {
		parent := ""
		if ref := strings.TrimSpace(*c.Parent); ref != "" {
			p, _, err := resolve(snap, ref)
			if err != nil {
				return issue.Issue{}, err
			}
			parent = p.ID
		}
		next.Parent = parent
	}
	return next, nil
}

// applySet returns current with removes deleted and adds appended, treating the
// values as an ordered set: removal wins over a simultaneous add, pre-existing
// duplicates collapse, and the result is nil when empty so the field marshals
// away.
func applySet(current, adds, removes []string) []string {
	drop := make(map[string]bool, len(removes))
	for _, r := range removes {
		drop[r] = true
	}
	seen := make(map[string]bool, len(current)+len(adds))
	var out []string
	keep := func(v string) {
		if drop[v] || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, v := range current {
		keep(v)
	}
	for _, v := range adds {
		keep(v)
	}
	return out
}

// removalIDs turns removal references into the ids to drop. Unlike an addition,
// a removal that resolves to nothing keeps the reference as written: a dangling
// edge names an issue no scan can find, and dropping it is exactly what a caller
// asking to remove it wants.
func removalIDs(snap *store.Snapshot, refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if iss, _, err := resolve(snap, ref); err == nil {
			out = append(out, iss.ID)
			continue
		}
		out = append(out, strings.TrimSpace(ref))
	}
	return out
}

// differs reports whether next says anything an update can change that before
// did not. It is the net-change rule: an update that adds and removes the same
// label lands here identical to what it started from, so nothing is written.
func differs(before, next issue.Issue) bool {
	return next.Title != before.Title ||
		next.Body != before.Body ||
		next.Assignee != before.Assignee ||
		next.Priority != before.Priority ||
		next.Parent != before.Parent ||
		!slices.Equal(next.Labels, before.Labels) ||
		!slices.Equal(next.DependsOn, before.DependsOn)
}

// checkCycles refuses a relationship change that would close a cycle. Only a
// cycle this change introduces refuses it: one that arrived some other way — a
// merge, a hand-edit — is doctor's to report (ADR 0005), and refusing every edit
// to an issue already caught in one would leave no way to edit it back out.
func checkCycles(snap *store.Snapshot, before, next issue.Issue) error {
	if slices.Equal(next.DependsOn, before.DependsOn) && next.Parent == before.Parent {
		return nil
	}
	was := issue.NewRelations(snap.Issues())
	now := issue.NewRelations(replacing(snap.Issues(), next))
	if !slices.Equal(next.DependsOn, before.DependsOn) {
		if cycle := introduced(was.Cycles(), now.Cycles()); cycle != nil {
			return &CycleError{Field: "depends_on", Cycle: cycle}
		}
	}
	if next.Parent != before.Parent {
		if cycle := introduced(was.ParentCycles(), now.ParentCycles()); cycle != nil {
			return &CycleError{Field: "parent", Cycle: cycle}
		}
	}
	return nil
}

// replacing returns issues with the one sharing next's id swapped for next, so
// the relationship graph can be derived over the store as the update would leave
// it.
func replacing(issues []issue.Issue, next issue.Issue) []issue.Issue {
	for i := range issues {
		if issues[i].ID == next.ID {
			issues[i] = next
			break
		}
	}
	return issues
}

// introduced returns the first cycle in after that was not already in before,
// or nil when the change added none. Cycles are sorted id lists, so identity is
// their joined form.
func introduced(before, after [][]string) []string {
	existing := make(map[string]bool, len(before))
	for _, cycle := range before {
		existing[strings.Join(cycle, ",")] = true
	}
	for _, cycle := range after {
		if !existing[strings.Join(cycle, ",")] {
			return cycle
		}
	}
	return nil
}
