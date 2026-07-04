package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
	"beaver/internal/userconfig"
)

// cmdInit initializes the store in the working directory. It is idempotent.
func cmdInit(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "init")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "init takes no arguments")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	root, created, err := store.Init(env.WorkDir)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	// Proactively establish the runner's identity: init is the moment to run step 4
	// of resolution for the person setting up, so the common solo case is "one
	// command and you're ready" (ADR 0008). This only ever seeds interactively and
	// only when nothing is saved yet — a non-interactive init (agent or CI) neither
	// prompts nor borrows the human's VCS name (ADR 0010).
	seeded := seedIdentity(env)

	if format == output.JSON {
		result := map[string]any{"store_path": root, "created": created}
		if seeded != "" {
			result["actor"] = seeded
		}
		if err := output.WriteJSON(env.Stdout, result); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	if created {
		fmt.Fprintf(env.Stdout, "Initialized empty Busy Beaver store in %s\n", root)
	} else {
		fmt.Fprintf(env.Stdout, "Reinitialized existing Busy Beaver store in %s\n", root)
	}
	if seeded != "" {
		fmt.Fprintf(env.Stdout, "Identity set to %q (saved to %s, never committed).\n", seeded, userconfig.Path(env.UserConfigDir))
	}
	return exitOK
}

// seedIdentity establishes the runner's saved identity when init can — an
// interactive session with none saved yet — and returns the name it saved, or ""
// when it does nothing. It never fails init: identity setup is a convenience laid
// over a store that is already created, so a declined or unreadable prompt only
// warns and leaves the store initialized.
func seedIdentity(env Env) string {
	if !env.StdinIsTTY {
		return "" // never seed non-interactively (ADR 0010)
	}
	cfg, err := userconfig.Load(env.UserConfigDir)
	if err != nil || cfg.Actor != "" {
		return "" // already set, or unreadable: leave it as-is
	}
	name, err := establishHumanIdentity(env)
	if err != nil {
		errf(env, "identity not set: %v", err)
		return ""
	}
	return name
}

// refList collects a repeatable, comma-separated flag of issue references, e.g.
// `--depends-on a,b --depends-on c`. Each Set splits on commas and appends the
// trimmed, non-empty references in order; resolving them to canonical ids and
// deduping happens later, once the store is open.
type refList struct{ refs []string }

func (r *refList) String() string { return strings.Join(r.refs, ",") }

func (r *refList) Set(v string) error {
	for part := range strings.SplitSeq(v, ",") {
		if ref := strings.TrimSpace(part); ref != "" {
			r.refs = append(r.refs, ref)
		}
	}
	return nil
}

// cmdCreate mints a new issue from a title, optionally wiring its one-sided
// relationship edges: --depends-on names issues this one waits on, --parent the
// issue it is a sub-issue of. Both are stored on this issue alone; the inverse is
// derived, never written (ADR 0011).
func cmdCreate(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "create")
	var dependsOn refList
	fs.Var(&dependsOn, "depends-on", "issue this depends on (a ref; repeatable, comma-separated)")
	parentFlag := fs.String("parent", "", "parent issue this is a sub-issue of (a ref)")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	switch len(pos) {
	case 1: // exactly one title argument
	case 0:
		errf(env, "create requires a title: beaver create \"<title>\"")
		return exitUsage
	default:
		errf(env, "create takes a single title argument (quote it): beaver create \"<title>\"")
		return exitUsage
	}
	title := strings.TrimSpace(pos[0])
	if title == "" {
		errf(env, "title must not be empty")
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

	// Resolve the relationship references to canonical ids before minting anything,
	// so a typo'd --depends-on/--parent fails fast and the stored edges hold real
	// ids (depends_on and parent store ids, never slugs). Every issue-addressing
	// input routes through the shared resolver, so an edge accepts the same
	// references show and done do.
	deps, code := resolveEdges(env, st, dependsOn.refs)
	if code != exitOK {
		return code
	}
	var parent string
	if ref := strings.TrimSpace(*parentFlag); ref != "" {
		p, _, c := resolveRef(env, st, ref)
		if c != exitOK {
			return c
		}
		parent = p.ID
	}

	id, err := mintID(env, st)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	now := env.Clock.Now().UTC().Truncate(time.Second)
	iss := issue.Issue{
		ID:        id,
		Title:     title,
		State:     issue.StateTodo,
		DependsOn: deps,
		Parent:    parent,
		Created:   now,
		Updated:   now,
	}
	path, err := st.Write(iss)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	if format == output.JSON {
		if err := output.WriteIssue(env.Stdout, iss, output.JSON); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintf(env.Stdout, "Created %s  %s\n  %s\n", iss.ID, iss.Title, relPath(env.WorkDir, path))
	return exitOK
}

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
	// state filter — --ready already implies todo.
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
	issues := selectIssues(all, *readyFlag, *blockedFlag, match)
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

// sortIssues orders issues deterministically for display: oldest first by creation
// time, with the stable random ID as a total-order tiebreak so issues minted at
// the same instant (common under a fixed test clock) still sort reproducibly.
// Priority-aware ordering arrives with p1k765.
func sortIssues(issues []issue.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if !a.Created.Equal(b.Created) {
			return a.Created.Before(b.Created)
		}
		return a.ID < b.ID
	})
}

