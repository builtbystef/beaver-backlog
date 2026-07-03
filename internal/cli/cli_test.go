package cli_test

import (
	"strings"
	"testing"
	"time"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

// AC: `beaver init` creates `.beaver/issues/` and a committed project config with
// a format-version marker.
func TestInitCreatesStore(t *testing.T) {
	h := beavertest.New(t)
	r := h.MustRun("init")

	h.IssueFiles() // fails the test if .beaver/issues/ is missing
	if cfg := h.ReadFile("config.yml"); !strings.Contains(cfg, "format_version:") {
		t.Errorf("config is missing a format-version marker:\n%s", cfg)
	}
	if got := h.DecodeJSON(r.Stdout)["created"]; got != true {
		t.Errorf("first init reported created=%v, want true", got)
	}
}

// AC: re-running init is safe.
func TestInitIsIdempotent(t *testing.T) {
	h := beavertest.New(t)
	h.MustRun("init")
	original := h.ReadFile("config.yml")

	r := h.MustRun("init")
	if got := h.DecodeJSON(r.Stdout)["created"]; got != false {
		t.Errorf("second init reported created=%v, want false", got)
	}
	if h.ReadFile("config.yml") != original {
		t.Error("re-running init clobbered the existing config")
	}
}

// AC: `beaver create` writes `<id>-<slug>.md` with id, title, state, created,
// updated, and a body; the filename mirrors the id; the frontmatter id is
// authoritative.
func TestCreateWritesIssueFile(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Clock.Set(time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC))

	out := h.DecodeJSON(h.MustRun("create", "Login form rejects valid passwords").Stdout)

	id, _ := out["id"].(string)
	if id == "" {
		t.Fatal("create returned an empty id")
	}
	if out["title"] != "Login form rejects valid passwords" {
		t.Errorf("title = %v", out["title"])
	}
	if out["state"] != "todo" {
		t.Errorf("state = %v, want todo", out["state"])
	}
	if out["created"] != "2026-06-27T18:30:00Z" || out["updated"] != "2026-06-27T18:30:00Z" {
		t.Errorf("timestamps = %v / %v", out["created"], out["updated"])
	}
	if out["body"] != "" {
		t.Errorf("body = %v, want empty", out["body"])
	}

	files := h.IssueFiles()
	if len(files) != 1 {
		t.Fatalf("want exactly one issue file, got %v", files)
	}
	if want := id + "-login-form-rejects-valid-passwords.md"; files[0] != want {
		t.Errorf("filename = %q, want %q", files[0], want)
	}

	parsed, err := issue.Unmarshal([]byte(h.ReadFile("issues/" + files[0])))
	if err != nil {
		t.Fatalf("created file does not parse: %v", err)
	}
	if parsed.ID != id {
		t.Errorf("frontmatter id %q != reported id %q", parsed.ID, id)
	}
	if issue.IDFromFileName(files[0]) != parsed.ID {
		t.Errorf("filename id-part %q != authoritative frontmatter id %q", issue.IDFromFileName(files[0]), parsed.ID)
	}
	if parsed.State != issue.StateTodo {
		t.Errorf("parsed state = %q, want todo", parsed.State)
	}
}

// AC: IDs are short and random (not sequential).
func TestCreateIDsAreShortRandomDistinct(t *testing.T) {
	h := beavertest.New(t).Init()
	a := h.DecodeJSON(h.MustRun("create", "First issue").Stdout)["id"].(string)
	b := h.DecodeJSON(h.MustRun("create", "Second issue").Stdout)["id"].(string)

	if a == b {
		t.Errorf("expected distinct ids, both were %q", a)
	}
	for _, id := range []string{a, b} {
		if len(id) != 6 {
			t.Errorf("id %q is not short (len %d)", id, len(id))
		}
	}
	if got := h.IssueFiles(); len(got) != 2 {
		t.Errorf("want 2 issue files, got %v", got)
	}
}

