package cli_test

import (
	"maps"
	"strings"
	"testing"

	"beaver/internal/beavertest"
	"beaver/internal/userconfig"
	"beaver/internal/vcs"
)

// AC: whoami resolves correctly across every branch of the precedence chain —
// flag, env, agent-detect, saved user-config, and the generic fallback — and each
// case is driven deterministically by injected env vars, the interactivity signal,
// and (where relevant) a saved identity. Overlapping signals are set on purpose so
// each case proves the *precedence*, not just that one lone signal works.
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
			env:        map[string]string{"BUSY_BEAVER_ACTOR": "envguy", "CLAUDECODE": "1"},
			asFlag:     "flaguy",
			wantActor:  "flaguy",
			wantSource: "flag",
		},
		{
			name:       "env beats agent detection",
			env:        map[string]string{"BUSY_BEAVER_ACTOR": "ci-bot", "CLAUDECODE": "1"},
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

// AC: a human's Git name seeds user-config interactively, with confirmation. A bare
// Enter accepts the offered VCS identity; it is then saved, so the next interactive
// run resolves straight from config without prompting again.
func TestWhoamiInteractiveSeedsFromVCSWithConfirmation(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = true
	h.VCS = &vcs.Fake{Name: "Ada Lovelace", Found: true}
	h.StdinText = "\n" // Enter accepts the [Y/n] default

	r := h.MustRun("whoami", "--format", "json")
	out := h.DecodeJSON(r.Stdout)
	if out["actor"] != "Ada Lovelace" || out["source"] != "prompt" {
		t.Errorf("resolved = %v/%v, want Ada Lovelace/prompt", out["actor"], out["source"])
	}
	if !strings.Contains(r.Stderr, "Ada Lovelace") {
		t.Errorf("the confirmation prompt should show the VCS name:\n%s", r.Stderr)
	}
	if got := savedActor(t, h); got != "Ada Lovelace" {
		t.Errorf("confirmed identity was not saved: %q", got)
	}

	// A second interactive run now short-circuits to the saved config: it must not
	// prompt, so an empty stdin is fine.
	h.StdinText = ""
	out2 := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out2["actor"] != "Ada Lovelace" || out2["source"] != "config" {
		t.Errorf("second run = %v/%v, want Ada Lovelace/config", out2["actor"], out2["source"])
	}
}

// Declining the VCS seed falls through to a free-form name prompt, and what the
// human types becomes (and is saved as) the identity.
func TestWhoamiInteractiveDeclineVCSThenType(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = true
	h.VCS = &vcs.Fake{Name: "Git Name", Found: true}
	h.StdinText = "n\nStefan\n" // decline the seed, then type a name

	out := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out["actor"] != "Stefan" {
		t.Errorf("actor = %v, want Stefan (typed after declining the seed)", out["actor"])
	}
	if got := savedActor(t, h); got != "Stefan" {
		t.Errorf("typed identity was not saved: %q", got)
	}
}

// With no VCS adapter there is nothing to seed from, so resolution prompts directly
// for a name.
func TestWhoamiInteractiveNoVCSPromptsForName(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = true
	h.VCS = nil // no adapter configured
	h.StdinText = "Grace Hopper\n"

	out := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out["actor"] != "Grace Hopper" || out["source"] != "prompt" {
		t.Errorf("resolved = %v/%v, want Grace Hopper/prompt", out["actor"], out["source"])
	}
	if got := savedActor(t, h); got != "Grace Hopper" {
		t.Errorf("prompted identity was not saved: %q", got)
	}
}

// AC: the VCS name is never used non-interactively. An agent inheriting the human's
// Git config must not claim under the human's name — a non-interactive run resolves
// to the generic agent instead, and saves nothing.
func TestVCSNameNeverUsedNonInteractively(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = false // the key: not an interactive session
	h.VCS = &vcs.Fake{Name: "Human Git Name", Found: true}

	out := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out["actor"] == "Human Git Name" {
		t.Fatal("a non-interactive run adopted the human's VCS name; it must not")
	}
	if out["actor"] != "agent" || out["source"] != "fallback" {
		t.Errorf("resolved = %v/%v, want agent/fallback", out["actor"], out["source"])
	}
	if got := savedActor(t, h); got != "" {
		t.Errorf("a non-interactive run saved an identity (%q); it must not", got)
	}
}

// The saved human identity is likewise interactive-only: a non-interactive run does
// not borrow it, even when one exists.
func TestSavedIdentityIgnoredNonInteractively(t *testing.T) {
	h := beavertest.New(t).Init()
	saveActor(t, h, "saved-human")
	h.StdinIsTTY = false

	out := h.DecodeJSON(h.MustRun("whoami", "--format", "json").Stdout)
	if out["actor"] != "agent" || out["source"] != "fallback" {
		t.Errorf("resolved = %v/%v, want agent/fallback (saved identity is interactive-only)", out["actor"], out["source"])
	}
}

// AC: an unknown non-interactive caller becomes the generic agent — loudly, so it
// is noticed and can be named with BUSY_BEAVER_ACTOR or --as rather than silently
// conflated with other unidentified agents.
func TestGenericAgentFallbackIsLoud(t *testing.T) {
	h := beavertest.New(t).Init()
	r := h.MustRun("whoami", "--format", "json") // nothing set, non-interactive

	if out := h.DecodeJSON(r.Stdout); out["actor"] != "agent" {
		t.Errorf("actor = %v, want the generic agent", out["actor"])
	}
	if !strings.Contains(r.Stderr, "BUSY_BEAVER_ACTOR") || !strings.Contains(r.Stderr, "--as") {
		t.Errorf("the fallback should loudly explain how to name the actor:\n%s", r.Stderr)
	}
}

// Human output is the bare resolved name, not JSON — the demoable one-liner.
func TestWhoamiHumanPrintsName(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Env["BUSY_BEAVER_ACTOR"] = "stefan"
	h.IsTTY = true // interactive stdout → human format

	out := h.MustRun("whoami").Stdout
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected a bare name at a TTY, got JSON:\n%s", out)
	}
	if strings.TrimSpace(out) != "stefan" {
		t.Errorf("human whoami = %q, want the bare name 'stefan'", out)
	}
}

