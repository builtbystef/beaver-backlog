package web_test

// The filter bar as a reader meets it: an address produces the listing the core
// returns for it, the same address reproduces the same view in a fresh tab, an
// htmx request gets the fragment instead of the whole page, and the board keeps
// its four columns however far the filters cut. What the filters *mean* is the
// core's own suite; none of it is re-asserted here.

import (
	"net/http"
	"net/http/httptest"
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
// draws the same page — listing and controls alike.
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

// The bar submits on its own without JavaScript: a plain GET form aimed at the
// view it filters.
func TestFilterBarSubmitsWithoutJavaScript(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	for _, c := range []struct{ path, action string }{{"/", "/"}, {"/issues", "/issues"}} {
		body := get(h, c.path).Body.String()
		if !strings.Contains(body, `method="get"`) || !strings.Contains(body, `action="`+c.action+`"`) {
			t.Errorf("%s has no plain filter form aimed at %s:\n%s", c.path, c.action, body)
		}
		for _, field := range []string{`name="state"`, `name="ready"`, `name="blocked"`, `name="label"`, `name="priority"`, `name="assignee"`, `name="actor"`, `name="parent"`, `name="search"`} {
			if !strings.Contains(body, field) {
				t.Errorf("%s filter bar is missing %s", c.path, field)
			}
		}
	}
}

// The board's "show all" is a query parameter like any other, so filtering does
// not quietly re-hide what the reader asked to see.
func TestFilteringKeepsTheBoardsShowAllInTheForm(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	body := get(h, "/?all=done").Body.String()

	if !strings.Contains(body, `type="hidden" name="all" value="done"`) {
		t.Errorf("the filter form drops the shown-in-full column:\n%s", body)
	}
}

func hxGet(h http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}
