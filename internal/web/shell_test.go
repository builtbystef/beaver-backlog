package web_test

// The app shell: the sidebar every page renders beside, what its navigation
// offers, and the two things the shell itself has to say — the banner naming
// the files the scan skipped, and the notice a redirect after a write leaves
// behind. What is asserted here is structure a reader can observe — a
// navigation entry, a badge's number, the term still in the search box — never
// the class names that draw it.

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// shellPages is every kind of page the shell has to hold: the views, the two
// forms, one issue, and an address that names nothing at all.
func shellPages(t *testing.T) (http.Handler, []string) {
	t.Helper()
	return shellPagesIn(t, newStore(t))
}

// shellPagesIn is shellPages over a store the caller placed, for the tests that
// care what the project is called.
func shellPagesIn(t *testing.T, dir string) (http.Handler, []string) {
	t.Helper()
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	return newHandler(t, dir), []string{
		"/", "/issues", "/graph", "/doctor",
		"/issues/new", "/issues/" + target.ID, "/issues/" + target.ID + "/edit",
		"/nope",
	}
}

func TestEveryPageRendersInsideTheShell(t *testing.T) {
	h, pages := shellPages(t)

	for _, path := range pages {
		body := get(h, path).Body.String()
		if !strings.Contains(body, `<main id="view"`) {
			t.Errorf("%s renders no main region:\n%s", path, body)
		}
		if nav(t, body) == "" {
			t.Errorf("%s renders no sidebar navigation:\n%s", path, body)
		}
		if strings.Contains(body, "topbar") {
			t.Errorf("%s still renders a topbar", path)
		}
	}
}

func TestSidebarOffersEveryViewAndMarksTheOneInHand(t *testing.T) {
	h, pages := shellPages(t)
	// The address the reader is on, and the entry that must say so. One issue
	// and its edit form live under Issues; the create form and an address that
	// names nothing are nowhere in particular, so those mark nothing.
	current := map[string]string{
		"/": "/", "/issues": "/issues", "/graph": "/graph", "/doctor": "/doctor",
	}
	for _, path := range pages {
		if strings.HasPrefix(path, "/issues/") && path != "/issues/new" {
			current[path] = "/issues"
		}
	}

	for _, path := range pages {
		entries := navEntries(t, get(h, path).Body.String())
		for _, want := range []string{"/", "/issues", "/graph", "/doctor"} {
			if _, ok := entries[want]; !ok {
				t.Errorf("%s offers no way to %s", path, want)
			}
		}
		for href, entry := range entries {
			marked := strings.Contains(entry.tag, `aria-current="page"`)
			if want := href == current[path]; marked != want {
				t.Errorf("%s marks %s as current = %v, want %v", path, href, marked, want)
			}
		}
	}
	for _, label := range []string{"Board", "Issues", "Graph", "Doctor"} {
		entries := navEntries(t, get(h, "/").Body.String())
		if !slicesContainsLabel(entries, label) {
			t.Errorf("the navigation offers no %s entry", label)
		}
	}
}

func TestDoctorEntryBadgesTheFilesTheScanSkipped(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Perfectly fine"})
	h := newHandler(t, dir)

	if got := badge(t, get(h, "/").Body.String()); got != 0 {
		t.Errorf("a store with nothing wrong badges Doctor with %d, want no badge", got)
	}

	for _, name := range []string{"broken.md", "also-broken.md"} {
		file := filepath.Join(dir, ".beaver", "issues", name)
		if err := os.WriteFile(file, []byte("not an issue file"), 0o644); err != nil {
			t.Fatalf("seed invalid file: %v", err)
		}
	}

	if got := badge(t, get(h, "/").Body.String()); got != 2 {
		t.Errorf("two unusable files badge Doctor with %d, want 2", got)
	}
}

