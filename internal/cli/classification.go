package cli

import (
	"fmt"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// This file holds the classification verbs — priority and label — that mutate
// the two triage fields after creation. Each is one core update of its own
// field, so the rules that decide the result (the level a word names, which
// labels survive an add and a remove, and whether anything changed at all) are
// the core's; what stays here is the wording.

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
	level, err := core.ParsePriority(pos[1])
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	svc, err := open(env)
	if err != nil {
		return coreError(env, err)
	}
	out, err := svc.Update(pos[0], core.Changes{Priority: &level})
	warnSkipped(env, out.Warnings)
	if err != nil {
		return coreError(env, err)
	}
	if !out.Changed {
		return reportIssue(env, format, out.Issue, fmt.Sprintf("%s is already %s", out.Issue.ID, priorityWord(level)))
	}
	line := fmt.Sprintf("Set %s priority to %s", out.Issue.ID, level)
	if level == "" {
		line = fmt.Sprintf("Cleared %s priority", out.Issue.ID)
	}
	return reportIssue(env, format, out.Issue, line)
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
	adds := splitList(pos[1:])
	removes := remove.values
	if len(adds) == 0 && len(removes) == 0 {
		errf(env, "label requires at least one label to add or remove")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	svc, err := open(env)
	if err != nil {
		return coreError(env, err)
	}
	out, err := svc.Update(pos[0], core.Changes{AddLabels: adds, RemoveLabels: removes})
	warnSkipped(env, out.Warnings)
	if err != nil {
		return coreError(env, err)
	}
	if !out.Changed {
		return reportIssue(env, format, out.Issue, fmt.Sprintf("%s labels unchanged", out.Issue.ID))
	}
	return reportIssue(env, format, out.Issue,
		fmt.Sprintf("Updated %s labels: %s", out.Issue.ID, labelSummary(out.Issue.Labels)))
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

// labelSummary renders a label set for a human confirmation line, showing
// "(none)" for an emptied list.
func labelSummary(labels []string) string {
	if len(labels) == 0 {
		return "(none)"
	}
	return strings.Join(labels, ", ")
}
