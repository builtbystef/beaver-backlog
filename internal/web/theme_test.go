package web_test

// The theme control: the reader's own say over which palette the UI draws in,
// offered from the shell's sidebar. What is asserted here is what a reader can
// observe on the page: the control is there, it names its three states, and the
// script that puts the remembered one in force runs before the page is drawn.
// The palettes themselves are the token stylesheet's, pinned in tokens_test.go.

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestThemeControlIsInTheShellOnEveryPage(t *testing.T) {
	h, pages := shellPages(t)

	for _, path := range pages {
		if themeControl(get(h, path).Body.String()) == "" {
			t.Errorf("%s offers no theme control", path)
		}
	}
}

// Three states and no more: system, which is no override at all, and the two
// that override. A fourth would be a palette nothing draws.
func TestThemeControlOffersSystemLightAndDark(t *testing.T) {
	h, _ := shellPages(t)

	control := themeControl(get(h, "/").Body.String())

	if got := themeStates(control); !slices.Equal(got, []string{"dark", "light", "system"}) {
		t.Errorf("the theme control offers %v, want system, light and dark", got)
	}
	for _, name := range []string{"System", "Light", "Dark"} {
		if !strings.Contains(control, name) {
			t.Errorf("the theme control does not say %q: %s", name, control)
		}
	}
}

// Absent a choice the reader is on the operating system's preference, and the
// control has to say so before any script has run.
func TestThemeControlStartsOnSystem(t *testing.T) {
	h, _ := shellPages(t)

	control := themeControl(get(h, "/").Body.String())

	m := checkedState.FindStringSubmatch(control)
	if m == nil {
		t.Fatalf("the theme control marks no state as the one in force: %s", control)
	}
	if m[1] != "system" {
		t.Errorf("the theme control starts on %q, want system", m[1])
	}
	if got := len(checkedState.FindAllString(control, -1)); got != 1 {
		t.Errorf("the theme control marks %d states as in force, want 1", got)
	}
}

// The chosen palette has to be in force at the first paint, so the script that
// applies it is the one script the shell does not defer: a deferred one runs
// after the document is parsed, which is a page drawn in the other palette
// first.
func TestThemeIsAppliedBeforeThePageIsDrawn(t *testing.T) {
	h, pages := shellPages(t)

	for _, path := range pages {
		body := get(h, path).Body.String()
		tag := scriptTag(body, "/assets/theme.js")
		if tag == "" {
			t.Errorf("%s loads no theme script:\n%s", path, body)
			continue
		}
		if strings.Contains(tag, "defer") || strings.Contains(tag, "async") {
			t.Errorf("%s puts the theme off until the page is drawn: %s", path, tag)
		}
		if at := strings.Index(body, tag); at > strings.Index(body, "<body") {
			t.Errorf("%s loads the theme script below the head: %s", path, tag)
		}
	}
}

func TestThemeScriptIsServed(t *testing.T) {
	if got := get(newHandler(t, newStore(t)), "/assets/theme.js").Code; got != http.StatusOK {
		t.Errorf("GET /assets/theme.js = %d, want 200", got)
	}
}

var (
	themeTag     = regexp.MustCompile(`(?s)<fieldset[^>]*id="theme"[^>]*>.*?</fieldset>`)
	themeValue   = regexp.MustCompile(`value="([^"]*)"`)
	checkedState = regexp.MustCompile(`value="([^"]*)"[^>]*\bchecked\b`)
)

// themeControl is the sidebar's theme control as rendered, empty when the page
// renders none.
func themeControl(body string) string { return themeTag.FindString(body) }

// themeStates lists, sorted, the states the control offers.
func themeStates(control string) []string {
	var out []string
	for _, m := range themeValue.FindAllStringSubmatch(control, -1) {
		out = append(out, m[1])
	}
	slices.Sort(out)
	return out
}

// scriptTag is the tag loading one script, empty when the page loads it from
// nowhere.
func scriptTag(body, src string) string {
	return regexp.MustCompile(`<script[^>]*src="` + regexp.QuoteMeta(src) + `"[^>]*>`).FindString(body)
}
