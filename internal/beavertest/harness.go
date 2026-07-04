// Package beavertest is the in-process test harness for the Busy Beaver CLI. It runs
// real commands against a temporary store with an injected clock, environment,
// and TTY signal, then exposes the exit code, captured stdio, and the resulting
// files for assertions.
package beavertest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"beaver/internal/cli"
	"beaver/internal/issue"
	"beaver/internal/vcs"
)

// DefaultNow is the instant the harness clock starts at, unless a test changes
// it. A fixed default keeps created/updated deterministic across runs.
var DefaultNow = time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)

// FakeClock is a Clock whose time tests control explicitly.
type FakeClock struct{ now time.Time }

// Now reports the current fake time.
func (c *FakeClock) Now() time.Time { return c.now }

// Set jumps the clock to t.
func (c *FakeClock) Set(t time.Time) { c.now = t }

// Advance moves the clock forward by d (handy for asserting `updated` bumps in
// later slices).
func (c *FakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// Result captures one in-process command run.
type Result struct {
	Code   int
	Stdout string
	Stderr string
}

// Harness drives the CLI in-process against a temporary store.
type Harness struct {
	t             *testing.T
	Dir           string             // project directory; the store lives at Dir/.beaver
	UserConfigDir string             // per-machine user-config dir; separate from Dir, never committed (ADR 0008)
	Clock         *FakeClock         // controllable time source
	Env           map[string]string  // environment seen by the CLI
	IsTTY         bool               // whether stdout looks interactive (default false → JSON)
	StdinIsTTY    bool               // the interactivity signal that gates human identity setup (default false)
	StdinText     string             // interactive input fed to prompts (identity confirmation)
	VCS           vcs.System         // VCS integration (identity seed + commit); nil means no adapter
	NewID         func() string      // ID generator override; nil uses the real one
	Editor        func(string) error // fake editor: given a file path, it may rewrite the file to simulate a human editing in $EDITOR; nil means no editor is available
}

// New returns a harness backed by a fresh temp directory. The store is not yet
// initialized — call Init or Run("init"). The user-config directory is a distinct
// temp dir, so identity (per-machine, never committed) is always kept apart from
// the project store.
func New(t *testing.T) *Harness {
	t.Helper()
	return &Harness{
		t:             t,
		Dir:           t.TempDir(),
		UserConfigDir: t.TempDir(),
		Clock:         &FakeClock{now: DefaultNow},
		Env:           map[string]string{},
		IsTTY:         false,
	}
}

// Run executes one CLI command in-process and returns its result.
func (h *Harness) Run(args ...string) Result {
	h.t.Helper()
	newID := h.NewID
	if newID == nil {
		newID = issue.NewID
	}
	var stdout, stderr bytes.Buffer
	code := cli.Run(cli.Env{
		Args:          args,
		Stdin:         strings.NewReader(h.StdinText),
		Stdout:        &stdout,
		Stderr:        &stderr,
		WorkDir:       h.Dir,
		Getenv:        func(k string) string { return h.Env[k] },
		Clock:         h.Clock,
		NewID:         newID,
		Edit:          h.Editor,
		StdoutIsTTY:   h.IsTTY,
		StdinIsTTY:    h.StdinIsTTY,
		VCS:           h.VCS,
		UserConfigDir: h.UserConfigDir,
	})
	return Result{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

// MustRun runs a command and fails the test if it exits non-zero.
func (h *Harness) MustRun(args ...string) Result {
	h.t.Helper()
	r := h.Run(args...)
	if r.Code != 0 {
		h.t.Fatalf("beaver %s: exit %d\nstderr: %s", strings.Join(args, " "), r.Code, r.Stderr)
	}
	return r
}

// Init initializes the store and returns the harness for chaining.
func (h *Harness) Init() *Harness {
	h.t.Helper()
	h.MustRun("init")
	return h
}

// BeaverDir is the path to the .beaver directory.
func (h *Harness) BeaverDir() string { return filepath.Join(h.Dir, ".beaver") }

// IssuesDir is the path to the issues directory.
func (h *Harness) IssuesDir() string { return filepath.Join(h.BeaverDir(), "issues") }

// IssueFiles returns the names (not full paths) of issue files, sorted. It fails
// the test if the issues directory cannot be read.
func (h *Harness) IssueFiles() []string {
	h.t.Helper()
	entries, err := os.ReadDir(h.IssuesDir())
	if err != nil {
		h.t.Fatalf("read issues dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// ReadFile reads a file at a path relative to the .beaver directory (e.g.
// "config.yml" or "issues/x.md"), failing the test on error.
func (h *Harness) ReadFile(rel string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.BeaverDir(), rel))
	if err != nil {
		h.t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// WriteFile writes raw content to a path relative to the .beaver directory, for
// seeding hand-edited or malformed files in tests.
func (h *Harness) WriteFile(rel, content string) {
	h.t.Helper()
	path := filepath.Join(h.BeaverDir(), rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		h.t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.t.Fatalf("write %s: %v", rel, err)
	}
}

// DecodeJSON parses s as a JSON object and fails the test on error.
func (h *Harness) DecodeJSON(s string) map[string]any {
	h.t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		h.t.Fatalf("decode JSON: %v\ninput: %s", err, s)
	}
	return v
}
