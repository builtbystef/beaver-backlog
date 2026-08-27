package web_test

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The token stylesheet is the design system's colour table: one set of names
// given a value once per palette, so nothing below it knows which theme it is
// drawing. These are the three blocks that carry a palette — the light base,
// the dark a system preference asks for, and the dark a reader chose.
var paletteBlocks = map[string]*regexp.Regexp{
	"light":       regexp.MustCompile(`(?m)^:root\s*\{([^}]*)\}`),
	"system dark": regexp.MustCompile(`:root:not\(\[data-theme=["']?light["']?\]\)\s*\{([^}]*)\}`),
	"chosen dark": regexp.MustCompile(`:root\[data-theme=["']?dark["']?\]\s*\{([^}]*)\}`),
}

// paletteTokens are the categories the design system promises: neutral
// surfaces and text, the beaver-orange accent, and the muted marks for state,
// priority, and the derived conditions.
var paletteTokens = []string{
	"--canvas", "--surface", "--surface-raised", "--surface-hover",
	"--line", "--line-strong",
	"--ink", "--ink-muted", "--ink-subtle", "--ink-on-accent",
	"--accent", "--accent-hover", "--accent-soft",
	"--state-todo", "--state-in-progress", "--state-done", "--state-cancelled",
	"--priority-urgent", "--priority-high", "--priority-medium", "--priority-low",
	"--condition-blocked", "--condition-ready", "--condition-stuck",
}

// The stylesheet is compiled by a tool nobody has at build time, so the one
// thing worth pinning is that the committed copy really did reach the binary.
func TestTokenStylesheetIsServed(t *testing.T) {
	res := get(newHandler(t, newStore(t)), "/assets/tailwind.css")
	if res.Code != http.StatusOK {
		t.Fatalf("GET /assets/tailwind.css = %d, want 200", res.Code)
	}
	if got := res.Body.Len(); got < 1000 {
		t.Errorf("token stylesheet is %d bytes, want a compiled sheet", got)
	}
}

func TestEveryPaletteDeclaresTheSameTokens(t *testing.T) {
	sheet := get(newHandler(t, newStore(t)), "/assets/tailwind.css").Body.String()

	declared := map[string][]string{}
	for palette, block := range paletteBlocks {
		m := block.FindStringSubmatch(sheet)
		if m == nil {
			t.Fatalf("token stylesheet declares no %s palette", palette)
		}
		declared[palette] = customProperties(m[1])
	}
	for palette, got := range declared {
		for _, want := range paletteTokens {
			if !slices.Contains(got, want) {
				t.Errorf("%s palette is missing token %s", palette, want)
			}
		}
	}
	light := declared["light"]
	for palette, got := range declared {
		if palette != "light" && !slices.Equal(got, light) {
			t.Errorf("%s palette declares %v; the light palette declares %v", palette, got, light)
		}
	}
}

// customProperties lists, sorted, the custom properties a declaration block
// sets — the block's contribution to the one set of names.
func customProperties(block string) []string {
	var out []string
	for _, decl := range strings.Split(block, ";") {
		name, _, ok := strings.Cut(decl, ":")
		if name = strings.TrimSpace(name); ok && strings.HasPrefix(name, "--") {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}
