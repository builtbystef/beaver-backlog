package main

import (
	"os"
	"path/filepath"

	"golang.org/x/term"

	"github.com/builtbystef/beaver-backlog/internal/cli"
)

func main() {
	// No core options: the real process wants the real clock and ID generator,
	// which the core defaults to.
	os.Exit(cli.Run(cli.Env{
		Args:          os.Args[1:],
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		WorkDir:       workDir(),
		Getenv:        os.Getenv,
		StdoutIsTTY:   term.IsTerminal(int(os.Stdout.Fd())),
		StdinIsTTY:    term.IsTerminal(int(os.Stdin.Fd())),
		UserConfigDir: userConfigDir(),
	}))
}

func workDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// userConfigDir is where the per-machine identity lives — Beaver Backlog's own
// subdirectory of the OS user-config directory (e.g. ~/.config/beaver), never
// the project. It returns "" when the OS location cannot be determined.
func userConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "beaver")
}
