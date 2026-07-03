package cli_test

import (
	"slices"
	"strings"
	"testing"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

// stamps are the two required timestamp lines every seeded raw frontmatter needs.
const stamps = "created: 2026-06-27T18:30:00Z\nupdated: 2026-06-27T18:30:00Z\n"

// AC: an invalid file is skipped with a loud warning; valid issues still list. The
// command succeeds (graceful, not fail-fast), lists only the valid issue, and
// prints one stderr warning per invalid file that names the file and the specific
// problem (ADR 0005). The three invalid classes — unparseable frontmatter, a
// missing id, and an illegal state — are each skipped and each named.
func TestListWarnsAndSkipsInvalidFiles(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "good11", "Keep me", issue.StateTodo, beavertest.DefaultNow)
	h.WriteFile("issues/bad-yaml.md", "this is not an issue file\n")
	h.WriteFile("issues/no-id.md", "---\ntitle: No id\nstate: todo\n"+stamps+"---\n")
	h.WriteFile("issues/bad-state.md", "---\nid: bad222\ntitle: Bad state\nstate: archived\n"+stamps+"---\n")

	r := h.Run("list", "--state", "all")

	// Graceful: the command still succeeds and serves the valid issue.
	if r.Code != 0 {
		t.Fatalf("list exit = %d, want 0 (skip invalid, keep going)\nstderr: %s", r.Code, r.Stderr)
	}
	if got := ids(decodeArray(t, r.Stdout)); !slices.Equal(got, []string{"good11"}) {
		t.Errorf("listed %v, want just the valid issue [good11]", got)
	}

	// Loud: one warning per invalid file, each naming the file and its specific
	// problem (not a generic "a file was bad").
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

// The warning lands on stderr, never stdout, so it cannot corrupt the JSON an
// agent parses (ADR 0013): stdout is still a clean, parseable array.
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

// AC: show keeps working past an unrelated invalid file — it renders the resolved
// issue and warns about the broken one, rather than failing the whole command.
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

// AC / ADR 0014: an unknown frontmatter key is lint, not a validation failure. A
// file carrying one still lists and shows (the issue is valid and usable), the key
// is preserved in output, and read paths do NOT warn about it here — reporting a
// stray/typo'd key is doctor's job (n9b4a7), not a per-read skip.
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

// ADR 0014 supersedes this issue's original closed-schema criterion: a rewriting
// command must NOT refuse a file that carries unknown frontmatter keys. It
// preserves the key and completes the transition, rather than dropping data or
// aborting.
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
