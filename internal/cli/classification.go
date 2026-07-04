package cli

import (
	"fmt"
	"slices"
	"strings"

	"beaver/internal/issue"
)

// This file holds the classification verbs — priority and label — that mutate the
// two triage fields after creation (create sets them inline via --priority and
// --label). There is no separate "type" concept: a type such as bug is just a
// label (CONTEXT.md), so tagging and typing are the one label command. Both verbs
// change only their own field, leave state and assignee untouched, and skip the
// write entirely when nothing changes, so `updated` is churned only on a real edit
// — the same idempotent-no-op stance the ownership verbs take (ADR 0009).

// parsePriority interprets a priority argument from the command line. The four
// levels map to themselves; "none" — and the empty string, i.e. an omitted create
// flag — map to the empty Priority that means unprioritized. Anything else is a
// usage error naming the accepted values. It is the one boundary that validates
// priority: a level reaches the store only through here (or a hand-edit doctor
// lints), so Validate stays narrow (ADR 0005).
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

// cmdPriority sets or clears an issue's priority — the single-valued triage
// ranking. `priority <ref> <level>` sets it to one of urgent|high|medium|low;
// `priority <ref> none` clears it back to unprioritized. State and assignee are
// untouched; setting the priority an issue already has is an idempotent no-op that
// does not rewrite the file.
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
// "unprioritized" rather than a blank so the no-op message reads naturally.
func priorityWord(p issue.Priority) string {
	if p == "" {
		return "unprioritized"
	}
	return string(p)
}

// cmdLabel adds or removes an issue's labels — the free-form, multi-valued tags.
// Positional arguments are labels to add; --remove names labels to drop. Labels
// are a set: adding one already present or removing one absent is a no-op on that
// label, and an invocation whose net effect changes nothing does not rewrite the
// file. Adds and removes may be combined in one call; a label named in both is
// removed (the removal wins), so the outcome does not depend on ordering.
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
	// Positional adds are split on commas too, so `label ref a,b` and `label ref a b`
	// mean the same thing and match create's --label a,b.
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

// splitList applies splitCSV to each item, so a slice of raw arguments (each
// possibly a `a,b` group) flattens to its trimmed, non-empty labels.
func splitList(items []string) []string {
	var out []string
	for _, it := range items {
		out = append(out, splitCSV(it)...)
	}
	return out
}

// applyLabels returns current with removes deleted and adds appended, treating
// labels as an ordered set: existing order is preserved, genuinely new labels
// append in first-seen order, and a label present in removes is dropped even if it
// is also in adds (removal wins). Any pre-existing duplicate in current collapses
// to one, so the write also tidies a hand-edited list. The result is nil when
// empty, so the field marshals away entirely (omitempty).
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

// labelSummary renders a label set for a human confirmation line, showing "(none)"
// for an emptied list so the message still says something concrete.
func labelSummary(labels []string) string {
	if len(labels) == 0 {
		return "(none)"
	}
	return strings.Join(labels, ", ")
}
