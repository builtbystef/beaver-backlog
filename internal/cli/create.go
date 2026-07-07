package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/builtbystef/busy-beaver/internal/issue"
	"github.com/builtbystef/busy-beaver/internal/output"
	"github.com/builtbystef/busy-beaver/internal/store"
)

// cmdCreate mints a new issue from a title, optionally wiring relationship
// edges: --depends-on names issues this one waits on, --parent the issue it is
// a sub-issue of. Both are stored on this issue alone; the inverse is derived,
// never written. The body can be supplied at creation with --body or
// --body-file, so a non-interactive caller can author a complete issue in one
// command.
func cmdCreate(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "create")
	var dependsOn csvList
	fs.Var(&dependsOn, "depends-on", "issue this depends on (a ref; repeatable, comma-separated)")
	parentFlag := fs.String("parent", "", "parent issue this is a sub-issue of (a ref)")
	var labels csvList
	fs.Var(&labels, "label", "label to tag this issue with (free-form; repeatable, comma-separated)")
	priorityFlag := fs.String("priority", "", "priority: urgent|high|medium|low")
	bodyFlag := fs.String("body", "", "issue body (free-form Markdown)")
	bodyFileFlag := fs.String("body-file", "", "read the issue body from a file, or - for stdin")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	// Validate the priority before any store work so a bad value fails fast.
	priority, err := parsePriority(*priorityFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}
	body, code := readBody(env, *bodyFlag, *bodyFileFlag)
	if code != exitOK {
		return code
	}
	// Zero positionals is allowed only when an interactive editor can supply
	// the title; otherwise create requires one and fails before any store work.
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

	// One snapshot answers every store question — edges, parent, id collision —
	// so the command scans the files once, not once per question.
	snap, err := st.Snapshot()
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	// Resolve relationship references to canonical ids before minting anything,
	// so a typo'd --depends-on/--parent fails fast and the stored edges hold
	// real ids, never slugs.
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
		Body:      body,
	}

	// With no title, the gate above has already guaranteed an interactive
	// session with an editor, so authorInEditor can supply it — a --body given
	// alongside no title seeds the skeleton the editor opens on.
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

// readBody resolves the mutually exclusive --body/--body-file pair into the
// issue body, taken verbatim as free-form Markdown. --body-file reads a path
// resolved against the working directory, or stdin for "-", so an agent can
// pipe multi-line Markdown without shell-quoting it into an argument.
func readBody(env Env, inline, file string) (string, int) {
	if inline != "" && file != "" {
		errf(env, "--body and --body-file are mutually exclusive")
		return "", exitUsage
	}
	switch {
	case file == "-":
		data, err := io.ReadAll(env.Stdin)
		if err != nil {
			errf(env, "could not read body from stdin: %v", err)
			return "", exitError
		}
		return string(data), exitOK
	case file != "":
		if !filepath.IsAbs(file) {
			file = filepath.Join(env.WorkDir, file)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			errf(env, "could not read body file: %v", err)
			return "", exitError
		}
		return string(data), exitOK
	default:
		return inline, exitOK
	}
}

// mintID generates a fresh ID, retrying on the rare collision with an existing
// issue. The bound guards against a pathological generator, not a full store.
func mintID(env Env, snap *store.Snapshot) (string, error) {
	for range 100 {
		id := env.NewID()
		if !snap.IDTaken(id) {
			return id, nil
		}
	}
	return "", errors.New("could not generate a unique issue ID")
}
