package cli_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
)

// Init offers the directory's name as the name to keep, and writes whatever the
// human types instead into the committed config.
func TestInitOffersTheDirectoryNameAndWritesWhatIsAccepted(t *testing.T) {
	h := beavertest.New(t)
	h.StdinIsTTY = true
	h.StdinText = "Apollo Guidance\nAda Lovelace\n" // the project is asked about before the person

	r := h.MustRun("init")

	if !strings.Contains(r.Stderr, filepath.Base(h.Dir)) {
		t.Errorf("the prompt does not offer the directory's name as the default:\n%s", r.Stderr)
	}
	if out := h.DecodeJSON(r.Stdout); out["project"] != "Apollo Guidance" {
		t.Errorf("init reported project = %v, want the name that was typed", out["project"])
	}
	if cfg := h.ReadFile("config.yml"); !strings.Contains(cfg, "Apollo Guidance") {
		t.Errorf("the accepted name did not reach the committed config:\n%s", cfg)
	}
	// The name is the project's, the identity is the person's; they live apart.
	if got := savedActor(t, h); got != "Ada Lovelace" {
		t.Errorf("saved identity = %q, want the name given at the second prompt", got)
	}
}

// Saying nothing leaves the store named after its directory and writes no key:
// the default is what happens anyway.
func TestInitAcceptingNoProjectNameLeavesTheKeyAbsent(t *testing.T) {
	h := beavertest.New(t)
	h.StdinIsTTY = true
	h.StdinText = "\nAda Lovelace\n"

	out := h.DecodeJSON(h.MustRun("init").Stdout)

	if out["project"] != filepath.Base(h.Dir) {
		t.Errorf("init reported project = %v, want the directory's name %q", out["project"], filepath.Base(h.Dir))
	}
	if cfg := h.ReadFile("config.yml"); strings.Contains(cfg, "name:") {
		t.Errorf("an empty answer still wrote a name into the config:\n%s", cfg)
	}
}

// An agent or a CI run is never asked and never has a name written for it.
func TestInitDoesNotNameTheProjectNonInteractively(t *testing.T) {
	h := beavertest.New(t)
	h.StdinIsTTY = false
	h.StdinText = "Apollo Guidance\n" // available, but a non-interactive init must not read it

	r := h.MustRun("init")

	if out := h.DecodeJSON(r.Stdout); out["project"] != filepath.Base(h.Dir) {
		t.Errorf("init reported project = %v, want the directory's name %q", out["project"], filepath.Base(h.Dir))
	}
	if cfg := h.ReadFile("config.yml"); strings.Contains(cfg, "name:") {
		t.Errorf("a non-interactive init named the project:\n%s", cfg)
	}
	if strings.Contains(r.Stderr, "Project name") {
		t.Errorf("a non-interactive init prompted anyway:\n%s", r.Stderr)
	}
}

// Re-running init on a store that already says what it is called leaves that
// name alone, and does not ask again.
func TestInitLeavesAnAlreadyNamedProjectAlone(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("config.yml", "format_version: 1\nname: Apollo Guidance\n")
	original := h.ReadFile("config.yml")
	saveActor(t, h, "existing") // so the identity prompt stays quiet too
	h.StdinIsTTY = true
	h.StdinText = "Gemini\n"

	r := h.MustRun("init")

	if out := h.DecodeJSON(r.Stdout); out["project"] != "Apollo Guidance" {
		t.Errorf("init reported project = %v, want the configured name", out["project"])
	}
	if got := h.ReadFile("config.yml"); got != original {
		t.Errorf("re-running init rewrote the named config:\n%s", got)
	}
}
