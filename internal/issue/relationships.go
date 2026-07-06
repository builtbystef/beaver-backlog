package issue

import (
	"slices"
	"sort"
)

// Relations is a derived, read-only view over a set of issues that answers the
// relationship questions Busy Beaver never stores on disk: which of an issue's
// dependencies are still unmet, whether it is ready or blocked or stuck, and the
// inverse edges — what an issue blocks, and what its children are.
//
// Relationships are stored one-sided (ADR 0011): depends_on lives on the
// dependent, parent on the child, and neither inverse is written to a file. Every
// answer here is computed from the indexed set by scanning, so the stored forward
// edges are the single source of truth and no inverse can desync. A dependency is
// satisfied only when its target is done; a cancelled target never satisfies it,
// and a missing target (a dangling reference, doctor's concern) is likewise unmet
// — both leave the dependent not ready.
type Relations struct {
	byID map[string]Issue
}

// NewRelations indexes issues by their authoritative id for O(1) lookup. When two
// issues share an id — which validation forbids but a half-merged store can
// momentarily hold — the first in the given order wins, so a caller that passes
// issues in a stable order (the store's path order) gets a deterministic index.
func NewRelations(issues []Issue) *Relations {
	byID := make(map[string]Issue, len(issues))
	for _, iss := range issues {
		if _, seen := byID[iss.ID]; !seen {
			byID[iss.ID] = iss
		}
	}
	return &Relations{byID: byID}
}

// Blocker is one unmet dependency of an issue: the depends_on target, whether it
// is missing from the store, and its state when present. A dependency is met only
// when its target exists and is done, so every Blocker names a dependency that is
// not (yet, or ever) satisfied.
type Blocker struct {
	ID      string // the depends_on target id
	Missing bool   // the target is not in the store — a dangling reference (ADR 0005/0011)
	State   State  // the target's state; the zero State when Missing
}

// Cancelled reports whether this dependency can never be met on its own because
// its target was cancelled. Only done satisfies a dependency, so a cancelled
// target leaves the dependent stuck rather than merely waiting (ADR 0011).
func (b Blocker) Cancelled() bool { return !b.Missing && b.State == StateCancelled }

// BlockedOn returns iss's unmet dependencies, in its stored depends_on order:
// every target that is missing or not done. A target a hand-edit listed twice
// counts once — a duplicated edge is redundant, not a second blocker — so every
// consumer (show's waiting-on, start's warning, Stuck, doctor) sees each
// dependency exactly once. An empty result means every dependency is satisfied —
// or the issue has none. Only the issue's direct dependencies are considered: a
// dependency that is itself blocked is simply not done, so it keeps the dependent
// blocked with no transitive walk (and a dependency cycle just stays perpetually
// blocked, which doctor surfaces).
func (r *Relations) BlockedOn(iss Issue) []Blocker {
	var blockers []Blocker
	seen := make(map[string]bool, len(iss.DependsOn))
	for _, dep := range iss.DependsOn {
		if seen[dep] {
			continue
		}
		seen[dep] = true
		switch target, ok := r.byID[dep]; {
		case !ok:
			blockers = append(blockers, Blocker{ID: dep, Missing: true})
		case target.State != StateDone:
			blockers = append(blockers, Blocker{ID: dep, State: target.State})
		}
	}
	return blockers
}

// Ready reports whether iss can be started now: it is todo and every dependency is
// done. An issue with no dependencies is ready the moment it is todo. Ready is the
// selector behind `list --ready`.
func (r *Relations) Ready(iss Issue) bool {
	return iss.State == StateTodo && len(r.BlockedOn(iss)) == 0
}

// Blocked reports whether iss has any unmet dependency, independent of its own
// state — the glossary's "blocked", the condition show reports for any issue. The
// blocked *queue* (`list --blocked`) narrows this to todo work, but the derived
// condition itself is state-agnostic: an in-progress issue with an unmet dependency
// is still blocked.
func (r *Relations) Blocked(iss Issue) bool {
	return len(r.BlockedOn(iss)) > 0
}

// Stuck reports whether iss is blocked by a cancelled dependency — an edge only
// done could satisfy and cancellation never will, so the dependent cannot free
// itself by waiting. It is a proper subset of Blocked; doctor surfaces it for a
// human to re-scope or drop the edge (ADR 0011).
func (r *Relations) Stuck(iss Issue) bool {
	for _, b := range r.BlockedOn(iss) {
		if b.Cancelled() {
			return true
		}
	}
	return false
}

// Blocks returns the ids of issues that depend on iss — the derived inverse of
// depends_on, computed by scanning and never stored (ADR 0011), sorted for
// deterministic output.
func (r *Relations) Blocks(iss Issue) []string {
	var out []string
	for id, other := range r.byID {
		if slices.Contains(other.DependsOn, iss.ID) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// Children returns the ids of issues whose parent is iss — the derived inverse of
// parent (ADR 0011), sorted for deterministic output. An issue with children is
// informally an epic.
func (r *Relations) Children(iss Issue) []string {
	var out []string
	for id, other := range r.byID {
		if other.Parent == iss.ID {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// Relationship is the derived relationship summary for a single issue — the shape
// show renders: whether it is ready/blocked/stuck, exactly what it is waiting on,
// and the inverse edges it never stores. It is a snapshot computed from a Relations
// index (see For), not anything read from a file.
type Relationship struct {
	Ready     bool
	Blocked   bool
	Stuck     bool
	BlockedOn []Blocker
	Blocks    []string
	Children  []string
}

// For assembles the full derived Relationship for iss, computing the unmet
// dependencies once and deriving the readiness flags from them.
func (r *Relations) For(iss Issue) Relationship {
	blockedOn := r.BlockedOn(iss)
	stuck := false
	for _, b := range blockedOn {
		if b.Cancelled() {
			stuck = true
			break
		}
	}
	return Relationship{
		Ready:     iss.State == StateTodo && len(blockedOn) == 0,
		Blocked:   len(blockedOn) > 0,
		Stuck:     stuck,
		BlockedOn: blockedOn,
		Blocks:    r.Blocks(iss),
		Children:  r.Children(iss),
	}
}