func TestSidebarSearchBoxCarriesTheTermTheListWasFilteredBy(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	to := get(h, "/search?q=parsing").Header().Get("Location")
	if to != "/issues?search=parsing" {
		t.Fatalf("searching went to %q, want the filtered list", to)
	}

	box := searchBox(t, get(h, to).Body.String())
	if !strings.Contains(box, `action="/search"`) {
		t.Errorf("the search box does not submit to the search view: %s", box)
	}
	if !strings.Contains(box, `value="parsing"`) {
		t.Errorf("the search box has forgotten what was searched for: %s", box)
	}
}

// A write the reader is redirected away from still gets a word, and it arrives
// inside the shell like everything else.
func TestNoticeAfterAWriteReachesTheReader(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	body := get(h, "/?deleted=abc123").Body.String()

	if !strings.Contains(body, "abc123") {
		t.Errorf("the board says nothing about the write that redirected here:\n%s", body)
	}
	if !strings.Contains(body, `role="status"`) {
		t.Errorf("the notice is not announced as a status:\n%s", body)
	}
}

// The brand slot names the project the reader is looking at rather than the
// application they opened — two projects served at once are two names. Every
// page says it, the one that names nothing included.
func TestBrandSlotNamesTheProjectOnEveryPage(t *testing.T) {
	h, pages := shellPagesIn(t, newStoreNamed(t, "orbital-mechanics"))

	for _, path := range pages {
		slot := brandSlot(t, get(h, path).Body.String())
		if !strings.Contains(slot, "orbital-mechanics") {
			t.Errorf("%s does not name the project in the brand slot: %s", path, slot)
		}
		if strings.Contains(slot, "Beaver Backlog") {
			t.Errorf("%s still names the application in the brand slot: %s", path, slot)
		}
	}
}

// A name too long for the rail is cut short by the stylesheet, so the whole of
// it has to remain reachable on the element itself.
func TestBrandSlotKeepsTheWholeNameForHover(t *testing.T) {
	name := "a-project-with-a-really-rather-long-directory-name"
	h, _ := shellPagesIn(t, newStoreNamed(t, name))

	slot := brandSlot(t, get(h, "/").Body.String())

	if !strings.Contains(slot, `title="`+name+`"`) {
		t.Errorf("the brand slot offers no way to read the whole name: %s", slot)
	}
}

// The brand slot is still the way back to the Board, and still a link, which is
// what takes the focus ring the design system draws on anything focused from
// the keyboard.
func TestBrandSlotLeadsToTheBoard(t *testing.T) {
	h, pages := shellPagesIn(t, newStoreNamed(t, "orbital-mechanics"))

	for _, path := range pages {
		slot := brandSlot(t, get(h, path).Body.String())
		m := anchorRef.FindStringSubmatch(slot)
		if m == nil {
			t.Errorf("%s renders no link in the brand slot: %s", path, slot)
			continue
		}
		if m[1] != "/" {
			t.Errorf("%s brand slot leads to %q, want the Board", path, m[1])
		}
	}
}

// The tab is how two projects open at once are told apart, so the project comes
// first — a narrow tab strip cuts the end off, never the front.
func TestPageTitleNamesTheProjectFirst(t *testing.T) {
	h, pages := shellPagesIn(t, newStoreNamed(t, "orbital-mechanics"))

	for _, path := range pages {
		title := pageTitle(t, get(h, path).Body.String())
		if !strings.HasPrefix(title, "orbital-mechanics") {
			t.Errorf("the tab for %s reads %q, which does not start with the project", path, title)
		}
	}
}

// A project that names itself in its committed config is called that everywhere
// the shell names it, the rail and the tab both, rather than after the directory
// it sits in.
func TestBrandSlotAndTitleUseTheConfiguredName(t *testing.T) {
	dir := newStoreNamed(t, "orbital-mechanics")
	if err := open(t, dir).SetProjectName("Apollo Guidance"); err != nil {
		t.Fatalf("SetProjectName: %v", err)
	}
	h, pages := shellPagesIn(t, dir)

	for _, path := range pages {
		body := get(h, path).Body.String()
		if slot := brandSlot(t, body); !strings.Contains(slot, "Apollo Guidance") {
			t.Errorf("%s does not name the project by its configured name: %s", path, slot)
		}
		if title := pageTitle(t, body); !strings.HasPrefix(title, "Apollo Guidance") {
			t.Errorf("the tab for %s reads %q, which does not start with the configured name", path, title)
		}
		if strings.Contains(body, "orbital-mechanics") {
			t.Errorf("%s still names the project after its directory", path)
		}
	}
}

