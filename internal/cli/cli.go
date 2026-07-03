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
	Args        []string            // arguments after the program name
	Stdin       io.Reader           // reserved for interactive commands (later slices)
	Stdout      io.Writer           // command output
	Stderr      io.Writer           // diagnostics and errors
	WorkDir     string              // directory the store is resolved from
	Getenv      func(string) string // environment lookup
	Clock       clock.Clock         // source of timestamps
	NewID       func() string       // issue ID generator (injectable for tests)
	StdoutIsTTY bool                // whether stdout is an interactive terminal
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
	case "show":
		return cmdShow(env, args)
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
  beaver show <ref>           show an issue by its ID

global flags:
  --format human|json         override output format (default: auto-detect)
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
// json`; pulling positionals out one at a time and re-parsing the remainder lets
// flags sit on either side. ok is false when a flag failed to parse (flag has
// already reported why), and the caller should exit with a usage error.
func parseArgs(fs *flag.FlagSet, args []string) (positionals []string, ok bool) {
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positionals, true
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
}

// errf prints a "beaver: …" diagnostic to stderr.
func errf(env Env, format string, a ...any) {
	fmt.Fprintf(env.Stderr, "beaver: "+format+"\n", a...)
}