// An interactive session that cannot produce a name (nothing typed) fails cleanly
// and saves nothing, rather than inventing an identity.
func TestWhoamiInteractiveNoInputErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	h.StdinIsTTY = true
	h.VCS = nil
	h.StdinText = "" // interactive, but the human provides no name

	r := h.Run("whoami", "--format", "json")
	if r.Code != 1 {
		t.Errorf("exit = %d, want 1 (could not establish an identity)", r.Code)
	}
	if got := savedActor(t, h); got != "" {
		t.Errorf("nothing should be saved when setup fails, got %q", got)
	}
}

// AC: init seeds the runner's identity into user-level config. An interactive init
// runs the identity setup proactively; the name is saved (never into the committed
// project config), so later runs resolve it from config.
func TestInitSeedsIdentityInteractively(t *testing.T) {
	h := beavertest.New(t) // not yet initialized
	h.StdinIsTTY = true
	h.VCS = &vcs.Fake{Name: "Ada Lovelace", Found: true}
	h.StdinText = "\n" // accept the VCS seed

	out := h.DecodeJSON(h.MustRun("init").Stdout)
	if out["actor"] != "Ada Lovelace" {
		t.Errorf("init reported actor = %v, want Ada Lovelace", out["actor"])
	}
	if got := savedActor(t, h); got != "Ada Lovelace" {
		t.Errorf("init did not seed the identity: saved = %q", got)
	}
	// Never committed: the identity must not land in the project config.
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

// A non-interactive init never seeds: it must not prompt and must not silently
// adopt the human's VCS name (the same footgun rule as resolution).
func TestInitDoesNotSeedNonInteractively(t *testing.T) {
	h := beavertest.New(t)
	h.StdinIsTTY = false
	h.VCS = &vcs.Fake{Name: "Ada Lovelace", Found: true}

	out := h.DecodeJSON(h.MustRun("init").Stdout)
	if _, ok := out["actor"]; ok {
		t.Errorf("non-interactive init reported an actor (%v); it must not seed", out["actor"])
	}
	if got := savedActor(t, h); got != "" {
		t.Errorf("non-interactive init seeded an identity (%q); it must not", got)
	}
}

// init's seeding is idempotent: with an identity already saved, it neither prompts
// nor overwrites — so a re-init after setup is safe.
func TestInitIdentitySeedingIsIdempotent(t *testing.T) {
	h := beavertest.New(t)
	saveActor(t, h, "existing")
	h.StdinIsTTY = true
	h.VCS = &vcs.Fake{Name: "Different Name", Found: true}
	h.StdinText = "" // if init prompted, an empty read would error; it must not prompt

	out := h.DecodeJSON(h.MustRun("init").Stdout)
	if _, ok := out["actor"]; ok {
		t.Errorf("init re-seeded an already-set identity (%v)", out["actor"])
	}
	if got := savedActor(t, h); got != "existing" {
		t.Errorf("init changed the saved identity to %q, want unchanged 'existing'", got)
	}
}

// whoami misuse is a usage error (exit 2): stray positional arguments, or an
// invalid format.
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

// saveActor writes a saved user-config identity, standing in for a prior
// interactive setup so tests can exercise the config branch directly.
func saveActor(t *testing.T, h *beavertest.Harness, name string) {
	t.Helper()
	if err := userconfig.Save(h.UserConfigDir, userconfig.Config{Actor: name}); err != nil {
		t.Fatalf("save user config: %v", err)
	}
}

// savedActor reads back the saved user-config identity ("" when none), to assert
// what a run persisted.
func savedActor(t *testing.T, h *beavertest.Harness) string {
	t.Helper()
	cfg, err := userconfig.Load(h.UserConfigDir)
	if err != nil {
		t.Fatalf("load user config: %v", err)
	}
	return cfg.Actor
}
