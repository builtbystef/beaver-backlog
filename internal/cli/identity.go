package cli

import (
	"bufio"
	"fmt"
	"strings"

	"beaver/internal/output"
	"beaver/internal/userconfig"
)

// Identity resolution answers "who is doing this?" for every command that
// attributes work, resolving the current actor through one precedence chain so
// both humans and agents are named correctly with no configuration in the common
// case (ADR 0010, refining ADR 0008):
//
//  1. --as <actor>         explicit, always wins
//  2. BUSY_BEAVER_ACTOR    explicit override (agents, CI); never a human's stored identity
//  3. agent detection      AGENT, else a known marker (CLAUDECODE → claude, …)
//  4. interactive human    the saved user-config identity; if unset, seed from the
//                          VCS and confirm, or prompt, then save (ADR 0008)
//  5. non-interactive      a loud generic `agent` when nothing above matched
//
// Two rules keep it footgun-proof: the human's stored/VCS identity (step 4) is
// consulted *only* in an interactive session — a non-interactive run never borrows
// it — and a human's identity is never placed in BUSY_BEAVER_ACTOR, which a child agent
// would inherit and act under.

// genericAgent is the loud fallback name when no signal identifies the actor and
// the session is non-interactive (step 5).
const genericAgent = "agent"

// actorSource records which rule in the chain named the actor, so whoami can show
// the resolution and commands can be loud about the generic fallback.
type actorSource string

const (
	sourceFlag     actorSource = "flag"     // --as
	sourceEnv      actorSource = "env"      // BUSY_BEAVER_ACTOR
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

// resolveActor runs the precedence chain for the current command, given the
// command's --as value (empty when absent). It may prompt on stderr and read from
// stdin when it reaches interactive human setup (step 4), and it is loud on stderr
// when it falls to the generic agent (step 5). An error is returned only when
// interactive setup cannot complete (e.g. no name given, or the identity cannot be
// saved).
func resolveActor(env Env, asFlag string) (actor, error) {
	// 1. --as — explicit and always decisive.
	if name := strings.TrimSpace(asFlag); name != "" {
		return actor{name, sourceFlag}, nil
	}
	// 2. BUSY_BEAVER_ACTOR — the programmatic override agents and CI set for themselves.
	if name := strings.TrimSpace(env.Getenv("BUSY_BEAVER_ACTOR")); name != "" {
		return actor{name, sourceEnv}, nil
	}
	// 3. Agent detection — a signal the harness sets, free of the human-inheritance
	// footgun that git config and human-set env vars carry.
	if name, ok := knownAgent(env.Getenv); ok {
		return actor{name, sourceAgent}, nil
	}
	// 4. Interactive human — only here do we consult the saved identity or the VCS
	// seed. A non-interactive run must never borrow the human's identity, so this
	// whole branch is gated on an interactive session.
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

// knownAgent maps the environment to a named agent harness. The AGENT convention
// (AGENT=goose|amp|codex) is honored first; then a small, community-extensible set
// of tool-specific markers. These are set by the agent, not the human, so unlike
// git config they carry no inheritance footgun. It backs both step 3 of identity
// resolution and the output-format bias, so the registry lives in exactly one
// place (ADR 0010, ADR 0013).
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

// establishHumanIdentity runs step 4's interactive acquisition: it offers the VCS
// identity for confirmation, or prompts for a name, then saves the result to
// user-level config so later interactive runs skip straight to it. The session is
// already known to be interactive and to have no saved identity.
func establishHumanIdentity(env Env) (string, error) {
	name, err := promptForIdentity(env)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("no identity provided; set one with --as or BUSY_BEAVER_ACTOR")
	}
	if err := userconfig.Save(env.UserConfigDir, userconfig.Config{Actor: name}); err != nil {
		return "", fmt.Errorf("saving identity: %w", err)
	}
	return name, nil
}

// promptForIdentity asks the human for their name on stderr and reads the answer
// from stdin. When the VCS offers an identity it is presented as a confirmable
// default (Enter or y accepts it); declining, or having no VCS seed at all, falls
// to a free-form name prompt. Prompts go to stderr so they never pollute stdout,
// which a caller may be capturing.
func promptForIdentity(env Env) (string, error) {
	r := bufio.NewReader(env.Stdin)
	if seed, ok := vcsIdentity(env); ok {
		fmt.Fprintf(env.Stderr, "Use %q as your Busy Beaver identity? [Y/n] ", seed)
		line, err := readReply(r)
		if err != nil {
			return "", err
		}
		if isYes(line) {
			return seed, nil
		}
	}
	fmt.Fprint(env.Stderr, "Enter your Busy Beaver identity (a name): ")
	return readReply(r)
}

// vcsIdentity reads the configured VCS identity through the System, to seed the
// interactive confirmation. A nil System (no adapter), a not-found result, or any
// error all mean "no seed" — resolution then prompts for a name instead. Reading
// the VCS identity is only ever a seed for confirmation, never an agent identity
// (ADR 0010).
func vcsIdentity(env Env) (string, bool) {
	if env.VCS == nil {
		return "", false
	}
	name, found, err := env.VCS.Identity()
	if err != nil || !found {
		return "", false
	}
	name = strings.TrimSpace(name)
	return name, name != ""
}

// warnGenericAgent is the "loud" in step 5's loud generic agent: it tells stderr
// that work is being attributed to the shared `agent` name and how to distinguish
// actors, so an unconfigured agent is noticed rather than silently conflated.
func warnGenericAgent(env Env) {
	errf(env, "no actor identity resolved; proceeding as the generic %q. "+
		"Set BUSY_BEAVER_ACTOR or pass --as to name this actor.", genericAgent)
}

// readReply reads one line of interactive input, trimming the trailing newline. A
// bare EOF (the input closed with nothing typed) surfaces as an error; input that
// arrives without a trailing newline is still returned.
func readReply(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if line == "" && err != nil {
		return "", fmt.Errorf("reading identity: %w", err)
	}
	return line, nil
}

// isYes reports whether a confirmation reply accepts the offered default. An empty
// reply (a bare Enter) accepts it, since the prompt shows an uppercase default.
func isYes(reply string) bool {
	switch strings.ToLower(strings.TrimSpace(reply)) {
	case "", "y", "yes":
		return true
	default:
		return false
	}
}

// cmdWhoami prints the actor Busy Beaver resolves for the current environment,
// making the whole precedence chain demoable and testable. It performs the same
// resolution every attributing command will, including the interactive setup and
// the loud generic fallback.
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
