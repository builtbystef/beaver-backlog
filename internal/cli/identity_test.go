package cli_test

import (
	"maps"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/userconfig"
)

// Overlapping signals are set on purpose so each case proves the precedence, not
// just that one lone signal works.
func TestWhoamiPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		env        map[string]string
		asFlag     string
		stdinTTY   bool
		preSave    string // a saved user-config identity, if any
		wantActor  string
		wantSource string
	}{
		{
			name:       "flag beats env and agent",
			env:        map[string]string{"BEAVER_BACKLOG_ACTOR": "envguy", "CLAUDECODE": "1"},
			asFlag:     "flaguy",
			wantActor:  "flaguy",
			wantSource: "flag",
		},
		{
			name:       "env beats agent detection",
			env:        map[string]string{"BEAVER_BACKLOG_ACTOR": "ci-bot", "CLAUDECODE": "1"},
			wantActor:  "ci-bot",
			wantSource: "env",
		},
		{
			name:       "AGENT names the agent, ahead of the human steps",
			env:        map[string]string{"AGENT": "goose", "CLAUDECODE": "1"},
			wantActor:  "goose",
			wantSource: "agent",
		},
		{
			name:       "CLAUDECODE marker maps to claude",
			env:        map[string]string{"CLAUDECODE": "1"},
			wantActor:  "claude",
			wantSource: "agent",
		},
		{
			name:       "interactive human reads the saved identity",
			stdinTTY:   true,
			preSave:    "ada",
			wantActor:  "ada",
			wantSource: "config",
		},
		{
			name:       "non-interactive with no signal falls to the generic agent",
			wantActor:  "agent",
			wantSource: "fallback",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := beavertest.New(t).Init() // init runs non-interactively here; it seeds nothing
			maps.Copy(h.Env, c.env)
			h.StdinIsTTY = c.stdinTTY
			if c.preSave != "" {
				saveActor(t, h, c.preSave)
			}

			args := []string{"whoami", "--format", "json"}
			if c.asFlag != "" {
				args = append(args, "--as", c.asFlag)
			}
			out := h.DecodeJSON(h.MustRun(args...).Stdout)
			if out["actor"] != c.wantActor {
				t.Errorf("actor = %v, want %v", out["actor"], c.wantActor)
			}
			if out["source"] != c.wantSource {
				t.Errorf("source = %v, want %v", out["source"], c.wantSource)
			}
		})
	}
}

// The interactive prompt is free-form: whatever the human types becomes the
// identity, and it is saved so later runs read it straight from config.
func TestWhoamiInteractivePromptsForNameAndSaves(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = true
	h.StdinText = "Grace Hopper\n"

	r := h.MustRun("whoami", "--format", "json")
	out := h.DecodeJSON(r.Stdout)
	if out["actor"] != "Grace Hopper" || out["source"] != "prompt" {
		t.Errorf("resolved = %v/%v, want Grace Hopper/prompt", out["actor"], out["source"])
	}
	if !strings.Contains(r.Stderr, "identity") {
		t.Errorf("the prompt should ask for an identity on stderr:\n%s", r.Stderr)
	}
	if got := savedActor(t, h); got != "Grace Hopper" {
		t.Errorf("prompted identity was not saved: %q", got)
	}

	// The second run must not prompt, so an empty stdin is fine.
	h.StdinText = ""
	out2 := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out2["actor"] != "Grace Hopper" || out2["source"] != "config" {
		t.Errorf("second run = %v/%v, want Grace Hopper/config", out2["actor"], out2["source"])
	}
}

// The saved human identity is likewise interactive-only: an agent must not borrow it.
func TestSavedIdentityIgnoredNonInteractively(t *testing.T) {
	h := beavertest.New(t).Init()
	saveActor(t, h, "saved-human")
	h.StdinIsTTY = false

	out := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out["actor"] != "agent" || out["source"] != "fallback" {
		t.Errorf("resolved = %v/%v, want agent/fallback (saved identity is interactive-only)", out["actor"], out["source"])
	}
}

// The fallback is loud so an unidentified caller gets named with BEAVER_BACKLOG_ACTOR
// or --as rather than silently conflated with other agents.
func TestGenericAgentFallbackIsLoud(t *testing.T) {
	h := beavertest.New(t).Init()
	r := h.MustRun("whoami", "--format", "json") // nothing set, non-interactive

	if out := h.DecodeJSON(r.Stdout); out["actor"] != "agent" {
		t.Errorf("actor = %v, want the generic agent", out["actor"])
	}
	if !strings.Contains(r.Stderr, "BEAVER_BACKLOG_ACTOR") || !strings.Contains(r.Stderr, "--as") {
		t.Errorf("the fallback should loudly explain how to name the actor:\n%s", r.Stderr)
	}
}

