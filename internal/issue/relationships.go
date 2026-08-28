package issue

import (
	"slices"
	"sort"
)

// Relations is a derived, read-only view over a set of issues that answers the
// relationship questions never stored on disk: which dependencies are unmet,
// whether an issue is ready, blocked, or stuck, and the inverse edges: what an
// issue blocks, and what its children are.
//
// Relationships are stored one-sided, with depends_on on the dependent and
// parent on the child. Every answer is computed by scanning, so the stored
// forward edges are the single source of truth and no inverse can desync. A dependency
// is satisfied only when its target is done; a cancelled or missing target is
// unmet and leaves the dependent not ready.
type Relations struct {
	byID map[string]Issue
}

// NewRelations indexes issues by their authoritative id. When two issues share
// an id, the first in the given order wins, so a stable input order gives a
// deterministic index.
func NewRelations(issues []Issue) *Relations {
	byID := make(map[string]Issue, len(issues))
	for _, iss := range issues {
		if _, seen := byID[iss.ID]; !seen {
			byID[iss.ID] = iss
		}
	}
	return &Relations{byID: byID}
}

// Blocker is one unmet dependency of an issue: the depends_on target, whether
// it is missing from the store, and its state when present.
type Blocker struct {
	ID      string // the depends_on target id
	Missing bool   // the target is not in the store: a dangling reference
	State   State  // the target's state; the zero State when Missing
}

// Cancelled reports whether this dependency can never be met on its own because
// its target was cancelled. Only done satisfies a dependency, so a cancelled
// target leaves the dependent stuck rather than merely waiting.
func (b Blocker) Cancelled() bool { return !b.Missing && b.State == StateCancelled }

// BlockedOn returns iss's unmet dependencies, every target that is missing or
// not done, in stored depends_on order, with a duplicated edge counted once.
// Only direct dependencies are considered: a dependency that is itself blocked
// is simply not done, so there is no transitive walk.
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

// Ready reports whether iss can be started now: it is todo and every dependency
// is done.
func (r *Relations) Ready(iss Issue) bool {
	return iss.State == StateTodo && len(r.BlockedOn(iss)) == 0
}

// Blocked reports whether iss has any unmet dependency, independent of its own
// state: an in-progress issue with an unmet dependency is still blocked.
func (r *Relations) Blocked(iss Issue) bool {
	return len(r.BlockedOn(iss)) > 0
}

// Stuck reports whether iss is blocked by a cancelled dependency, which waiting
// can never clear. It is a proper subset of Blocked.
func (r *Relations) Stuck(iss Issue) bool {
	for _, b := range r.BlockedOn(iss) {
		if b.Cancelled() {
			return true
		}
	}
	return false
}

// Blocks returns the ids of issues that depend on iss, the derived inverse of
// depends_on, sorted for deterministic output.
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

// Children returns the ids of issues whose parent is iss, the derived inverse
// of parent, sorted for deterministic output.
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

// Relationship is the derived relationship summary for a single issue: whether
// it is ready, blocked, or stuck, what it is waiting on, and the inverse edges.
// It is a snapshot computed from a Relations index (see For), not read from a
// file.
type Relationship struct {
	Ready     bool
	Blocked   bool
	Stuck     bool
	BlockedOn []Blocker
	Blocks    []string
	Children  []string
}

// For assembles the full derived Relationship for iss, computing the unmet
// dependencies once.
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
