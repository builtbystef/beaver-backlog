// Package issue defines Busy Beaver's core domain type — the Issue — together with
// its on-disk Markdown representation (YAML frontmatter + body), its identity
// (a short random ID), and the slug derived from its title.
//
// An Issue file is the single source of truth (ADR 0001); the ID stored in the
// frontmatter is authoritative and the filename only mirrors it (ADR 0002).
package issue

import "time"

// State is an issue's position in its lifecycle. The set is fixed at four values
// (ADR 0004); "open" and "closed" are derived query views, never stored.
type State string

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

const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// Issue is the unit of work Busy Beaver tracks. The fields mirror the frontmatter
// schema; Body is the Markdown content that follows the frontmatter and holds
// the description (and, in later slices, the notes log).
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

	// Custom holds frontmatter keys Busy Beaver does not define — user-added fields
	// (e.g. sprint:, estimate:) that the schema knows nothing about. They are
	// carried through a read-modify-write untouched rather than silently
	// dropped, so a hand-added field survives commands like done or claim
	// (ADR 0014). Values are whatever YAML the user wrote (scalars, sequences,
	// maps); Busy Beaver preserves them but never interprets them. Nil when the file
	// has no custom keys.
	Custom map[string]any
}
