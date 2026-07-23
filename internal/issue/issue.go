// Package issue defines Beaver Backlog's core domain type — the Issue — together with
// its on-disk Markdown representation (YAML frontmatter + body), its identity
// (a short random ID), and the slug derived from its title.
//
// An Issue file is the single source of truth; the ID stored in the frontmatter
// is authoritative and the filename only mirrors it.
package issue

import (
	"sort"
	"time"
)

// State is an issue's position in its lifecycle. "Open" and "closed" are
// derived query views, never stored.
type State string

// The four legal issue states.
const (
	StateTodo       State = "todo"
	StateInProgress State = "in-progress"
	StateDone       State = "done"
	StateCancelled  State = "cancelled"
)

// Valid reports whether s is one of the four legal states.
func (s State) Valid() bool {
	switch s {
	case StateTodo, StateInProgress, StateDone, StateCancelled:
		return true
	default:
		return false
	}
}

// Priority is an optional ordinal ranking of urgency. Absent means "no priority".
type Priority string

// The four priority levels, most urgent first.
const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// Valid reports whether p is one of the four levels or empty (unprioritized).
// An invalid priority is not a load failure; only doctor surfaces it.
func (p Priority) Valid() bool {
	switch p {
	case PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, "":
		return true
	default:
		return false
	}
}

// Rank returns p's sort order, most urgent first. Empty and unrecognized
// priorities rank last so they sort to the bottom rather than the top.
func (p Priority) Rank() int {
	switch p {
	case PriorityUrgent:
		return 0
	case PriorityHigh:
		return 1
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 3
	default:
		return 4
	}
}

// Issue is the unit of work Beaver Backlog tracks. The fields mirror the frontmatter
// schema; Body is the Markdown content that follows the frontmatter.
type Issue struct {
	ID        string
	Title     string
	State     State
	Assignee  string   // optional, single actor
	Priority  Priority // optional
	Labels    []string // optional, free-form, multi
	DependsOn []string // optional, issue IDs
	Parent    string   // optional, issue ID
	Created   time.Time
	Updated   time.Time
	Body      string

	// Custom holds frontmatter keys Beaver Backlog does not define. User-added
	// fields are carried through a read-modify-write untouched, preserved but
	// never interpreted. Nil when the file has no custom keys.
	Custom map[string]any
}

// CustomKeys returns m's keys in sorted order, matching the order the YAML
// encoder writes them in.
func CustomKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