// cmdShow renders one issue resolved from a reference.
func cmdShow(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "show")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) != 1 {
		errf(env, "show requires an issue reference: beaver show <ref>")
		return exitUsage
	}
	ref := pos[0]
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := discover(env)
	if err != nil {
		return storeError(env, err)
	}

	iss, _, code := resolveRef(env, st, ref)
	if code != exitOK {
		return code
	}

	// Enrich the view with the derived relationship facts show is the natural home
	// for: what this issue is waiting on, whether it is ready/blocked/stuck, and the
	// inverse edges (what it blocks, its children) that are never stored (ADR 0011).
	// Deriving them needs the whole store, so read it and index it; the resolved
	// issue is among them.
	all, err := st.ReadAll()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	rel := issue.NewRelations(all).For(iss)

	if err := output.WriteIssueWithRelationship(env.Stdout, iss, rel, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
}

// mintID generates a fresh ID, retrying on the rare collision with an existing
// issue. The bound guards against a pathological generator rather than a real
// store ever filling up.
func mintID(env Env, st *store.Store) (string, error) {
	for range 100 {
		id := env.NewID()
		taken, err := st.IDTaken(id)
		if err != nil {
			return "", err
		}
		if !taken {
			return id, nil
		}
	}
	return "", errors.New("could not generate a unique issue ID")
}

func storeError(env Env, err error) int {
	if errors.Is(err, store.ErrNoStore) {
		errf(env, "not a Busy Beaver store; run `beaver init`")
		return exitNotFound
	}
	errf(env, "%v", err)
	return exitError
}

// discover finds the store from the working directory and wires it to report the
// invalid files it skips as loud stderr warnings (ADR 0005), so every command
// that reads the store degrades gracefully and visibly — it keeps working on the
// valid issues but never hides a broken file or lets it brick the command. It
// returns the same (store, error) as store.Discover; callers map the error with
// storeError.
func discover(env Env) (*store.Store, error) {
	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return nil, err
	}
	st.OnWarn(warnInvalid(env))
	return st, nil
}

// warnInvalid builds the store's warning handler for one command run: it prints
// each skipped file once, loudly, to stderr, naming the file and the specific
// problem (ADR 0005). It writes to stderr, never stdout, so a warning never
// corrupts the JSON an agent parses (ADR 0013). Dedup is by path, so a command
// that scans the store more than once (create's id-collision loop) still warns a
// given file only once.
func warnInvalid(env Env) func(store.Warning) {
	seen := make(map[string]bool)
	return func(w store.Warning) {
		if seen[w.Path] {
			return
		}
		seen[w.Path] = true
		errf(env, "skipping invalid issue %s: %v", relPath(env.WorkDir, w.Path), w.Err)
	}
}

// relPath renders path relative to workDir when it sits inside it, for friendlier
// human output, and falls back to the absolute path otherwise.
func relPath(workDir, path string) string {
	if rel, err := filepath.Rel(workDir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
