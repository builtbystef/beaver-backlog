package cli

import (
	"fmt"
	"sort"

	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// cmdList enumerates issues. By default (or with --state) it filters by state;
// --ready and --blocked instead select the two halves of the unstarted work over
// the dependency graph. The label, priority, and assignee filters refine any base
// selector.
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
	match, err := stateFilter(*stateFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	attr, err := attrFilter(labelFilter.values, *priorityFilter, *assigneeFilter)
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

	all, err := st.ReadAll()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	issues := attr(selectIssues(all, *readyFlag, *blockedFlag, match))
	sortIssues(issues)

	if err := output.WriteList(env.Stdout, issues, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}

// selectIssues applies the active selector: the ready queue (todo with every
// dependency done), the blocked queue (todo with an unmet dependency), or, by
// default, the state predicate. Both queues are scoped to todo because they answer
// "what unstarted work can I pick up"; show and doctor still surface a blocked
// in-progress issue.
func selectIssues(all []issue.Issue, ready, blocked bool, match func(issue.State) bool) []issue.Issue {
	out := make([]issue.Issue, 0, len(all))
	switch {
	case ready:
		rel := issue.NewRelations(all)
		for _, iss := range all {
			if rel.Ready(iss) {
				out = append(out, iss)
			}
		}
	case blocked:
		rel := issue.NewRelations(all)
		for _, iss := range all {
			if iss.State == issue.StateTodo && rel.Blocked(iss) {
				out = append(out, iss)
			}
		}
	default:
		for _, iss := range all {
			if match(iss.State) {
				out = append(out, iss)
			}
		}
	}
	return out
}

// stateFilter turns a --state value into a predicate over issue state. An omitted
// value and the explicit "all" match every state; anything else must be one of the
// four concrete states.
func stateFilter(value string) (func(issue.State) bool, error) {
	switch value {
	case "", "all":
		return func(issue.State) bool { return true }, nil
	case string(issue.StateTodo), string(issue.StateInProgress), string(issue.StateDone), string(issue.StateCancelled):
		want := issue.State(value)
		return func(s issue.State) bool { return s == want }, nil
	default:
		return nil, fmt.Errorf("invalid state %q (want one of: all, todo, in-progress, done, cancelled)", value)
	}
}

// attrFilter builds the attribute refinement list applies after its base selector:
// keep only issues carrying every label in wantLabels, at the wanted priority, and
// assigned to wantAssignee. An inactive dimension constrains nothing. The returned
// function preserves order and always returns a non-nil slice, so an empty match
// still renders as [] / "No issues.".
func attrFilter(wantLabels []string, priorityValue, wantAssignee string) (func([]issue.Issue) []issue.Issue, error) {
	// --priority "none" (match the unprioritized) must stay distinct from an
	// omitted flag (match any priority), but parsePriority folds both "" and
	// "none" to the empty Priority, so activeness is tracked apart from the value.
	priorityActive := priorityValue != ""
	var wantPriority issue.Priority
	if priorityActive {
		p, err := parsePriority(priorityValue)
		if err != nil {
			return nil, err
		}
		wantPriority = p
	}
	labels := dedupe(wantLabels)
	return func(in []issue.Issue) []issue.Issue {
		out := make([]issue.Issue, 0, len(in))
		for _, iss := range in {
			if priorityActive && iss.Priority != wantPriority {
				continue
			}
			if wantAssignee != "" && iss.Assignee != wantAssignee {
				continue
			}
			if !hasAllLabels(iss.Labels, labels) {
				continue
			}
			out = append(out, iss)
		}
		return out
	}, nil
}

// hasAllLabels reports whether have carries every label in want (AND semantics).
// An empty want matches everything.
func hasAllLabels(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := make(map[string]bool, len(have))
	for _, l := range have {
		set[l] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

// sortIssues orders issues by priority (urgent first, unprioritized last), then
// oldest first, with the ID as a tiebreak so issues minted at the same instant
// still sort reproducibly.
func sortIssues(issues []issue.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if ra, rb := a.Priority.Rank(), b.Priority.Rank(); ra != rb {
			return ra < rb
		}
		if !a.Created.Equal(b.Created) {
			return a.Created.Before(b.Created)
		}
		return a.ID < b.ID
	})
}
