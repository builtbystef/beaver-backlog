package cli

import (
	"errors"
	"fmt"
	"path/filepath"
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
		fmt.Fprintf(env.Stdout, "Initialized empty Beaver store in %s\n", root)
	} else {
		fmt.Fprintf(env.Stdout, "Reinitialized existing Beaver store in %s\n", root)
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

	iss, _, err := st.Resolve(ref)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			errf(env, "no issue found matching %q", ref)
			return exitNotFound
		}
		errf(env, "%v", err)
		return exitError
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
		errf(env, "not a Beaver store; run `beaver init`")
		return exitError
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
