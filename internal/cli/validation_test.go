package cli_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// stamps are the two required timestamp lines every seeded raw frontmatter needs.
const stamps = "created: 2026-06-27T18:30:00Z\nupdated: 2026-06-27T18:30:00Z\n"

// Invalid files are skipped, not fatal: the command still serves the valid issues,
// with one stderr warning per invalid file naming it and its specific problem.
func TestListWarnsAndSkipsInvalidFiles(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "good11", "Keep me", issue.StateTodo, beavertest.DefaultNow)
	h.WriteFile("issues/bad-yaml.md", "this is not an issue file\n")
	h.WriteFile("issues/no-id.md", "---\ntitle: No id\nstate: todo\n"+stamps+"---\n")
	h.WriteFile("issues/bad-state.md", "---\nid: bad222\ntitle: Bad state\nstate: archived\n"+stamps+"---\n")

	r := h.Run("list", "--state", "all")

	if r.Code != 0 {
		t.Fatalf("list exit = %d, want 0 (skip invalid, keep going)\nstderr: %s", r.Code, r.Stderr)
	}
	if got := ids(decodeArray(t, r.Stdout)); !slices.Equal(got, []string{"good11"}) {
		t.Errorf("listed %v, want just the valid issue [good11]", got)
	}

	for _, w := range []struct{ file, problem string }{
		{"bad-yaml.md", "frontmatter"},
		{"no-id.md", "missing id"},
		{"bad-state.md", "archived"}, // the offending value is named
	} {
		if !strings.Contains(r.Stderr, w.file) {
			t.Errorf("stderr should name %s:\n%s", w.file, r.Stderr)
		}
		if !strings.Contains(r.Stderr, w.problem) {
			t.Errorf("stderr should state %q for %s:\n%s", w.problem, w.file, r.Stderr)
		}
	}
}

// YAML admits non-finite floats (.nan, ±.inf) that encoding/json refuses. One such
// custom value must not fail an entire list, or make done report failure after its
// write succeeded.
func TestNonFiniteCustomValueDoesNotBreakJSON(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "good11", "Innocent bystander", issue.StateTodo, beavertest.DefaultNow)
	h.WriteFile("issues/nan001-odd-weight.md",
		"---\nid: nan001\ntitle: Odd Weight\nstate: todo\nweight: .nan\n"+stamps+"---\n")

	r := h.Run("list", "--state", "all")
	if r.Code != 0 {
		t.Fatalf("list exit = %d, want 0 despite a NaN custom value\nstderr: %s", r.Code, r.Stderr)
	}
	if got := ids(decodeArray(t, r.Stdout)); !slices.Equal(got, []string{"good11", "nan001"}) {
		t.Errorf("listed %v, want both issues", got)
	}
	if !strings.Contains(r.Stdout, `"weight": "NaN"`) {
		t.Errorf("the non-finite value should render as its name:\n%s", r.Stdout)
	}

	done := h.Run("done", "nan001")
	if done.Code != 0 {
		t.Fatalf("done exit = %d, want 0: the write succeeded and the report must too\nstderr: %s", done.Code, done.Stderr)
	}
	if out := h.DecodeJSON(done.Stdout); out["state"] != "done" {
		t.Errorf("done result state = %v, want done", out["state"])
	}
}

// The warning lands on stderr, never stdout, so it cannot corrupt the JSON an
// agent parses.
func TestInvalidFileWarningStaysOffStdout(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "good11", "Keep me", issue.StateTodo, beavertest.DefaultNow)
	h.WriteFile("issues/broken.md", "not an issue\n")

	r := h.Run("list")
	if strings.Contains(r.Stdout, "skipping") || strings.Contains(r.Stdout, "broken.md") {
		t.Errorf("the warning leaked onto stdout:\n%s", r.Stdout)
	}
	if got := ids(decodeArray(t, r.Stdout)); !slices.Equal(got, []string{"good11"}) {
		t.Errorf("stdout is not clean JSON of the valid issues: %v", got)
	}
	if !strings.Contains(r.Stderr, "broken.md") {
		t.Errorf("the warning should be on stderr:\n%s", r.Stderr)
	}
}

func TestShowWarnsButStillRendersValidIssue(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "good11", "Show me", issue.StateTodo, beavertest.DefaultNow)
	h.WriteFile("issues/broken.md", "this is not an issue file\n")

	r := h.Run("show", "good11")
	if r.Code != 0 {
		t.Fatalf("show exit = %d, want 0\nstderr: %s", r.Code, r.Stderr)
	}
	if got := h.DecodeJSON(r.Stdout); got["id"] != "good11" {
		t.Errorf("show id = %v, want good11", got["id"])
	}
	if !strings.Contains(r.Stderr, "broken.md") {
		t.Errorf("show should warn about the broken file:\n%s", r.Stderr)
	}
}

// An unknown frontmatter key is preserved lint, not a validation failure: read
// paths do not warn about it, since reporting a stray or typo'd key is doctor's
// job.
func TestUnknownFrontmatterKeyIsLintNotInvalid(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/cst111-custom.md", "---\nid: cst111\ntitle: Custom\nstate: todo\nsprint: 7\n"+stamps+"---\n\nBody.\n")

	r := h.Run("list", "--state", "all")
	if r.Code != 0 {
		t.Fatalf("list exit = %d, want 0\nstderr: %s", r.Code, r.Stderr)
	}
	if got := ids(decodeArray(t, r.Stdout)); !slices.Equal(got, []string{"cst111"}) {
		t.Errorf("listed %v, want the unknown-key issue (it is valid, not skipped)", got)
	}
	if strings.Contains(r.Stderr, "skipping") || strings.Contains(r.Stderr, "cst111") {
		t.Errorf("an unknown key is preserved lint, not a read-time warning:\n%s", r.Stderr)
	}

	shown := h.DecodeJSON(h.MustRun("show", "cst111").Stdout)
	if custom, _ := shown["custom"].(map[string]any); custom["sprint"] != float64(7) {
		t.Errorf("unknown key not preserved through show: %v", shown["custom"])
	}
}

// A rewriting command preserves unknown frontmatter keys and completes, rather
// than dropping data or refusing.
func TestRewriteDoesNotRefuseUnknownKeys(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/cst111-custom.md", "---\nid: cst111\ntitle: Custom\nstate: todo\nsprint: 7\n"+stamps+"---\n\nBody.\n")

	r := h.Run("done", "cst111")
	if r.Code != 0 {
		t.Fatalf("done on an unknown-key file exit = %d, want 0 (preserve, never refuse)\nstderr: %s", r.Code, r.Stderr)
	}
	shown := h.DecodeJSON(h.MustRun("show", "cst111").Stdout)
	if shown["state"] != "done" {
		t.Errorf("state = %v, want done", shown["state"])
	}
	if custom, _ := shown["custom"].(map[string]any); custom["sprint"] != float64(7) {
		t.Errorf("rewrite dropped the unknown key: %v", shown["custom"])
	}
}
