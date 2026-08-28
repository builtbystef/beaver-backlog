package web_test

// The filter bar as a reader meets it: an address produces the listing the core
// returns for it, the same address reproduces the same view in a fresh tab, an
// htmx request gets the fragment instead of the whole page, and the board keeps
// its four columns however far the filters cut. What the filters *mean* is the
// core's own suite; none of it is re-asserted here.

import (
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The worked example: state, label, and text together, against the listing the
// core returns for exactly that query.
func TestFilteredAddressListsWhatTheCoreReturnsForIt(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	create(t, svc, core.Draft{Title: "Local web UI", Labels: []string{"spec"}})
	create(t, svc, core.Draft{Title: "CLI polish", Body: "Nothing about the browser here.", Labels: []string{"spec"}})
	create(t, svc, core.Draft{Title: "Web board", Body: "No spec label.", Labels: []string{"bug"}})
	started := create(t, svc, core.Draft{Title: "Web graph", Labels: []string{"spec"}})
	start(t, svc, started.ID)

	body := get(newHandler(t, dir), "/issues?state=todo&label=spec&search=web").Body.String()

	listing, err := svc.List(core.Query{
		States: []issue.State{issue.StateTodo},
		Labels: []string{"spec"},
		Text:   "web",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listing.Issues) == 0 {
		t.Fatal("the core matches nothing for the worked example; the fixture is wrong")
	}
	all, err := svc.List(core.Query{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	for _, iss := range all.Issues {
		want := slices.ContainsFunc(listing.Issues, func(m issue.Issue) bool { return m.ID == iss.ID })
		if got := strings.Contains(body, iss.ID); got != want {
			t.Errorf("issue %s (%q) shown = %v, want %v", iss.ID, iss.Title, got, want)
		}
	}
}

// A filtered address is the whole view: the same one, pasted into a fresh tab,
// draws the same page, listing and controls alike.
func TestPastedAddressReproducesTheView(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	create(t, svc, core.Draft{Title: "Fix flag parsing", Labels: []string{"bug"}, Priority: issue.PriorityHigh})
	create(t, svc, core.Draft{Title: "Something else"})
	address := "/issues?state=todo&ready=1&label=bug&priority=high&assignee=unassigned&search=flag"

	first := get(newHandler(t, dir), address).Body.String()
	fresh := get(newHandler(t, dir), address).Body.String()

	if first != fresh {
		t.Error("the same address drew two different pages")
	}
	for _, want := range []string{`value="bug"`, `value="flag"`, "checked"} {
		if !strings.Contains(first, want) {
			t.Errorf("the bar does not carry %s back:\n%s", want, first)
		}
	}
	if strings.Contains(first, "Something else") {
		t.Error("an issue the filters exclude is on the page")
	}
}

// Changing a control asks for the fragment, not the page; without htmx the very
// same address is a plain, whole page.
func TestFragmentRequestOmitsThePageChrome(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, path := range []string{"/", "/issues"} {
		fragment := hxGet(h, path)
		if fragment.Code != http.StatusOK {
			t.Errorf("GET %s (htmx) = %d, want 200", path, fragment.Code)
			continue
		}
		body := fragment.Body.String()
		for _, chrome := range []string{"<!doctype", "<html", "<title", "</body>"} {
			if strings.Contains(strings.ToLower(body), chrome) {
				t.Errorf("GET %s (htmx) returned %s — the fragment is the page's inside only", path, chrome)
			}
		}
		if !strings.Contains(body, `id="issues"`) {
			t.Errorf("GET %s (htmx) has no issue fragment to swap:\n%s", path, body)
		}
		whole := get(h, path).Body.String()
		if !strings.Contains(strings.ToLower(whole), "<!doctype") {
			t.Errorf("GET %s without htmx is not a whole page:\n%s", path, whole)
		}
		if !strings.Contains(whole, `id="issues"`) {
			t.Errorf("GET %s without htmx has no issue fragment:\n%s", path, whole)
		}
	}
}

// The board narrows to what the filters select and stays a board: four columns,
// however few cards are left.
func TestBoardFiltersNarrowTheCardsAndKeepTheColumns(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	wanted := create(t, svc, core.Draft{Title: "Fix flag parsing", Labels: []string{"bug"}})
	other := create(t, svc, core.Draft{Title: "Write the docs"})
	started := create(t, svc, core.Draft{Title: "Also a bug", Labels: []string{"bug"}})
	start(t, svc, started.ID)

	body := get(newHandler(t, dir), "/?label=bug").Body.String()

	order, cards := board(t, body)
	if want := []string{"todo", "in-progress", "done", "cancelled"}; !slices.Equal(order, want) {
		t.Errorf("columns = %v, want %v — filtering never removes a column", order, want)
	}
	if want := []string{wanted.ID}; !slices.Equal(cards["todo"], want) {
		t.Errorf("todo column = %v, want %v", cards["todo"], want)
	}
	if want := []string{started.ID}; !slices.Equal(cards["in-progress"], want) {
		t.Errorf("in-progress column = %v, want %v", cards["in-progress"], want)
	}
	if strings.Contains(body, other.ID) {
		t.Errorf("issue %s (%q) survives a filter it does not match", other.ID, other.Title)
	}
}

// A parent nobody can resolve is a typed reference, not a broken page: the core's
// words land on the view the reader is already looking at.
func TestUnresolvableParentIsSaidOnThePage(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, path := range []string{"/?parent=no-such-issue", "/issues?parent=no-such-issue", "/graph?parent=no-such-issue"} {
		res := get(h, path)
		if res.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 — the view still renders", path, res.Code)
		}
		body := res.Body.String()
		if !strings.Contains(body, "no issue matches") || !strings.Contains(body, "no-such-issue") {
			t.Errorf("GET %s does not say what went wrong:\n%s", path, body)
		}
		if !strings.Contains(body, `name="parent"`) {
			t.Errorf("GET %s lost the filter bar, so the typo cannot be fixed in place:\n%s", path, body)
		}
	}
	if cards := get(h, "/?parent=no-such-issue").Body.String(); strings.Contains(cards, "Fix flag parsing") {
		t.Error("an unresolvable parent still listed issues")
	}
}

// The board's "show all" is a query parameter like any other, so nothing the
// toolbar offers quietly re-hides what the reader asked to see: not a control,
// not a chip, and not clearing every filter.
func TestNarrowingKeepsTheBoardsShowAllColumn(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	body := get(h, "/?all=done&state=todo").Body.String()

	if !strings.Contains(body, `type="hidden" name="all" value="done"`) {
		t.Errorf("the filter form drops the shown-in-full column:\n%s", body)
	}
	for _, where := range append(chips(t, body), link{href: clearAll(t, body), label: "clear all"}) {
		if !strings.Contains(where.href, "all=done") {
			t.Errorf("following %q goes to %s, which re-hides the column shown in full", where.label, where.href)
		}
	}
}

func hxGet(h http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

// The toolbar as a reader meets it on every view it narrows: the whole filter
// vocabulary within reach, the active filters said as chips that remove
// themselves, a refused reference worded where it was typed, and the sidebar's
// box and the toolbar's text field agreeing on the one term.

// Every filter the views understand is reachable from the toolbar, and the
// toolbar stands outside the region a filter request replaces: it is the
// view's chrome, not part of what filtering redraws.
func TestToolbarOffersEveryFilterOnEveryView(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	for _, view := range filteredViews {
		body := get(h, view).Body.String()
		bar := toolbar(t, body)
		for _, control := range []string{"state", "ready", "blocked", "priority", "assignee", "actor", "label", "parent", "search"} {
			if !strings.Contains(bar, `name="`+control+`"`) {
				t.Errorf("%s: the toolbar offers no %s control:\n%s", view, control, bar)
			}
		}
		if at, swapped := strings.Index(body, bar), strings.Index(body, `id="issues"`); at > swapped {
			t.Errorf("%s: the toolbar is inside the region filtering replaces", view)
		}
	}
}

// Each active filter is a chip that takes just itself off, and clearing all
// goes back to the view unnarrowed.
func TestEachChipRemovesItsOwnFilterAndClearAllRemovesThemAll(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	wanted := create(t, svc, core.Draft{Title: "Fix flag parsing", Labels: []string{"bug"}, Priority: issue.PriorityHigh})
	other := create(t, svc, core.Draft{Title: "Something else"})
	h := newHandler(t, dir)

	address := "/issues?state=todo&ready=1&label=bug&priority=high&assignee=unassigned&search=flag"
	body := get(h, address).Body.String()
	active := chips(t, body)
	if len(active) != 6 {
		t.Fatalf("six filters on the address are said as %d chips: %v", len(active), active)
	}

	for _, one := range active {
		narrower := get(h, one.href)
		if narrower.Code != http.StatusOK {
			t.Errorf("following the %q chip = %d, want 200", one.label, narrower.Code)
			continue
		}
		if got := chips(t, narrower.Body.String()); len(got) != len(active)-1 {
			t.Errorf("following the %q chip leaves %d filters, want %d: %v", one.label, len(got), len(active)-1, got)
		}
	}

	cleared := get(h, clearAll(t, body))
	if got := chips(t, cleared.Body.String()); len(got) != 0 {
		t.Errorf("clearing all leaves %v", got)
	}
	for _, iss := range []issue.Issue{wanted, other} {
		if !strings.Contains(cleared.Body.String(), iss.ID) {
			t.Errorf("the unfiltered view is missing issue %s (%q)", iss.ID, iss.Title)
		}
	}
}

// A reference that names no issue is the reader's to fix where they typed it,
// so the core's words stand beside the box holding it and nowhere else.
func TestRefusedReferenceIsSaidBesideTheBoxItWasTypedInto(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, view := range filteredViews {
		bar := toolbar(t, get(h, view+"?parent=no-such-issue").Body.String())
		box := strings.Index(bar, `name="parent"`)
		said := strings.Index(bar, "no issue matches")
		if box < 0 || said < 0 {
			t.Fatalf("%s: the toolbar carries the box at %d and the words at %d:\n%s", view, box, said, bar)
		}
		if said < box {
			t.Errorf("%s: the refusal is worded before the box it is about", view)
			continue
		}
		if between := bar[box:said]; strings.Contains(between, `name="`) && strings.Count(between, `name="`) > 1 {
			t.Errorf("%s: another control stands between the box and the words about it: %s", view, between)
		}
	}
}

// The sidebar's box and the toolbar's text field are one filter: every view
// that carries a toolbar says the same term in both, and an address that says
// no term leaves both empty.
func TestSidebarBoxAndToolbarTextFieldAreOneFilter(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, view := range filteredViews {
		filtered := get(h, view+"?search=parsing").Body.String()
		if box := textInput(t, searchBox(t, filtered), "q"); !strings.Contains(box, `value="parsing"`) {
			t.Errorf("%s: the sidebar box does not carry the term: %s", view, box)
		}
		if field := textInput(t, toolbar(t, filtered), "search"); !strings.Contains(field, `value="parsing"`) {
			t.Errorf("%s: the toolbar's text field does not carry the term: %s", view, field)
		}
		plain := get(h, view).Body.String()
		if box := textInput(t, searchBox(t, plain), "q"); strings.Contains(box, `value="parsing"`) {
			t.Errorf("%s unfiltered: the sidebar box still holds a term: %s", view, box)
		}
		if field := textInput(t, toolbar(t, plain), "search"); strings.Contains(field, `value="parsing"`) {
			t.Errorf("%s unfiltered: the toolbar's text field still holds a term: %s", view, field)
		}
	}
}

// filteredViews are the three views the toolbar narrows.
var filteredViews = []string{"/", "/issues", "/graph"}

var (
	toolbarForm = regexp.MustCompile(`(?s)<form[^>]*aria-label="Filter issues"[^>]*>.*?</form>`)
	chipList    = regexp.MustCompile(`(?s)<ul[^>]*aria-label="Active filters"[^>]*>(.*?)</ul>`)
	chipRegion  = regexp.MustCompile(`(?s)<div[^>]*id="filter-chips"[^>]*>(.*?)\n\s*</div>`)
	anchor      = regexp.MustCompile(`(?s)<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
)

// link is one anchor read back off a page: where it goes and what it says.
type link struct{ href, label string }

// toolbar is the filter toolbar as rendered, by its accessible name.
func toolbar(t *testing.T, body string) string {
	t.Helper()
	m := toolbarForm.FindString(body)
	if m == "" {
		t.Fatalf("no filter toolbar on the page:\n%s", body)
	}
	return m
}

// chips are the active filters the toolbar says one by one, in order.
func chips(t *testing.T, body string) []link {
	t.Helper()
	m := chipList.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no active-filter list on the page:\n%s", body)
	}
	var out []link
	for _, a := range anchor.FindAllStringSubmatch(m[1], -1) {
		out = append(out, read(a))
	}
	return out
}

var stripTags = regexp.MustCompile(`<[^>]*>`)

// read is one anchor's address and words as a browser would take them: the
// attribute unescaped, the words without the markup that decorates them.
func read(anchor []string) link {
	words := html.UnescapeString(stripTags.ReplaceAllString(anchor[2], ""))
	return link{
		href:  html.UnescapeString(anchor[1]),
		label: strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(words), "×")),
	}
}

// clearAll is where the toolbar's clear-all link goes.
func clearAll(t *testing.T, body string) string {
	t.Helper()
	m := chipRegion.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no chip region on the page:\n%s", body)
	}
	for _, a := range anchor.FindAllStringSubmatch(m[1], -1) {
		if one := read(a); one.label == "Clear all" {
			return one.href
		}
	}
	t.Fatalf("a filtered view offers no way to clear every filter:\n%s", m[1])
	return ""
}

// textInput is one named input inside a region, as rendered.
func textInput(t *testing.T, region, name string) string {
	t.Helper()
	m := regexp.MustCompile(`<input[^>]*name="` + name + `"[^>]*>`).FindString(region)
	if m == "" {
		t.Fatalf("no %q input in:\n%s", name, region)
	}
	return m
}
