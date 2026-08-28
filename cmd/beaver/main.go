package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"golang.org/x/term"

	"github.com/builtbystef/beaver-backlog/internal/cli"
)

// Build metadata, injected at link time by the release build:
//
//	go build -ldflags "-X main.version=1.0.0 -X main.commit=abc1234 -X main.date=2026-08-27" ./cmd/beaver
//
// A plain `go build` leaves them empty, which the version command reports as a
// dev build. They are the only place the binary learns what it is.
var (
	version string
	commit  string
	date    string
)

func main() {
	// Interrupt reaches the engine as a cancelled context, so a foreground
	// command (serve) can shut itself down cleanly instead of being killed
	// mid-response. A second signal is left to the runtime's default handler.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// No core options: the real process wants the real clock and ID generator,
	// which the core defaults to.
	os.Exit(cli.Run(cli.Env{
		Args:          os.Args[1:],
		Ctx:           ctx,
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		WorkDir:       workDir(),
		Getenv:        os.Getenv,
		StdoutIsTTY:   term.IsTerminal(int(os.Stdout.Fd())),
		StdinIsTTY:    term.IsTerminal(int(os.Stdin.Fd())),
		UserConfigDir: userConfigDir(),
		Build:         cli.Build{Version: version, Commit: commit, Date: date},
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
