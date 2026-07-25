package cli

import (
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/output"
	"github.com/builtbystef/beaver-backlog/internal/store"
	"github.com/builtbystef/beaver-backlog/internal/userconfig"
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

	// Seed the runner's identity so the common solo case is "one command and
	// you're ready". This only happens interactively and only when nothing is
	// saved yet — a non-interactive init (agent or CI) never prompts.
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
		fmt.Fprintf(env.Stdout, "Initialized empty Beaver Backlog store in %s\n", root)
	} else {
		fmt.Fprintf(env.Stdout, "Reinitialized existing Beaver Backlog store in %s\n", root)
	}
	if seeded != "" {
		fmt.Fprintf(env.Stdout, "Identity set to %q (saved to %s, never committed).\n", seeded, userconfig.Path(env.UserConfigDir))
	}
	return exitOK
}

// seedIdentity establishes the runner's saved identity when init can — an
// interactive session with none saved yet — and returns the name it saved, or
// "" when it does nothing. It never fails init: a declined or unreadable prompt
// only warns and leaves the store initialized.
func seedIdentity(env Env) string {
	if !env.StdinIsTTY {
		return "" // never seed non-interactively
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
