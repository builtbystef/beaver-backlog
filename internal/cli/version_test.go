package cli_test

import (
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/cli"
)

// The worked example from the spec: an injected build names itself completely,
// in both formats.
func TestVersionReportsInjectedBuild(t *testing.T) {
	h := beavertest.New(t)
	h.Build = cli.Build{Version: "1.0.0", Commit: "abc1234", Date: "2026-08-27"}

	human := h.MustRun("version", "--format", "human").Stdout
	if want := "beaver 1.0.0 (commit abc1234, built 2026-08-27)\n"; human != want {
		t.Errorf("human version = %q, want %q", human, want)
	}

	got := h.DecodeJSON(h.MustRun("version", "--format", "json").Stdout)
	want := map[string]any{"version": "1.0.0", "commit": "abc1234", "built": "2026-08-27"}
	if len(got) != len(want) {
		t.Errorf("JSON version = %v, want exactly the keys of %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("JSON %s = %v, want %v", k, got[k], v)
		}
	}
}

// A plain `go build` injects nothing, and an unreleased binary must not look
// like a released one: it reports "dev" and claims nothing it does not know.
func TestVersionOfAnUninjectedBuildIsDev(t *testing.T) {
	h := beavertest.New(t)

	if human := h.MustRun("version", "--format", "human").Stdout; human != "beaver dev\n" {
		t.Errorf("human version = %q, want %q", human, "beaver dev\n")
	}

	got := h.DecodeJSON(h.MustRun("version", "--format", "json").Stdout)
	if got["version"] != "dev" || got["commit"] != "" || got["built"] != "" {
		t.Errorf("JSON version = %v, want version dev with empty commit and built", got)
	}
}

func TestVersionFormatResolution(t *testing.T) {
	h := beavertest.New(t)

	h.IsTTY = false // auto-detects JSON when piped
	if out := h.MustRun("version").Stdout; !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("piped version = %q, want JSON", out)
	}
	h.IsTTY = true // auto-detects human at a terminal
	if out := h.MustRun("version").Stdout; strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("terminal version = %q, want human", out)
	}
	if out := h.MustRun("version", "--format", "json").Stdout; !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("--format json at a terminal = %q, want JSON", out)
	}
	if r := h.Run("version", "--format", "xml"); r.Code != 2 {
		t.Errorf("invalid format exit = %d, want 2 (usage)", r.Code)
	}
}

// The command answers for the binary, not for a project, so it must not need a
// store: a bug reporter runs it wherever they are.
func TestVersionWorksOutsideAStore(t *testing.T) {
	h := beavertest.New(t) // no init

	r := h.Run("version")
	if r.Code != 0 {
		t.Errorf("version outside a store exit = %d, want 0\nstderr: %s", r.Code, r.Stderr)
	}
	if strings.Contains(r.Stderr, "init") {
		t.Errorf("version outside a store complained about the store:\n%s", r.Stderr)
	}
}

func TestVersionTakesNoArguments(t *testing.T) {
	h := beavertest.New(t)

	r := h.Run("version", "1.0.0")
	if r.Code != 2 {
		t.Errorf("version with an argument exit = %d, want 2 (usage)", r.Code)
	}
	if r.Stdout != "" {
		t.Errorf("expected no stdout on a usage error, got %q", r.Stdout)
	}
}
