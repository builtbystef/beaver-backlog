package main

import (
	"os"

	"golang.org/x/term"

	"beaver/internal/cli"
	"beaver/internal/clock"
	"beaver/internal/issue"
)

func main() {
	os.Exit(cli.Run(cli.Env{
		Args:        os.Args[1:],
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		WorkDir:     workDir(),
		Getenv:      os.Getenv,
		Clock:       clock.System(),
		NewID:       issue.NewID,
		StdoutIsTTY: term.IsTerminal(int(os.Stdout.Fd())),
	}))
}

func workDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
