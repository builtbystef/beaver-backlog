package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/output"
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
	priority, err := core.ParsePriority(*priorityFlag)
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

	svc, err := open(env)
	if err != nil {
		return coreError(env, err)
	}
	draft := core.Draft{
		Title:     title,
		Body:      body,
		Labels:    labels.values,
		Priority:  priority,
		DependsOn: dependsOn.values,
		Parent:    *parentFlag,
	}

	// With no title, the gate above has already guaranteed an interactive
	// session with an editor, so the human supplies it there — a --body given
	// alongside no title seeds the skeleton the editor opens on.
	if title == "" {
		created, code := authorInEditor(env, svc, draft)
		if code != exitOK {
			return code
		}
		return reportCreated(env, format, created)
	}

	created, err := svc.Create(draft)
	warnSkipped(env, created.Warnings)
	if err != nil {
		return coreError(env, err)
	}
	return reportCreated(env, format, created)
}

// reportCreated renders a created issue: a confirmation naming the file for a
// human, or the issue itself as JSON for a machine.
func reportCreated(env Env, format output.Format, created core.Created) int {
	if format == output.JSON {
		if err := output.WriteIssue(env.Stdout, created.Issue, output.JSON); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintf(env.Stdout, "Created %s  %s\n  %s\n",
		created.Issue.ID, created.Issue.Title, relPath(env.WorkDir, created.Path))
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
