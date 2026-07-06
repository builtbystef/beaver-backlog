package cli

import (
	"fmt"
	"sort"

	"beaver/internal/issue"
	"beaver/internal/output"
)

// cmdList enumerates issues under one of three selectors. By default (or with
// --state) it filters by state: no flag and the explicit "all" list every issue,
// a concrete state narrows to it. --ready and --blocked instead select over the
// dependency graph — the two halves of the unstarted (todo) work: the ready queue
// (every dependency done) and the blocked queue (some dependency not done). Output
// is a human table or a JSON array, auto-detected.
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
	// The ready and blocked queues each define their own selection over the
	// dependency graph, so they are mutually exclusive and do not stack with the
	// state filter — --ready already implies todo. The attribute filters below
	// (label, priority, assignee) are refinements, so they do stack with any base
	// selector.
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

// selectIssues applies the active list selector to all issues: the ready queue
// (todo with every dependency done), the blocked queue (todo with an unmet
// dependency), or, by default, the state predicate. The two queues partition the
// todo set — every todo issue is ready or blocked, never both — deriving readiness
// from the dependency graph over the whole set (issue.Relations). Both are scoped
// to todo because they answer "what unstarted work can I pick up"; an in-progress
// issue is already being worked, and a closed one is done, so neither queues them
// even when an edge is unmet (show and doctor still surface a blocked in-progress
// issue as the anomaly it is).
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
// value (the default) and the explicit "all" match every state; otherwise the value
// must be one of the four concrete states. An unrecognized value is a usage error.
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

// attrFilter builds the attribute refinement that list applies after its base
// selector: keep only issues carrying every label in wantLabels, at the wanted
// priority, and assigned to wantAssignee. Each dimension is independent and any may
// be inactive — no labels, an empty priority string, an empty assignee — and
// constrains nothing when so, leaving the default list unfiltered. The priority is
// validated here (the four levels, plus "none" to select the unprioritized), the
// one place list rejects a bad filter flag. The returned function preserves order
// and always returns a non-nil slice, so an empty match still renders as [] /
// "No issues.".
func attrFilter(wantLabels []string, priorityValue, wantAssignee string) (func([]issue.Issue) []issue.Issue, error) {
	// An explicit --priority is validated and, uniquely, distinguishes "none" (match
	// the unprioritized) from an omitted flag (match any priority): parsePriority
	// folds both "" and "none" to the empty Priority, so activeness is tracked apart
	// from the value it resolves to.
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

// hasAllLabels reports whether have carries every label in want (AND semantics), so
// `--label a --label b` narrows to issues tagged both. An empty want matches every
// issue, so an inactive label filter keeps all.
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

// sortIssues orders issues deterministically for display: highest priority first
// (urgent > high > medium > low, then the unprioritized), and within one priority
// the stable creation order — oldest first, with the random ID as a total-order
// tiebreak so issues minted at the same instant (common under a fixed test clock)
// still sort reproducibly. Priority leads because list is a triage view: what to
// pick up next sorts to the top. Issues with no priority set all share the lowest
// rank, so a store that sets none keeps the pure creation order it had before.
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
