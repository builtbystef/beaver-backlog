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
	format, err := output.Resolve(*formatFlag, env.StdoutIsTTY, env.Getenv)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	root, created, err := store.Init(env.WorkDir)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	if format == output.JSON {
		if err := output.WriteJSON(env.Stdout, map[string]any{"store_path": root, "created": created}); err != nil {
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
	return exitOK
}

// cmdCreate mints a new issue from a title.
func cmdCreate(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "create")
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
	format, err := output.Resolve(*formatFlag, env.StdoutIsTTY, env.Getenv)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return storeError(env, err)
	}

	id, err := mintID(env, st)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	now := env.Clock.Now().UTC().Truncate(time.Second)
	iss := issue.Issue{
		ID:      id,
		Title:   title,
		State:   issue.StateTodo,
		Created: now,
		Updated: now,
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

// cmdList enumerates issues, optionally filtered by state. With no --state it lists
// all issues; --state narrows to a single concrete state (todo, in-progress, done,
// cancelled) or the explicit all. Output is a human table or a JSON array,
// auto-detected.
func cmdList(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "list")
	stateFlag := fs.String("state", "", "filter by state: all|todo|in-progress|done|cancelled")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "list takes no positional arguments (did you mean --state %s?)", pos[0])
		return exitUsage
	}
	match, err := stateFilter(*stateFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	format, err := output.Resolve(*formatFlag, env.StdoutIsTTY, env.Getenv)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return storeError(env, err)
	}

	all, err := st.ReadAll()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	issues := make([]issue.Issue, 0, len(all))
	for _, iss := range all {
		if match(iss.State) {
			issues = append(issues, iss)
		}
	}
	sortIssues(issues)

	if err := output.WriteList(env.Stdout, issues, format); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	return exitOK
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
	format, err := output.Resolve(*formatFlag, env.StdoutIsTTY, env.Getenv)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return storeError(env, err)
	}

	iss, _, code := resolveRef(env, st, ref)
	if code != exitOK {
		return code
	}

	if err := output.WriteIssue(env.Stdout, iss, format); err != nil {
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

// relPath renders path relative to workDir when it sits inside it, for friendlier
// human output, and falls back to the absolute path otherwise.
func relPath(workDir, path string) string {
	if rel, err := filepath.Rel(workDir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
