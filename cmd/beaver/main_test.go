package main

import (
	"os"
	"slices"
	"testing"
)

func TestTokenizeEditorSpec(t *testing.T) {
	cases := []struct {
		name string
		spec string
		want []string
	}{
		{"bare command", "vim", []string{"vim"}},
		{"command with flag", "code --wait", []string{"code", "--wait"}},
		{"collapses runs of spaces", "  code   --wait  ", []string{"code", "--wait"}},
		{"tab separated", "code\t--wait", []string{"code", "--wait"}},
		{"double-quoted spaced path", `"C:\Program Files\Vim\vim.exe" -f`, []string{`C:\Program Files\Vim\vim.exe`, "-f"}},
		{"single-quoted spaced path", `'/Applications/My Editor' --wait`, []string{"/Applications/My Editor", "--wait"}},
		{"quote joins adjacent text", `emacs-"nox"`, []string{"emacs-nox"}},
		{"unbalanced quote runs to end", `"unterminated arg`, []string{"unterminated arg"}},
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tokenizeEditorSpec(c.spec)
			if !slices.Equal(got, c.want) {
				t.Errorf("tokenizeEditorSpec(%q) = %q, want %q", c.spec, got, c.want)
			}
		})
	}
}

// A multi-word spec whose command is not on PATH falls to tokenizing, so its
// arguments are split off from the command.
func TestParseEditorSpecTokenizesUnknownCommand(t *testing.T) {
	name, args := parseEditorSpec("beaver-no-such-editor-xyz --wait --line=1")
	if name != "beaver-no-such-editor-xyz" {
		t.Errorf("name = %q, want beaver-no-such-editor-xyz", name)
	}
	if !slices.Equal(args, []string{"--wait", "--line=1"}) {
		t.Errorf("args = %q, want [--wait --line=1]", args)
	}
}

// A spec that is itself an executable path is taken whole, never split into
// arguments — the guard against shattering an unquoted spaced Windows path. The
// running test binary is a known-executable path, so it hits the LookPath-first
// branch.
func TestParseEditorSpecKeepsExecutablePathWhole(t *testing.T) {
	name, args := parseEditorSpec(os.Args[0])
	if name != os.Args[0] || len(args) != 0 {
		t.Errorf("parseEditorSpec(%q) = (%q, %q), want the path whole with no args", os.Args[0], name, args)
	}
}

// An empty or whitespace-only spec names no editor.
func TestParseEditorSpecEmpty(t *testing.T) {
	for _, spec := range []string{"", "   ", "\t"} {
		if name, _ := parseEditorSpec(spec); name != "" {
			t.Errorf("parseEditorSpec(%q) name = %q, want empty", spec, name)
		}
	}
}
