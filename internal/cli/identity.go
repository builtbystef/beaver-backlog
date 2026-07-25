package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/output"
	"github.com/builtbystef/beaver-backlog/internal/userconfig"
)

// Identity resolution answers "who is doing this?" for every command that
// attributes work, through one precedence chain:
//
//  1. --as <actor>         explicit, always wins
//  2. BEAVER_BACKLOG_ACTOR    explicit override (agents, CI)
//  3. agent detection      AGENT, else a known marker (CLAUDECODE → claude, …)
//  4. interactive human    the saved user-config identity; if unset, prompt, then save
//  5. non-interactive      a loud generic `agent` when nothing above matched
//
// The human's stored identity (step 4) is consulted only in an interactive
// session — a non-interactive run never borrows it — and a human's identity is
// never placed in BEAVER_BACKLOG_ACTOR, which a child agent would inherit and act
// under.

// genericAgent is the fallback name when no signal identifies the actor and the
// session is non-interactive.
const genericAgent = "agent"

// actorSource records which rule in the chain named the actor.
type actorSource string

const (
	sourceFlag     actorSource = "flag"     // --as
	sourceEnv      actorSource = "env"      // BEAVER_BACKLOG_ACTOR
	sourceAgent    actorSource = "agent"    // a detected agent harness
	sourceConfig   actorSource = "config"   // the saved user-level identity
	sourcePrompt   actorSource = "prompt"   // just established interactively and saved
	sourceFallback actorSource = "fallback" // the generic `agent`
)

// actor is a resolved identity paired with the rule that produced it.
type actor struct {
	name   string
	source actorSource
}

// resolveActor runs the precedence chain, given the command's --as value (empty
// when absent). It may prompt on stderr and read stdin for interactive human
// setup, and warns on stderr when it falls back to the generic agent. It errors
// only when interactive setup cannot complete.
func resolveActor(env Env, asFlag string) (actor, error) {
	// 1. --as — explicit and always decisive.
	if name := strings.TrimSpace(asFlag); name != "" {
		return actor{name, sourceFlag}, nil
	}
	// 2. BEAVER_BACKLOG_ACTOR — the programmatic override agents and CI set for themselves.
	if name := strings.TrimSpace(env.Getenv("BEAVER_BACKLOG_ACTOR")); name != "" {
		return actor{name, sourceEnv}, nil
	}
	// 3. Agent detection — set by the harness, not the human, so it carries no
	// inheritance footgun.
	if name, ok := knownAgent(env.Getenv); ok {
		return actor{name, sourceAgent}, nil
	}
	// 4. Interactive human — gated on an interactive session, because a
	// non-interactive run must never borrow the human's identity.
	if env.StdinIsTTY {
		cfg, err := userconfig.Load(env.UserConfigDir)
		if err != nil {
			return actor{}, err
		}
		if cfg.Actor != "" {
			return actor{cfg.Actor, sourceConfig}, nil
		}
		name, err := establishHumanIdentity(env)
		if err != nil {
			return actor{}, err
		}
		return actor{name, sourcePrompt}, nil
	}
	// 5. Nothing matched and no one to ask: proceed as a loud generic agent.
	warnGenericAgent(env)
	return actor{genericAgent, sourceFallback}, nil
}

// knownAgent maps the environment to a named agent harness: the AGENT convention
// first, then tool-specific markers. It backs both identity resolution and the
// output-format bias, so the registry lives in exactly one place.
func knownAgent(getenv func(string) string) (string, bool) {
	if getenv == nil {
		return "", false
	}
	if name := strings.TrimSpace(getenv("AGENT")); name != "" {
		return name, true
	}
	if getenv("CLAUDECODE") != "" {
		return "claude", true
	}
	return "", false
}

// establishHumanIdentity prompts for a name and saves it to user-level config so
// later interactive runs skip the prompt.
func establishHumanIdentity(env Env) (string, error) {
	name, err := promptForIdentity(env)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("no identity provided; set one with --as or BEAVER_BACKLOG_ACTOR")
	}
	if err := userconfig.Save(env.UserConfigDir, userconfig.Config{Actor: name}); err != nil {
		return "", fmt.Errorf("saving identity: %w", err)
	}
	return name, nil
}

// promptForIdentity asks the human for their name and reads the answer from
// stdin. The prompt goes to stderr so it never pollutes stdout, which a caller
// may be capturing.
func promptForIdentity(env Env) (string, error) {
	fmt.Fprint(env.Stderr, "Enter your Beaver Backlog identity (a name): ")
	return readReply(bufio.NewReader(env.Stdin))
}

// warnGenericAgent tells stderr that work is being attributed to the shared
// `agent` name, so an unconfigured agent is noticed rather than silently
// conflated.
func warnGenericAgent(env Env) {
	errf(env, "no actor identity resolved; proceeding as the generic %q. "+
		"Set BEAVER_BACKLOG_ACTOR or pass --as to name this actor.", genericAgent)
}

// readReply reads one line of interactive input, trimming the trailing newline.
// A bare EOF is an error; input without a trailing newline is still returned.
func readReply(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if line == "" && err != nil {
		return "", fmt.Errorf("reading identity: %w", err)
	}
	return line, nil
}

// cmdWhoami prints the actor Beaver Backlog resolves for the current environment. It
// performs the same resolution every attributing command does, including the
// interactive setup and the generic fallback.
func cmdWhoami(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "whoami")
	asFlag := fs.String("as", "", "resolve as this actor")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "whoami takes no arguments (did you mean --as %s?)", pos[0])
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	a, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	if format == output.JSON {
		if err := output.WriteJSON(env.Stdout, map[string]any{"actor": a.name, "source": string(a.source)}); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintln(env.Stdout, a.name)
	return exitOK
}
