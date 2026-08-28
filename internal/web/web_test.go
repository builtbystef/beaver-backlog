package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/web"
)

func TestNewWithoutStoreFails(t *testing.T) {
	if _, err := web.New(web.Config{WorkDir: t.TempDir(), Actor: "tester"}); !errors.Is(err, core.ErrNoStore) {
		t.Fatalf("New outside a store = %v, want core.ErrNoStore", err)
	}
}

func TestRouteStatuses(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	cases := []struct {
		path string
		want int
	}{
		{"/", http.StatusOK},
		{"/issues", http.StatusOK},
		{"/assets/htmx.min.js", http.StatusOK},
		{"/nope", http.StatusNotFound},
		{"/issues/anything", http.StatusNotFound},
	}
	for _, c := range cases {
		if got := get(h, c.path).Code; got != c.want {
			t.Errorf("GET %s = %d, want %d", c.path, got, c.want)
		}
	}
}

func TestListRendersEveryIssueInCoreOrder(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	create(t, svc, core.Draft{Title: "Low rider", Priority: issue.PriorityLow, Labels: []string{"chore"}})
	urgent := create(t, svc, core.Draft{Title: "Urgent thing", Priority: issue.PriorityUrgent})
	create(t, svc, core.Draft{Title: "No priority at all"})
	if _, err := svc.Start(urgent.ID, "tester", false); err != nil {
		t.Fatalf("start: %v", err)
	}

	body := get(newHandler(t, dir), "/issues").Body.String()

	want, err := svc.List(core.Query{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(want.Issues) != 3 {
		t.Fatalf("seeded %d issues, want 3", len(want.Issues))
	}
	prev := -1
	for _, iss := range want.Issues {
		at := strings.Index(body, iss.ID)
		if at < 0 {
			t.Fatalf("issue %s (%q) missing from the list page", iss.ID, iss.Title)
		}
		if at < prev {
			t.Errorf("issue %s renders out of the core's ordering", iss.ID)
		}
		prev = at
		for _, field := range []string{iss.Title, string(iss.State)} {
			if !strings.Contains(body, field) {
				t.Errorf("list page missing %q for issue %s", field, iss.ID)
			}
		}
	}
	for _, field := range []string{"urgent", "low", "chore", "tester"} {
		if !strings.Contains(body, field) {
			t.Errorf("list page missing %q", field)
		}
	}
}

// The handler holds no issue data between requests: an issue created after it
// was built shows up on the next render.
func TestListReflectsWritesAfterTheHandlerWasBuilt(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	if body := get(h, "/issues").Body.String(); strings.Contains(body, "Written later") {
		t.Fatal("empty store already renders the later issue")
	}

	later := create(t, open(t, dir), core.Draft{Title: "Written later"})

	if body := get(h, "/issues").Body.String(); !strings.Contains(body, later.ID) {
		t.Error("issue created after the handler was built is missing from the list")
	}
}

func TestBoardColumnsSpanTheLifecycleAndHoldTheirOwnIssues(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	waiting := create(t, svc, core.Draft{Title: "Waiting to start", Priority: issue.PriorityLow, Labels: []string{"chore"}})
	started := create(t, svc, core.Draft{Title: "Under way"})
	finished := create(t, svc, core.Draft{Title: "All finished"})
	dropped := create(t, svc, core.Draft{Title: "Never mind"})
	start(t, svc, started.ID)
	transition(t, svc, finished.ID, issue.StateDone)
	transition(t, svc, dropped.ID, issue.StateCancelled)

	body := get(newHandler(t, dir), "/").Body.String()

	order, cards := board(t, body)
	wantOrder := []string{"todo", "in-progress", "done", "cancelled"}
	if !slices.Equal(order, wantOrder) {
		t.Errorf("columns = %v, want %v", order, wantOrder)
	}
	for _, c := range []struct {
		state string
		want  []string
	}{
		{"todo", []string{waiting.ID}},
		{"in-progress", []string{started.ID}},
		{"done", []string{finished.ID}},
		{"cancelled", []string{dropped.ID}},
	} {
		if !slices.Equal(cards[c.state], c.want) {
			t.Errorf("%s column holds %v, want %v", c.state, cards[c.state], c.want)
		}
	}
	// A card carries what triage reads off it, and is a link to its issue.
	for _, field := range []string{"Waiting to start", waiting.ID, "low", "chore", "tester"} {
		if !strings.Contains(body, field) {
			t.Errorf("board missing %q", field)
		}
	}
	for _, iss := range []issue.Issue{waiting, started, finished, dropped} {
		if link := `href="/issues/` + iss.ID + `"`; !strings.Contains(body, link) {
			t.Errorf("card for %s is not a link to its detail page (%s)", iss.ID, link)
		}
	}
}

func TestBoardOrdersCardsWithinAColumnTheCoreWay(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	create(t, svc, core.Draft{Title: "Low rider", Priority: issue.PriorityLow})
	create(t, svc, core.Draft{Title: "Urgent thing", Priority: issue.PriorityUrgent})
	create(t, svc, core.Draft{Title: "No priority at all"})

	_, cards := board(t, get(newHandler(t, dir), "/").Body.String())

	listing, err := svc.List(core.Query{States: []issue.State{issue.StateTodo}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := make([]string, len(listing.Issues))
	for i, iss := range listing.Issues {
		want[i] = iss.ID
	}
	if !slices.Equal(cards["todo"], want) {
		t.Errorf("todo column = %v, want the core's ordering %v", cards["todo"], want)
	}
}

// Only the two terminal columns are windowed, and only until the reader asks
// for the rest.
func TestBoardWindowsTheTerminalColumnsToRecentUpdates(t *testing.T) {
	dir := newStore(t)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	clk := &testClock{now: now.AddDate(0, 0, -15)}
	svc := open(t, dir, core.WithClock(clk))
	staleTodo := create(t, svc, core.Draft{Title: "Long ignored"})
	staleDone := create(t, svc, core.Draft{Title: "Long finished"})
	staleCancelled := create(t, svc, core.Draft{Title: "Long dropped"})
	transition(t, svc, staleDone.ID, issue.StateDone)
	transition(t, svc, staleCancelled.ID, issue.StateCancelled)
	clk.now = now.AddDate(0, 0, -1)
	freshDone := create(t, svc, core.Draft{Title: "Just finished"})
	transition(t, svc, freshDone.ID, issue.StateDone)
	clk.now = now
	h := newHandler(t, dir, core.WithClock(clk))

	body := get(h, "/").Body.String()

	_, cards := board(t, body)
	if want := []string{staleTodo.ID}; !slices.Equal(cards["todo"], want) {
		t.Errorf("todo column = %v, want %v — open columns are never windowed", cards["todo"], want)
	}
	if want := []string{freshDone.ID}; !slices.Equal(cards["done"], want) {
		t.Errorf("done column = %v, want only the recent %v", cards["done"], want)
	}
	if len(cards["cancelled"]) != 0 {
		t.Errorf("cancelled column = %v, want nothing within the window", cards["cancelled"])
	}
	if !strings.Contains(body, `href="/?all=done"`) {
		t.Errorf("done column offers no way to show all:\n%s", body)
	}

	_, cards = board(t, get(h, "/?all=done&all=cancelled").Body.String())
	if want := []string{staleDone.ID, freshDone.ID}; !slices.Equal(cards["done"], want) {
		t.Errorf("done column shown in full = %v, want %v", cards["done"], want)
	}
	if want := []string{staleCancelled.ID}; !slices.Equal(cards["cancelled"], want) {
		t.Errorf("cancelled column shown in full = %v, want %v", cards["cancelled"], want)
	}
}

// Every page reaches the other views.
func TestEveryPageNavigatesToBoardAndList(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	for _, page := range []string{"/", "/issues", "/nope"} {
		body := get(h, page).Body.String()
		for _, link := range []string{`href="/"`, `href="/issues"`} {
			if !strings.Contains(body, link) {
				t.Errorf("%s has no navigation to %s", page, link)
			}
		}
	}
}

// Every view is a view like any other: a broken file costs a banner naming it
// and the reason, not a page (ADR 0003). Doctor is the exception, and reports
// the same file as a finding instead, so nothing is named twice.
func TestInvalidFileBecomesABannerNotAnError(t *testing.T) {
	dir := newStore(t)
	fine := create(t, open(t, dir), core.Draft{Title: "Perfectly fine"})
	broken := filepath.Join(dir, ".beaver", "issues", "broken.md")
	if err := os.WriteFile(broken, []byte("not an issue file"), 0o644); err != nil {
		t.Fatalf("seed invalid file: %v", err)
	}
	h := newHandler(t, dir)

	for _, view := range []string{"/", "/issues", "/graph", "/issues/" + fine.ID} {
		res := get(h, view)

		if res.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 despite the invalid file", view, res.Code)
			continue
		}
		body := res.Body.String()
		if !strings.Contains(body, "broken.md") {
			t.Errorf("%s does not name the skipped file:\n%s", view, body)
		}
		if !strings.Contains(body, "Perfectly fine") {
			t.Errorf("%s lost the valid issue; one broken file must not empty a view", view)
		}
	}
}

// The detail page is the whole file made readable: its own fields, its
// description, its notes in order, and every relationship as somewhere to go.
func TestDetailPageShowsEverythingTheFileHolds(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	settled := create(t, svc, core.Draft{Title: "Groundwork"})
	transition(t, svc, settled.ID, issue.StateDone)
	waiting := create(t, svc, core.Draft{Title: "Still going"})
	parent := create(t, svc, core.Draft{Title: "The spec"})
	target := create(t, svc, core.Draft{
		Title:     "Fix flag parsing",
		Body:      "The parser drops the last flag.",
		Priority:  issue.PriorityHigh,
		Labels:    []string{"bug"},
		DependsOn: []string{settled.ID, waiting.ID},
		Parent:    parent.ID,
	})
	child := create(t, svc, core.Draft{Title: "A slice of it", Parent: target.ID})
	dependent := create(t, svc, core.Draft{Title: "Waits on the fix", DependsOn: []string{target.ID}})
	start(t, svc, target.ID)
	note(t, svc, target.ID, "Reproduced on an empty argv.")
	note(t, svc, target.ID, "Cause is the loop bound.")
	addCustomField(t, dir, target.ID, "epic", "Q3 cleanup")

	res := get(newHandler(t, dir), "/issues/"+target.ID)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, want := range []string{
		target.ID, "Fix flag parsing", "in-progress", "high", "bug", "tester",
		"The parser drops the last flag.",
		"Reproduced on an empty argv.", "Cause is the loop bound.",
		string(issue.StateTodo), // the state of the dependency still unmet
		"epic", "Q3 cleanup",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q:\n%s", want, body)
		}
	}
	if first, second := strings.Index(body, "Reproduced on"), strings.Index(body, "Cause is"); first > second {
		t.Error("notes render out of the order they were written in")
	}
	// Every related issue is a way to get there.
	for _, rel := range []issue.Issue{settled, waiting, parent, child, dependent} {
		if link := `href="/issues/` + rel.ID + `"`; !strings.Contains(body, link) {
			t.Errorf("relationship to %s (%q) is not a link (%s)", rel.ID, rel.Title, link)
		}
	}
}

// A URL takes any reference the CLI takes, and nothing else.
func TestDetailResolvesEveryFormOfReference(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, ref := range []string{target.ID, "fix-flag-parsing", target.ID + "-fix-flag-parsing"} {
		res := get(h, "/issues/"+ref)
		if res.Code != http.StatusOK {
			t.Errorf("GET /issues/%s = %d, want 200", ref, res.Code)
			continue
		}
		if !strings.Contains(res.Body.String(), target.ID) {
			t.Errorf("GET /issues/%s renders some other issue", ref)
		}
	}
	if res := get(h, "/issues/no-such-issue"); res.Code != http.StatusNotFound {
		t.Errorf("GET /issues/no-such-issue = %d, want 404", res.Code)
	}
}

// An address this interface does not serve is answered by this interface: its
// own page, inside the shell, naming what was asked for.
func TestUnknownAddressGetsThisInterfacesOwnPage(t *testing.T) {
	res := get(newHandler(t, newStore(t)), "/no-such-page")

	if res.Code != http.StatusNotFound {
		t.Fatalf("GET /no-such-page = %d, want 404", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "/no-such-page") {
		t.Errorf("the 404 page does not name the address that was asked for:\n%s", body)
	}
	if !strings.Contains(body, `href="/issues"`) {
		t.Errorf("the 404 page is not drawn inside the shell:\n%s", body)
	}
}

// A slug two issues share names neither of them, so the page offers the choice
// rather than guessing. Picking between them takes more than a pair of ids, so
// each candidate arrives with its title and the state it is in.
func TestDetailOnASharedSlugOffersTheMatches(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	first := create(t, svc, core.Draft{Title: "Fix flag parsing"})
	second := create(t, svc, core.Draft{Title: "Fix flag parsing"})
	start(t, svc, second.ID)

	h := newHandler(t, dir)
	res := get(h, "/issues/fix-flag-parsing")

	body := res.Body.String()
	rows := matchRows(t, body)
	for _, iss := range []issue.Issue{first, fetch(t, svc, second.ID)} {
		if link := `href="/issues/` + iss.ID + `"`; !strings.Contains(body, link) {
			t.Errorf("shared slug page does not link %s by ID:\n%s", iss.ID, body)
		}
		row, ok := rows[iss.ID]
		if !ok {
			t.Errorf("shared slug page offers no row for %s:\n%s", iss.ID, body)
			continue
		}
		for _, want := range []string{iss.Title, string(iss.State)} {
			if !strings.Contains(row, want) {
				t.Errorf("the row for %s does not say %q: %s", iss.ID, want, row)
			}
		}
	}
	// Searching the shared slug is still a reference, so it lands on the same
	// choice rather than on a text filter.
	if to := get(h, "/search?q=fix-flag-parsing").Header().Get("Location"); to != "/issues/fix-flag-parsing" {
		t.Errorf("searching the shared slug went to %q, want the disambiguation page", to)
	}
}

var (
	matchRow    = regexp.MustCompile(`(?s)<li\b[^>]*>(.*?)</li>`)
	matchRowRef = regexp.MustCompile(`href="/issues/([^"/]+)"`)
)

// matchRows reads the disambiguation page back as what it offers: one row per
// candidate, keyed by the id that row leads to.
func matchRows(t *testing.T, body string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, m := range matchRow.FindAllStringSubmatch(body, -1) {
		if ref := matchRowRef.FindStringSubmatch(m[1]); ref != nil {
			out[ref[1]] = m[1]
		}
	}
	return out
}

// The one search box: a reference goes to the issue, anything else filters the
// list; the reader never picks a mode.
func TestSearchJumpsToAnIssueForAReference(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, q := range []string{target.ID, "fix-flag-parsing"} {
		res := get(h, "/search?q="+q)
		if res.Code != http.StatusSeeOther {
			t.Errorf("GET /search?q=%s = %d, want 303", q, res.Code)
			continue
		}
		if want := "/issues/" + target.ID; res.Header().Get("Location") != want {
			t.Errorf("GET /search?q=%s went to %q, want %q", q, res.Header().Get("Location"), want)
		}
	}
}

func TestSearchWithoutAMatchFiltersTheList(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	target := create(t, svc, core.Draft{Title: "Fix flag parsing"})
	other := create(t, svc, core.Draft{Title: "Something else entirely"})
	h := newHandler(t, dir)

	res := get(h, "/search?q=parsing")

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 — an unmatched search is a filter, not an error", res.Code)
	}
	to := res.Header().Get("Location")
	if want := "/issues?search=parsing"; to != want {
		t.Fatalf("went to %q, want %q", to, want)
	}
	body := get(h, to).Body.String()
	if !strings.Contains(body, target.ID) {
		t.Errorf("filtered list is missing %s (%q):\n%s", target.ID, target.Title, body)
	}
	if strings.Contains(body, other.ID) {
		t.Errorf("filtered list still holds %s (%q), which does not match", other.ID, other.Title)
	}
}

// The search box is on every page, wherever the reader happens to be.
func TestSearchBoxAppearsOnEveryPage(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, path := range []string{"/", "/issues", "/issues/" + target.ID, "/nope"} {
		body := get(h, path).Body.String()
		if !strings.Contains(body, `action="/search"`) || !strings.Contains(body, `name="q"`) {
			t.Errorf("%s has no search box:\n%s", path, body)
		}
	}
}

// newStore returns a project directory holding a fresh, empty store.
func newStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, _, err := core.Init(dir); err != nil {
		t.Fatalf("init store: %v", err)
	}
	return dir
}

// newStoreNamed initializes a store inside a directory of a chosen name, the
// name being what the project is called, for the pages that have to say it.
func newStoreNamed(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make project directory: %v", err)
	}
	if _, _, err := core.Init(dir); err != nil {
		t.Fatalf("init store: %v", err)
	}
	return dir
}

