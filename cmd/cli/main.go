package main

import (
	"os"
	"path/filepath"

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
		StdoutIsTTY:   term.IsTerminal(int(os.Stdout.Fd())),
		StdinIsTTY:    term.IsTerminal(int(os.Stdin.Fd())),
		VCS:           vcs.Git{Dir: wd}, // reference adapter: reads the identity seed from git
		UserConfigDir: userConfigDir(),
	}))
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
