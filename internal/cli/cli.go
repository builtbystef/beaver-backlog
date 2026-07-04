// Package cli is the engine that turns a command invocation into actions on the
// store and rendered output. Everything it depends on (arguments, stdio, the
// working directory, the environment, the clock, and the ID generator) arrives
// through an Env struct.
package cli

import (
	"flag"
	"fmt"
	"io"

	"beaver/internal/clock"
	"beaver/internal/vcs"
)

// Exit codes. 0 is success; the rest are stable so scripts and agents can branch
// on outcome without parsing text (ADR 0013).
const (
	exitOK       = 0
	exitError    = 1 // an unexpected or runtime failure
	exitUsage    = 2 // the command was invoked incorrectly
	exitNotFound = 3 // a referenced issue (or the store) was not found
)

// Env bundles everything a command run depends on. Production wires these to the
// real process; the test harness wires them to buffers, a temp directory, and a
// fixed clock.
type Env struct {
	Args          []string            // arguments after the program name
	Stdin         io.Reader           // interactive input (identity confirmation/prompt)
	Stdout        io.Writer           // command output
	Stderr        io.Writer           // diagnostics, errors, and interactive prompts
	WorkDir       string              // directory the store is resolved from
	Getenv        func(string) string // environment lookup
	Clock         clock.Clock         // source of timestamps
	NewID         func() string       // issue ID generator (injectable for tests)
	Edit          func(string) error  // open a file in the user's editor, blocking until it exits; nil (or a non-interactive session) means no editor, so edit and interactive create refuse rather than hang
	StdoutIsTTY   bool                // whether stdout is an interactive terminal
	StdinIsTTY    bool                // whether stdin is interactive: the signal that gates human identity setup (ADR 0010)
	VCS           vcs.System          // version-control integration (identity seed + commit); nil means no adapter (ADR 0006/0007)
	UserConfigDir string              // per-machine user-config dir (identity lives here, never committed; ADR 0008)
}

// Run dispatches one command and returns its exit code. It never calls os.Exit;
// main translates the returned code into a process exit.
func Run(env Env) int {
	if len(env.Args) == 0 {
		printUsage(env.Stderr)
		return exitUsage
	}
	cmd, args := env.Args[0], env.Args[1:]
	switch cmd {
	case "init":
		return cmdInit(env, args)
	case "create":
		return cmdCreate(env, args)
	case "list":
		return cmdList(env, args)
	case "show":
		return cmdShow(env, args)
	case "done":
		return cmdDone(env, args)
	case "cancel":
		return cmdCancel(env, args)
	case "reopen":
		return cmdReopen(env, args)
	case "claim":
		return cmdClaim(env, args)
	case "assign":
		return cmdAssign(env, args)
	case "release":
		return cmdRelease(env, args)
	case "start":
		return cmdStart(env, args)
	case "priority":
		return cmdPriority(env, args)
	case "label":
		return cmdLabel(env, args)
	case "edit":
		return cmdEdit(env, args)
	case "delete":
		return cmdDelete(env, args)
	case "note":
		return cmdNote(env, args)
	case "whoami":
		return cmdWhoami(env, args)
	case "help", "-h", "--help":
		printUsage(env.Stdout)
		return exitOK
	default:
		fmt.Fprintf(env.Stderr, "beaver: unknown command %q\n\n", cmd)
		printUsage(env.Stderr)
		return exitUsage
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `beaver: a local-first, file-based issue tracker

usage:
  beaver init                 initialize a store in the current project
  beaver create "<title>"     create a new issue
  beaver list                 list issues (default: all)
  beaver show <ref>           show an issue by ID or slug
  beaver done <ref>           mark an issue done
  beaver cancel <ref>         cancel an issue (deliberately abandon it)
  beaver reopen <ref>         return a done or cancelled issue to todo
  beaver claim <ref>          claim an issue (make yourself its assignee)
  beaver assign <ref> <actor> assign an issue to a named actor
  beaver release <ref>        clear an issue's assignee
  beaver start <ref>          start an issue (in-progress; auto-claims if unowned)
  beaver priority <ref> <lvl> set or clear priority (urgent|high|medium|low|none)
  beaver label <ref> <label>  add labels (or remove them with --remove)
  beaver edit <ref>           open an issue in $EDITOR for freeform hand-editing
  beaver delete <ref>         delete an issue's file (for junk; the VCS keeps history)
  beaver note <ref> "<text>"  append a note to an issue's coordination log
  beaver whoami               print the actor Busy Beaver resolves you as

common flags (after the command):
  --format human|json         override output format (default: auto-detect)

create flags:
  --label <label>             tag with a label (free-form; repeatable, comma-separated)
  --priority <level>          set priority: urgent|high|medium|low
  --depends-on <ref>          depend on an issue (repeatable, comma-separated)
  --parent <ref>              set the parent issue (makes this a sub-issue)

list flags:
  --state <state>             filter: all|todo|in-progress|done|cancelled
  --ready                     only ready issues (todo, every dependency done)
  --blocked                   only blocked issues (todo, an unmet dependency)
  --label <label>             only issues carrying every named label (repeatable)
  --priority <level>          only issues at this priority (none = unprioritized)
  --assignee <actor>          only issues assigned to this actor
  issues are ordered by priority (urgent first), then oldest first

label flags:
  --remove <label>            remove a label instead of adding (repeatable, comma-separated)

claim / start flags:
  --as <actor>                act as this actor (overrides identity detection)
  --force                     steal an issue already claimed by another actor

whoami flags:
  --as <actor>                resolve as this actor (overrides all detection)

note flags:
  --as <actor>                attribute the note to this actor (overrides detection)

a <ref> is a full issue ID, its slug, or the full <id>-<slug> name.
show reports what an issue is waiting on and whether it is ready or blocked.
run "beaver create" with no title in a terminal to author the issue in $EDITOR;
a non-interactive create still requires a title argument.

exit codes:
  0  success
  1  an unexpected or runtime failure
  2  the command was invoked incorrectly
  3  a referenced issue (or the store) was not found
`)
}

// newFlagSet builds a flag set wired to the env's stderr, with the shared
// --format flag registered.
func newFlagSet(env Env, name string) (fs *flag.FlagSet, format *string) {
	fs = flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	format = fs.String("format", "", "output format: human|json")
	return fs, format
}

// parseArgs parses flags that may appear anywhere among the positional
// arguments, returning the positionals. The standard flag package stops at the
// first positional, which would reject the natural `beaver show <ref> --format
// json`;
func parseArgs(fs *flag.FlagSet, args []string) (positionals []string, ok bool) {
	var literals []string
	for i, a := range args {
		if a == "--" {
			args, literals = args[:i], args[i+1:]
			break
		}
	}
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return append(positionals, literals...), true
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
}

// errf prints a "beaver: …" diagnostic to stderr.
func errf(env Env, format string, a ...any) {
	fmt.Fprintf(env.Stderr, "beaver: "+format+"\n", a...)
}
