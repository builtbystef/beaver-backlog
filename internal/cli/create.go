package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
)

// cmdCreate mints a new issue from a title, optionally wiring its one-sided
// relationship edges: --depends-on names issues this one waits on, --parent the
// issue it is a sub-issue of. Both are stored on this issue alone; the inverse is
// derived, never written (ADR 0011).
func cmdCreate(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "create")
	var dependsOn csvList
	fs.Var(&dependsOn, "depends-on", "issue this depends on (a ref; repeatable, comma-separated)")
	parentFlag := fs.String("parent", "", "parent issue this is a sub-issue of (a ref)")
	var labels csvList
	fs.Var(&labels, "label", "label to tag this issue with (free-form; repeatable, comma-separated)")
	priorityFlag := fs.String("priority", "", "priority: urgent|high|medium|low")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	// Validate the priority up front, before any store work, so a bad value fails
	// fast as a usage error. An omitted flag ("") maps to the unprioritized empty
	// value, so create defaults to no priority.
	priority, err := parsePriority(*priorityFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	// A title comes from the command line or, in an interactive session, from the
	// editor create opens on the new file. Exactly one positional is the title; none
	// is allowed only when an editor can supply it — otherwise create still requires a
	// title, and says so before any store work (a non-interactive create, ADR 0010).
	var title string
	switch len(pos) {
	case 1:
		title = strings.TrimSpace(pos[0])
		if title == "" {
			errf(env, "title must not be empty")
			return exitUsage
		}
	case 0:
		if editorGate(env) != nil {
			errf(env, "create requires a title: beaver create \"<title>\"")
			return exitUsage
		}
	default:
		errf(env, "create takes a single title argument (quote it): beaver create \"<title>\"")
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

	// One snapshot answers everything create asks of the store — each edge, the
	// parent, and the id-collision check — so the whole command scans the files
	// once, not once per question, and every answer comes from the same instant.
	snap, err := st.Snapshot()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	// Resolve the relationship references to canonical ids before minting anything,
	// so a typo'd --depends-on/--parent fails fast and the stored edges hold real
	// ids (depends_on and parent store ids, never slugs). Every issue-addressing
	// input routes through the shared resolver, so an edge accepts the same
	// references show and done do.
	deps, code := resolveEdges(env, snap, dependsOn.values)
	if code != exitOK {
		return code
	}
	var parent string
	if ref := strings.TrimSpace(*parentFlag); ref != "" {
		p, _, c := resolveRef(env, snap, ref)
		if c != exitOK {
			return c
		}
		parent = p.ID
	}

	id, err := mintID(env, snap)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	now := env.Clock.Now().UTC().Truncate(time.Second)
	iss := issue.Issue{
		ID:        id,
		Title:     title, // empty in the interactive editor path; the human supplies it
		State:     issue.StateTodo,
		Priority:  priority,
		Labels:    dedupe(labels.values), // nil when none, so the field marshals away
		DependsOn: deps,
		Parent:    parent,
		Created:   now,
		Updated:   now,
	}

	// With a title in hand, write the issue directly. With none, the gate above has
	// already guaranteed an interactive session with an editor: seed the file, hand it
	// to $EDITOR, and take the title and body the human writes back (authorInEditor).
	var path string
	if title != "" {
		if path, err = st.Write(iss); err != nil {
			errf(env, "%v", err)
			return exitError
		}
	} else {
		var code int
		if iss, path, code = authorInEditor(env, st, iss); code != exitOK {
			return code
		}
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

// mintID generates a fresh ID, retrying on the rare collision with an issue in
// the snapshot. The bound guards against a pathological generator rather than a
// real store ever filling up.
func mintID(env Env, snap *store.Snapshot) (string, error) {
	for range 100 {
		id := env.NewID()
		if !snap.IDTaken(id) {
			return id, nil
		}
	}
	return "", errors.New("could not generate a unique issue ID")
}
