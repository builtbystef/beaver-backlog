package cli

import (
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// cmdList enumerates issues. By default (or with --state) it filters by state;
// --ready and --blocked instead select the two halves of the unstarted work over
// the dependency graph. The label, priority, and assignee filters refine any base
// selector. Selection and ordering belong to the core; this handler only turns
// flags into a query.
func cmdList(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "list")
	stateFlag := fs.String("state", "", "filter by state: all|todo|in-progress|done|cancelled")
	readyFlag := fs.Bool("ready", false, "only ready issues: todo with every dependency done")
	blockedFlag := fs.Bool("blocked", false, "only blocked issues: todo with an unmet dependency")
	var labelFilter csvList
	fs.Var(&labelFilter, "label", "only issues carrying every named label (repeatable, comma-separated)")
	priorityFilter := fs.String("priority", "", "only issues at this priority: urgent|high|medium|low|none")
	assigneeFilter := fs.String("assignee", "", "only issues assigned to this actor")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "list takes no positional arguments (did you mean --state %s?)", pos[0])
		return exitUsage
	}
	// Ready and blocked each define their own selection, so they combine neither
	// with each other nor with --state (--ready already implies todo). The
	// attribute filters are refinements and stack with any base selector.
	if *readyFlag && *blockedFlag {
		errf(env, "--ready and --blocked are mutually exclusive")
		return exitUsage
	}
	if (*readyFlag || *blockedFlag) && *stateFlag != "" {
		errf(env, "--state does not combine with --ready or --blocked")
		return exitUsage
	}
	q := core.Query{Ready: *readyFlag, Blocked: *blockedFlag, Labels: labelFilter.values}
	states, err := parseStates(*stateFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	q.States = states
	// An omitted --priority constrains nothing while --priority none selects the
	// unprioritized; both parse to the empty Priority, so the flag's presence is
	// what tells them apart.
	if *priorityFilter != "" {
		level, err := parsePriority(*priorityFilter)
		if err != nil {
			errf(env, "%v", err)
			return exitUsage
		}
		q.Priorities = []issue.Priority{level}
	}
	if *assigneeFilter != "" {
		q.Assignee = assigneeFilter
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
	listing, err := svc.List(q)
	warnSkipped(env, listing.Warnings)
	if err != nil {
		return coreError(env, err)
	}

	if err := output.WriteList(env.Stdout, listing.Issues, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}

// parseStates turns a --state value into the states a query selects. An omitted
// value and the explicit "all" select every state; anything else must be one of
// the four concrete states.
func parseStates(value string) ([]issue.State, error) {
	switch value {
	case "", "all":
		return nil, nil
	case string(issue.StateTodo), string(issue.StateInProgress), string(issue.StateDone), string(issue.StateCancelled):
		return []issue.State{issue.State(value)}, nil
	default:
		return nil, fmt.Errorf("invalid state %q (want one of: all, todo, in-progress, done, cancelled)", value)
	}
}
