package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// This file holds the classification verbs — priority and label — that mutate
// the two triage fields after creation. Both change only their own field and
// skip the write entirely when nothing changes, so `updated` is churned only on
// a real edit.

// parsePriority interprets a priority argument from the command line. The four
// levels map to themselves; "none" and the empty string map to the empty
// Priority meaning unprioritized; anything else is an error naming the accepted
// values.
func parsePriority(s string) (issue.Priority, error) {
	switch p := issue.Priority(strings.TrimSpace(s)); p {
	case issue.PriorityUrgent, issue.PriorityHigh, issue.PriorityMedium, issue.PriorityLow:
		return p, nil
	case "", "none":
		return "", nil
	default:
		return "", fmt.Errorf("invalid priority %q (want one of: urgent, high, medium, low, none)", s)
	}
}

// cmdPriority sets or clears an issue's priority: `priority <ref> <level>` sets
// one of urgent|high|medium|low, `priority <ref> none` clears it. Setting the
// priority an issue already has is an idempotent no-op that does not rewrite
// the file.
func cmdPriority(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "priority")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 2 {
		errf(env, "priority requires an issue reference and a level: beaver priority <ref> <urgent|high|medium|low|none>")
		return exitUsage
	}
	level, err := parsePriority(pos[1])
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}
	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	if iss.Priority == level {
		return reportIssue(env, format, iss, fmt.Sprintf("%s is already %s", iss.ID, priorityWord(level)))
	}
	iss.Priority = level
	line := fmt.Sprintf("Set %s priority to %s", iss.ID, level)
	if level == "" {
		line = fmt.Sprintf("Cleared %s priority", iss.ID)
	}
	return writeAndReport(env, st, path, iss, format, line)
}

// priorityWord names a priority for a human line, rendering the empty value as
// "unprioritized".
func priorityWord(p issue.Priority) string {
	if p == "" {
		return "unprioritized"
	}
	return string(p)
}

// cmdLabel adds or removes an issue's labels. Positional arguments are labels
// to add; --remove names labels to drop, and a label named in both is removed.
// An invocation whose net effect changes nothing does not rewrite the file.
func cmdLabel(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "label")
	var remove csvList
	fs.Var(&remove, "remove", "label to remove (repeatable, comma-separated)")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) < 1 {
		errf(env, "label requires an issue reference: beaver label <ref> [<label>...] [--remove <label>...]")
		return exitUsage
	}
	// Positional adds are split on commas too, so `label ref a,b` matches
	// create's --label a,b.
	adds := dedupe(splitList(pos[1:]))
	removes := dedupe(remove.values)
	if len(adds) == 0 && len(removes) == 0 {
		errf(env, "label requires at least one label to add or remove")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}
	iss, path, code := resolveRef(env, st, pos[0])
	if code != exitOK {
		return code
	}

	next := applyLabels(iss.Labels, adds, removes)
	if slices.Equal(next, iss.Labels) {
		return reportIssue(env, format, iss, fmt.Sprintf("%s labels unchanged", iss.ID))
	}
	iss.Labels = next
	return writeAndReport(env, st, path, iss, format, fmt.Sprintf("Updated %s labels: %s", iss.ID, labelSummary(next)))
}

// splitList flattens raw arguments (each possibly a comma-separated group) into
// their trimmed, non-empty values.
func splitList(items []string) []string {
	var out []string
	for _, it := range items {
		out = append(out, splitCSV(it)...)
	}
	return out
}

// applyLabels returns current with removes deleted and adds appended, treating
// labels as an ordered set: removal wins over a simultaneous add, pre-existing
// duplicates collapse, and the result is nil when empty so the field marshals
// away.
func applyLabels(current, adds, removes []string) []string {
	drop := make(map[string]bool, len(removes))
	for _, r := range removes {
		drop[r] = true
	}
	seen := make(map[string]bool, len(current)+len(adds))
	var out []string
	keep := func(l string) {
		if drop[l] || seen[l] {
			return
		}
		seen[l] = true
		out = append(out, l)
	}
	for _, l := range current {
		keep(l)
	}
	for _, l := range adds {
		keep(l)
	}
	return out
}

// labelSummary renders a label set for a human confirmation line, showing
// "(none)" for an emptied list.
func labelSummary(labels []string) string {
	if len(labels) == 0 {
		return "(none)"
	}
	return strings.Join(labels, ", ")
}
