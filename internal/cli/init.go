package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/output"
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

	root, created, err := core.Init(env.WorkDir)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}

	svc, err := core.Open(env.WorkDir)
	if err != nil {
		// The store was made a moment ago; if it will not open, say so.
		errf(env, "store initialized, but reading it back failed: %v", err)
	}

	// One reader for both prompts: a buffered reader takes in more than the line
	// it hands back, so a second one would find the input already gone.
	in := bufio.NewReader(env.Stdin)

	// Seed the project's name and then the runner's identity, so the common solo
	// case is "one command and you're ready". Both happen only interactively and
	// only where nothing is set yet; a non-interactive init prompts for neither.
	var named string
	if svc != nil {
		named = seedProjectName(env, svc, in)
	}
	seeded := seedIdentity(env, in)

	if format == output.JSON {
		result := map[string]any{"store_path": root, "created": created}
		if svc != nil {
			result["project"] = svc.ProjectName()
		}
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
	if named != "" {
		fmt.Fprintf(env.Stdout, "Project named %q (saved to the project config, committed with the store).\n", named)
	}
	if seeded != "" {
		fmt.Fprintf(env.Stdout, "Identity set to %q (saved to %s, never committed).\n", seeded, userconfig.Path(env.UserConfigDir))
	}
	return exitOK
}

// seedProjectName offers to name the project when init can: an interactive
// session over a store whose config names none yet. It returns the name it
// wrote, "" when it wrote none, and never fails init.
func seedProjectName(env Env, svc *core.Service, in *bufio.Reader) string {
	if !env.StdinIsTTY {
		return "" // never name a project non-interactively
	}
	configured, err := svc.ConfiguredProjectName()
	if err != nil || configured != "" {
		return "" // already named, or an unreadable config: leave it as-is
	}
	name, err := promptForProjectName(env, svc.ProjectName(), in)
	if err != nil {
		errf(env, "project name not set: %v", err)
		return ""
	}
	if name == "" {
		return "" // the offered default is what happens anyway; writing it out is noise
	}
	if err := svc.SetProjectName(name); err != nil {
		errf(env, "project name not set: %v", err)
		return ""
	}
	return name
}

// promptForProjectName asks what the project is called, offering the name it
// already has as the default. The prompt goes to stderr so it never pollutes
// stdout. An empty answer, a bare Enter or no answer at all, comes back as "":
// the cue to write nothing.
func promptForProjectName(env Env, fallback string, in *bufio.Reader) (string, error) {
	fmt.Fprintf(env.Stderr, "Project name [%s]: ", fallback)
	line, err := in.ReadString('\n')
	if name := strings.TrimSpace(line); name != "" {
		return name, nil
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return "", nil
}

// seedIdentity establishes the runner's saved identity when init can, meaning
// in an interactive session with none saved yet, and returns the name it saved,
// or "" when it does nothing. It never fails init: a declined or unreadable prompt
// only warns and leaves the store initialized.
func seedIdentity(env Env, in *bufio.Reader) string {
	if !env.StdinIsTTY {
		return "" // never seed non-interactively
	}
	cfg, err := userconfig.Load(env.UserConfigDir)
	if err != nil || cfg.Actor != "" {
		return "" // already set, or unreadable: leave it as-is
	}
	name, err := establishHumanIdentity(env, in)
	if err != nil {
		errf(env, "identity not set: %v", err)
		return ""
	}
	return name
}