func TestWhoamiHumanPrintsName(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Env["BEAVER_BACKLOG_ACTOR"] = "stefan"
	h.IsTTY = true // interactive stdout → human format

	out := h.MustRun("whoami").Stdout
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected a bare name at a TTY, got JSON:\n%s", out)
	}
	if strings.TrimSpace(out) != "stefan" {
		t.Errorf("human whoami = %q, want the bare name 'stefan'", out)
	}
}

// A session that cannot produce a name fails and saves nothing, rather than
// inventing an identity.
func TestWhoamiInteractiveNoInputErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = true
	h.StdinText = "" // interactive, but the human provides no name

	r := h.Run("whoami", "--format", "json")
	if r.Code != 1 {
		t.Errorf("exit = %d, want 1 (could not establish an identity)", r.Code)
	}
	if got := savedActor(t, h); got != "" {
		t.Errorf("nothing should be saved when setup fails, got %q", got)
	}
}

func TestInitSeedsIdentityInteractively(t *testing.T) {
	h := beavertest.New(t) // not yet initialized
	h.StdinIsTTY = true
	h.StdinText = "Ada Lovelace\n"

	out := h.DecodeJSON(h.MustRun("init").Stdout)
	if out["actor"] != "Ada Lovelace" {
		t.Errorf("init reported actor = %v, want Ada Lovelace", out["actor"])
	}
	if got := savedActor(t, h); got != "Ada Lovelace" {
		t.Errorf("init did not seed the identity: saved = %q", got)
	}
	// The identity is personal: it must never land in the committed project config.
	if cfg := h.ReadFile("config.yml"); strings.Contains(cfg, "Ada") {
		t.Errorf("identity leaked into the committed project config:\n%s", cfg)
	}
	// Later resolution reads it straight from config, without prompting.
	h.StdinText = ""
	who := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if who["actor"] != "Ada Lovelace" || who["source"] != "config" {
		t.Errorf("post-init whoami = %v/%v, want Ada Lovelace/config", who["actor"], who["source"])
	}
}

func TestInitDoesNotSeedNonInteractively(t *testing.T) {
	h := beavertest.New(t)
	h.StdinIsTTY = false
	h.StdinText = "Ada Lovelace\n" // available, but a non-interactive init must not read it

	out := h.DecodeJSON(h.MustRun("init").Stdout)
	if _, ok := out["actor"]; ok {
		t.Errorf("non-interactive init reported an actor (%v); it must not seed", out["actor"])
	}
	if got := savedActor(t, h); got != "" {
		t.Errorf("non-interactive init seeded an identity (%q); it must not", got)
	}
}

func TestInitIdentitySeedingIsIdempotent(t *testing.T) {
	h := beavertest.New(t)
	saveActor(t, h, "existing")
	h.StdinIsTTY = true
	h.StdinText = "" // if init prompted, an empty read would error; it must not prompt

	out := h.DecodeJSON(h.MustRun("init").Stdout)
	if _, ok := out["actor"]; ok {
		t.Errorf("init re-seeded an already-set identity (%v)", out["actor"])
	}
	if got := savedActor(t, h); got != "existing" {
		t.Errorf("init changed the saved identity to %q, want unchanged 'existing'", got)
	}
}

func TestWhoamiUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	for _, args := range [][]string{
		{"whoami", "extra"},
		{"whoami", "--format", "xml"},
	} {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

// --- helpers ---

// saveActor writes a user-config identity, standing in for a prior interactive setup.
func saveActor(t *testing.T, h *beavertest.Harness, name string) {
	t.Helper()
	if err := userconfig.Save(h.UserConfigDir, userconfig.Config{Actor: name}); err != nil {
		t.Fatalf("save user config: %v", err)
	}
}

// savedActor reads back the saved user-config identity ("" when none).
func savedActor(t *testing.T, h *beavertest.Harness) string {
	t.Helper()
	cfg, err := userconfig.Load(h.UserConfigDir)
	if err != nil {
		t.Fatalf("load user config: %v", err)
	}
	return cfg.Actor
}
