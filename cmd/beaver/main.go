package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"beaver/internal/cli"
	"beaver/internal/clock"
	"beaver/internal/issue"
	"beaver/internal/vcs"
)

func main() {
	wd := workDir()
	os.Exit(cli.Run(cli.Env{
		Args:          os.Args[1:],
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		WorkDir:       wd,
		Getenv:        os.Getenv,
		Clock:         clock.System(),
		NewID:         issue.NewID,
		Edit:          editor(os.Getenv),
		StdoutIsTTY:   term.IsTerminal(int(os.Stdout.Fd())),
		StdinIsTTY:    term.IsTerminal(int(os.Stdin.Fd())),
		VCS:           vcs.Git{Dir: wd}, // reference adapter: reads the identity seed from git
		UserConfigDir: userConfigDir(),
	}))
}

// editor returns the function the CLI uses to open a file in the user's editor for
// edit and interactive create, or nil when neither $VISUAL nor $EDITOR names one —
// the "no editor" state the CLI's editorGate refuses cleanly, up front, before any
// file is written (rather than an "editor failed" error after the fact). The
// setting is resolved once at startup and split into a command and arguments, so a
// multi-word value like "code --wait" works; the command runs with the file
// appended and its standard streams wired to the real terminal, blocking until the
// editor exits.
func editor(getenv func(string) string) func(string) error {
	spec := strings.TrimSpace(getenv("VISUAL"))
	if spec == "" {
		spec = strings.TrimSpace(getenv("EDITOR"))
	}
	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return nil
	}
	return func(path string) error {
		cmd := exec.Command(fields[0], append(fields[1:], path)...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}
}

func workDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// userConfigDir is where the per-machine identity lives — Busy Beaver's own
// subdirectory of the OS user-config directory (e.g. ~/.config/beaver), never the
// project (ADR 0008). An empty string when the OS location cannot be determined;
// resolution then falls back to a prompt and reports if it cannot save.
func userConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "beaver")
}
