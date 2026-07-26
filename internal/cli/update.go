package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// This file holds update, the one command behind every mutation that is not a
// state change: title, description, assignee, priority, labels, dependencies,
// and parent, any number of them in a single invocation. What a change set means
// — what an empty value clears, how a set is applied, whether anything changed
// at all — is the core's; what stays here is turning flags into a core.Changes
// and wording what came back.
//
// A flag's presence, not its value, marks a field for change, so an update that
// empties a field is told apart from one that leaves it alone. The clearing
// flags say it in words rather than by an empty value — --unassign, --no-parent,
// and `--priority none` — so an empty --assignee is a mistyped invocation, not a
// silent release.

// mutationFlags are update's field flags. At least one must be named: an update
// that asks for no change is a caller mistake, distinct from one whose changes
// net out to nothing.
var mutationFlags = []string{
	"title", "body", "body-file", "assignee", "unassign",
	"priority", "label", "depends-on", "parent", "no-parent",
}

// exclusivePairs are the flag pairs that contradict each other: two sources for
// the description, and a set paired with its own clear.
var exclusivePairs = [][2]string{
	{"body", "body-file"},
	{"assignee", "unassign"},
	{"parent", "no-parent"},
}

// cmdUpdate applies field changes to one issue. It takes a single reference and
// carries no --state or --claim: lifecycle verbs are the only path to a state,
// and an agent claims work with start.
func cmdUpdate(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "update")
	titleFlag := fs.String("title", "", "set the title (renames the file to the new slug)")
	bodyFlag := fs.String("body", "", "replace the description, preserving the notes section")
	bodyFileFlag := fs.String("body-file", "", "read the replacement description from a file, or - for stdin")
	assigneeFlag := fs.String("assignee", "", "assign the issue to this actor")
	unassignFlag := fs.Bool("unassign", false, "clear the assignee")
	priorityFlag := fs.String("priority", "", "set priority: urgent|high|medium|low|none")
	var labels csvList
	fs.Var(&labels, "label", "label to add, or -<label> to remove (repeatable, comma-separated)")
	var dependsOn csvList
	fs.Var(&dependsOn, "depends-on", "issue to depend on, or -<ref> to stop depending on (repeatable, comma-separated)")
	parentFlag := fs.String("parent", "", "parent issue this is a sub-issue of (a ref)")
	noParentFlag := fs.Bool("no-parent", false, "detach the issue from its parent")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "update requires a single issue reference: beaver update <ref> <flags>")
		return exitUsage
	}

	named := namedFlags(fs)
	if !changesRequested(named) {
		errf(env, "update requires at least one field to change: beaver update <ref> --title|--body|--assignee|--priority|--label|--depends-on|--parent")
		return exitUsage
	}
	for _, pair := range exclusivePairs {
		if named[pair[0]] && named[pair[1]] {
			errf(env, "--%s and --%s are mutually exclusive", pair[0], pair[1])
			return exitUsage
		}
	}

	// Everything the invocation alone can be wrong about is settled before the
	// store is touched, so a typo costs no read and no write.
	priority, err := core.ParsePriority(*priorityFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	labelChange, err := parseSetChange("label", labels.values)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	depChange, err := parseSetChange("depends-on", dependsOn.values)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	changes := core.Changes{
		AddLabels:       labelChange.add,
		RemoveLabels:    labelChange.remove,
		AddDependsOn:    depChange.add,
		RemoveDependsOn: depChange.remove,
	}
	if named["title"] {
		changes.Title = titleFlag
	}
	if named["priority"] {
		changes.Priority = &priority
	}
	unowned, detached := "", ""
	switch {
	case named["assignee"]:
		if strings.TrimSpace(*assigneeFlag) == "" {
			errf(env, "--assignee requires an actor; use --unassign to clear it")
			return exitUsage
		}
		changes.Assignee = assigneeFlag
	case *unassignFlag:
		changes.Assignee = &unowned
	}
	switch {
	case named["parent"]:
		if strings.TrimSpace(*parentFlag) == "" {
			errf(env, "--parent requires an issue reference; use --no-parent to clear it")
			return exitUsage
		}
		changes.Parent = parentFlag
	case *noParentFlag:
		changes.Parent = &detached
	}
	if named["body"] || named["body-file"] {
		body, code := readBody(env, *bodyFlag, *bodyFileFlag)
		if code != exitOK {
			return code
		}
		changes.Body = &body
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
	out, err := svc.Update(pos[0], changes)
	warnSkipped(env, out.Warnings)
	if err != nil {
		return updateError(env, err)
	}

	// An unchanged outcome is the net-change rule: the update landed where it
	// started, so nothing was written and `updated` stands.
	line := fmt.Sprintf("Updated %s", out.Issue.ID)
	if !out.Changed {
		line = fmt.Sprintf("%s is unchanged", out.Issue.ID)
	}
	return reportIssue(env, format, out.Issue, line)
}

// updateError maps update's own refusal — a relationship change that would close
// a cycle — onto its diagnostic and exit code. The store is sound and the
// reference resolved; what was asked for is an issue graph that cannot exist, so
// it is a usage error like any other bad invocation.
func updateError(env Env, err error) int {
	var cycle *core.CycleError
	if errors.As(err, &cycle) {
		errf(env, "%v", cycle)
		return exitUsage
	}
	return coreError(env, err)
}

// namedFlags reports which flags the invocation actually named, across however
// many passes parseArgs took to collect them. Presence is what marks a field for
// change, so a flag given an empty value is told apart from one left off.
func namedFlags(fs *flag.FlagSet) map[string]bool {
	named := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { named[f.Name] = true })
	return named
}

// changesRequested reports whether the invocation named any field flag.
func changesRequested(named map[string]bool) bool {
	for _, name := range mutationFlags {
		if named[name] {
			return true
		}
	}
	return false
}

// setChange is one repeatable set flag's values split into additions and
// removals.
type setChange struct{ add, remove []string }

// parseSetChange splits a set flag's values by their prefix: a bare or
// "+"-prefixed value adds, a "-"-prefixed one removes. Both halves travel to the
// core, which decides what survives — a value named in both is removed — so a
// caller can add and drop entries in one invocation without resending the ones
// it does not know about.
func parseSetChange(name string, values []string) (setChange, error) {
	var c setChange
	for _, v := range values {
		value, remove := strings.CutPrefix(v, "-")
		if !remove {
			value = strings.TrimPrefix(v, "+")
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return setChange{}, fmt.Errorf("--%s %q names no value", name, v)
		}
		if remove {
			c.remove = append(c.remove, value)
			continue
		}
		c.add = append(c.add, value)
	}
	return c, nil
}