func open(t *testing.T, dir string, opts ...core.Option) *core.Service {
	t.Helper()
	svc, err := core.Open(dir, opts...)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return svc
}

// testClock is a clock a test moves by hand, so an issue can be written at any
// instant the board's window has to reason about.
type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

func create(t *testing.T, svc *core.Service, d core.Draft) issue.Issue {
	t.Helper()
	created, err := svc.Create(d)
	if err != nil {
		t.Fatalf("create %q: %v", d.Title, err)
	}
	return created.Issue
}

func start(t *testing.T, svc *core.Service, ref string) {
	t.Helper()
	if _, err := svc.Start(ref, "tester", false); err != nil {
		t.Fatalf("start %s: %v", ref, err)
	}
}

func transition(t *testing.T, svc *core.Service, ref string, to issue.State) {
	t.Helper()
	if _, err := svc.Transition(ref, to); err != nil {
		t.Fatalf("move %s to %s: %v", ref, to, err)
	}
}

func note(t *testing.T, svc *core.Service, ref, text string) {
	t.Helper()
	if _, err := svc.Note(ref, "tester", text); err != nil {
		t.Fatalf("note %s: %v", ref, err)
	}
}

// addCustomField splices a user-defined frontmatter key into an issue file, the
// way a hand-edit would, since the core has no way to write one.
func addCustomField(t *testing.T, dir, id, key, value string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".beaver", "issues", id+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("find file for %s: %v (matched %v)", id, err, matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	edited := strings.Replace(string(raw), "---\n", "---\n"+key+": "+value+"\n", 1)
	if err := os.WriteFile(matches[0], []byte(edited), 0o644); err != nil {
		t.Fatalf("write %s: %v", matches[0], err)
	}
}

func newHandler(t *testing.T, dir string, opts ...core.Option) http.Handler {
	t.Helper()
	h, err := web.New(web.Config{WorkDir: dir, Actor: "tester", CoreOptions: opts})
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	return h
}

var (
	columnMark = regexp.MustCompile(`data-column="([a-z-]+)"`)
	cardMark   = regexp.MustCompile(`data-issue="([a-z0-9]+)"`)
)

// board reads the rendered page back as the structure it depicts: the columns
// in the order they appear, and the issue IDs carded in each.
func board(t *testing.T, body string) (order []string, cards map[string][]string) {
	t.Helper()
	marks := columnMark.FindAllStringSubmatchIndex(body, -1)
	if len(marks) == 0 {
		t.Fatalf("no columns on the page:\n%s", body)
	}
	cards = map[string][]string{}
	for i, m := range marks {
		state := body[m[2]:m[3]]
		end := len(body)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		order = append(order, state)
		for _, c := range cardMark.FindAllStringSubmatch(body[m[1]:end], -1) {
			cards[state] = append(cards[state], c[1])
		}
	}
	return order, cards
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
	return res
}