// The wordmark is a lockup whose "B" is the beaver itself; at rail size the
// animal does not survive being cap-height, so the shell wears the icon mark
// instead and the wordmark asset goes with it.
func TestShellNoLongerCarriesTheWordmark(t *testing.T) {
	h, pages := shellPagesIn(t, newStoreNamed(t, "orbital-mechanics"))

	for _, path := range pages {
		if body := get(h, path).Body.String(); strings.Contains(body, "logo.svg") {
			t.Errorf("%s still draws the wordmark", path)
		}
	}
	if got := get(h, "/assets/logo.svg").Code; got != http.StatusNotFound {
		t.Errorf("the wordmark is still served: GET /assets/logo.svg = %d, want 404", got)
	}
}

var (
	navBlock  = regexp.MustCompile(`(?s)<nav[^>]*aria-label="Views"[^>]*>(.*?)</nav>`)
	navAnchor = regexp.MustCompile(`(?s)<a([^>]*)>(.*?)</a>`)
	anchorRef = regexp.MustCompile(`href="([^"]*)"`)
	badgeText = regexp.MustCompile(`>\s*(\d+)\s*<`)
	searchTag = regexp.MustCompile(`(?s)<form[^>]*role="search"[^>]*>.*?</form>`)
	brandTag  = regexp.MustCompile(`(?s)<header[^>]*>.*?</header>`)
	titleTag  = regexp.MustCompile(`<title>(.*?)</title>`)
)

// navEntry is one sidebar entry read back off the page: the anchor's own
// attributes, and what it says inside.
type navEntry struct{ tag, inner string }

// nav is the sidebar's navigation region, empty when the page renders none.
func nav(t *testing.T, body string) string {
	t.Helper()
	m := navBlock.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// navEntries reads the sidebar's entries back off a page, keyed by where each
// one goes.
func navEntries(t *testing.T, body string) map[string]navEntry {
	t.Helper()
	region := nav(t, body)
	if region == "" {
		t.Fatalf("no sidebar navigation on the page:\n%s", body)
	}
	out := map[string]navEntry{}
	for _, m := range navAnchor.FindAllStringSubmatch(region, -1) {
		href := anchorRef.FindStringSubmatch(m[1])
		if href == nil {
			continue
		}
		out[href[1]] = navEntry{tag: m[1], inner: m[2]}
	}
	return out
}

func slicesContainsLabel(entries map[string]navEntry, label string) bool {
	for _, e := range entries {
		if strings.Contains(e.inner, label) {
			return true
		}
	}
	return false
}

// badge is the number the Doctor entry wears, and zero when it wears none.
func badge(t *testing.T, body string) int {
	t.Helper()
	entry, ok := navEntries(t, body)["/doctor"]
	if !ok {
		t.Fatalf("no Doctor entry in the navigation:\n%s", body)
	}
	m := badgeText.FindStringSubmatch(entry.inner)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("Doctor badge reads %q: %v", m[1], err)
	}
	return n
}

// searchBox is the shell's search form as rendered.
func searchBox(t *testing.T, body string) string {
	t.Helper()
	m := searchTag.FindString(body)
	if m == "" {
		t.Fatalf("no search box on the page:\n%s", body)
	}
	return m
}

// brandSlot is the sidebar's brand region as rendered: the icon mark and the
// project's name, above the navigation on the same rail.
func brandSlot(t *testing.T, body string) string {
	t.Helper()
	m := brandTag.FindString(body)
	if m == "" {
		t.Fatalf("no brand slot on the page:\n%s", body)
	}
	return m
}

// pageTitle is what the browser puts on the tab.
func pageTitle(t *testing.T, body string) string {
	t.Helper()
	m := titleTag.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("the page carries no title:\n%s", body)
	}
	return m[1]
}
