package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/term"

	"github.com/builtbystef/beaver-backlog/internal/cli"
	"github.com/builtbystef/beaver-backlog/internal/clock"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/vcs"
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

// editor returns the function the CLI uses to open a file in the user's editor,
// or nil when neither $VISUAL nor $EDITOR names one, so the CLI can refuse up
// front rather than hang. The editor runs with its streams on the real
// terminal, blocking until it exits.
func editor(getenv func(string) string) func(string) error {
	spec := strings.TrimSpace(getenv("VISUAL"))
	if spec == "" {
		spec = strings.TrimSpace(getenv("EDITOR"))
	}
	name, args := parseEditorSpec(spec)
	if name == "" {
		return nil
	}
	return func(path string) error {
		argv := append(append([]string{}, args...), path)
		cmd := exec.Command(name, argv...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}
}

// parseEditorSpec splits a $VISUAL/$EDITOR value into the editor command and its
// leading arguments. A spec that is itself an executable path is taken whole, so
// an unquoted path with spaces (common on Windows) is not split apart; anything
// else is tokenized on whitespace with single and double quotes honored. An
// empty spec yields an empty name — the caller's "no editor" signal.
func parseEditorSpec(spec string) (name string, args []string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", nil
	}
	if _, err := exec.LookPath(spec); err == nil {
		return spec, nil
	}
	tokens := tokenizeEditorSpec(spec)
	if len(tokens) == 0 {
		return "", nil
	}
	return tokens[0], tokens[1:]
}

// tokenizeEditorSpec splits spec into words on unquoted whitespace, honoring
// single and double quotes so a quoted path with spaces stays one token. Quotes
// group without appearing in the token; an unbalanced quote runs to the end.
func tokenizeEditorSpec(spec string) []string {
	var tokens []string
	var cur strings.Builder
	inToken := false
	var quote rune
	for _, r := range spec {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			inToken = true
		case unicode.IsSpace(r):
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens
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