// A generated id that collides with an existing issue must be regenerated. This
// also exercises the injectable ID generator.
func TestCreateRegeneratesOnIDCollision(t *testing.T) {
	h := beavertest.New(t).Init()
	ids := []string{"a1a1", "a1a1", "b2b2"} // 2nd create's first draw collides
	i := 0
	h.NewID = func() string {
		id := ids[i]
		i++
		return id
	}

	if got := h.DecodeJSON(h.MustRun("create", "one").Stdout)["id"]; got != "a1a1" {
		t.Errorf("first id = %v, want a1a1", got)
	}
	if got := h.DecodeJSON(h.MustRun("create", "two").Stdout)["id"]; got != "b2b2" {
		t.Errorf("second id = %v, want b2b2 (collision should regenerate)", got)
	}
}

// AC: output is human at a TTY, JSON when piped.
func TestShowAutoDetectsFormat(t *testing.T) {
	h := beavertest.New(t).Init()
	id := h.DecodeJSON(h.MustRun("create", "Some issue").Stdout)["id"].(string)

	h.IsTTY = false
	if got := h.DecodeJSON(h.MustRun("show", id).Stdout)["id"]; got != id {
		t.Errorf("piped show id = %v, want %v", got, id)
	}

	h.IsTTY = true
	human := h.MustRun("show", id).Stdout
	if strings.HasPrefix(strings.TrimSpace(human), "{") {
		t.Errorf("expected human output at a TTY, got JSON:\n%s", human)
	}
	if !strings.Contains(human, id) || !strings.Contains(human, "Some issue") {
		t.Errorf("human output missing fields:\n%s", human)
	}
}

// AC: `--format` overrides the auto-detection (both directions).
func TestShowFormatOverride(t *testing.T) {
	h := beavertest.New(t).Init()
	id := h.DecodeJSON(h.MustRun("create", "Some issue").Stdout)["id"].(string)

	h.IsTTY = true // would be human, but force json
	h.DecodeJSON(h.MustRun("show", id, "--format", "json").Stdout)

	h.IsTTY = false // would be json, but force human (flag before the ref)
	human := h.MustRun("show", "--format", "human", id).Stdout
	if strings.HasPrefix(strings.TrimSpace(human), "{") {
		t.Errorf("forced human output is still JSON:\n%s", human)
	}
}

// AC: exit codes distinguish success from not-found.
func TestShowNotFound(t *testing.T) {
	h := beavertest.New(t).Init()
	r := h.Run("show", "zzzz")

	if r.Code != 3 {
		t.Errorf("not-found exit code = %d, want 3", r.Code)
	}
	if r.Stdout != "" {
		t.Errorf("expected no stdout on not-found, got %q", r.Stdout)
	}
	if !strings.Contains(r.Stderr, "zzzz") {
		t.Errorf("error should name the missing ref:\n%s", r.Stderr)
	}
	// Distinct from a successful run.
	id := h.DecodeJSON(h.MustRun("create", "real").Stdout)["id"].(string)
	if ok := h.Run("show", id); ok.Code != 0 {
		t.Errorf("show of an existing issue exited %d, want 0", ok.Code)
	}
}

// AC: timestamps come from the injected clock (deterministic).
func TestClockIsInjected(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Clock.Set(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC))

	out := h.DecodeJSON(h.MustRun("create", "timed issue").Stdout)
	if out["created"] != "2030-01-02T03:04:05Z" || out["updated"] != "2030-01-02T03:04:05Z" {
		t.Errorf("clock not injected: created=%v updated=%v", out["created"], out["updated"])
	}
}

// AC: a missing store exits 3 (not-found), the code the CLI documents for "the
// store was not found".
func TestCommandsRequireStore(t *testing.T) {
	for _, args := range [][]string{{"create", "x"}, {"show", "x"}} {
		h := beavertest.New(t) // no init
		r := h.Run(args...)
		if r.Code != 3 {
			t.Errorf("%v without a store exit = %d, want 3 (not-found)", args, r.Code)
		}
		if !strings.Contains(r.Stderr, "init") {
			t.Errorf("%v error should suggest init:\n%s", args, r.Stderr)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	cases := [][]string{
		{"create"},                       // missing title
		{"create", "a", "b"},             // too many args
		{"show"},                         // missing ref
		{"frobnicate"},                   // unknown command
		{"show", "x", "--format", "xml"}, // invalid format
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}
