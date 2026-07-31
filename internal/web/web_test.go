package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{"/assets/htmx.min.js", http.StatusOK},
		{"/assets/app.css", http.StatusOK},
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

	body := get(newHandler(t, dir), "/").Body.String()

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
	if body := get(h, "/").Body.String(); strings.Contains(body, "Written later") {
		t.Fatal("empty store already renders the later issue")
	}

	later := create(t, open(t, dir), core.Draft{Title: "Written later"})

	if body := get(h, "/").Body.String(); !strings.Contains(body, later.ID) {
		t.Error("issue created after the handler was built is missing from the list")
	}
}

func TestInvalidFileBecomesABannerNotAnError(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Perfectly fine"})
	broken := filepath.Join(dir, ".beaver", "issues", "broken.md")
	if err := os.WriteFile(broken, []byte("not an issue file"), 0o644); err != nil {
		t.Fatalf("seed invalid file: %v", err)
	}

	res := get(newHandler(t, dir), "/")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 despite the invalid file", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "broken.md") {
		t.Errorf("banner does not name the skipped file:\n%s", body)
	}
	if !strings.Contains(body, "Perfectly fine") {
		t.Error("the valid issue is missing; one broken file must not empty the page")
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

func open(t *testing.T, dir string) *core.Service {
	t.Helper()
	svc, err := core.Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return svc
}

func create(t *testing.T, svc *core.Service, d core.Draft) issue.Issue {
	t.Helper()
	created, err := svc.Create(d)
	if err != nil {
		t.Fatalf("create %q: %v", d.Title, err)
	}
	return created.Issue
}

func newHandler(t *testing.T, dir string) http.Handler {
	t.Helper()
	h, err := web.New(web.Config{WorkDir: dir, Actor: "tester"})
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	return h
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
	return res
}
