package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// The point of the whole slice: a write nobody in the browser made — the CLI,
// an agent, a git pull — changes the answer at /changed, which is how an open
// page learns to re-fetch its view.
func TestAStoreWriteChangesTheAnswerAtChanged(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	before := etag(t, get(h, "/changed"))

	create(t, open(t, dir), core.Draft{Title: "Written by someone else"})

	res := poll(h, before)
	if res.Code != http.StatusOK {
		t.Fatalf("GET /changed after a write = %d, want 200", res.Code)
	}
	if after := etag(t, res); after == before {
		t.Errorf("ETag %q did not change with the store", after)
	}
}

// A quiet store answers 304: the page on screen is still the truth, and the
// poll costs a validator comparison rather than a render.
func TestAQuietStoreAnswersNotModified(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Settled long ago"})
	h := newHandler(t, dir)

	res := poll(h, etag(t, get(h, "/changed")))
	if res.Code != http.StatusNotModified {
		t.Errorf("GET /changed on a quiet store = %d, want 304", res.Code)
	}
}

// A poll with nothing to compare — the page just opened — is answered with the
// current validator, never a redraw signal it would have no baseline for.
func TestTheFirstPollEstablishesABaseline(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)

	res := get(h, "/changed")
	if res.Code != http.StatusOK {
		t.Fatalf("GET /changed = %d, want 200", res.Code)
	}
	if etag(t, res) == "" {
		t.Error("the first poll carried no ETag to compare the next one against")
	}
}

// The regression this file guards (rpliqf): liveness must not hold a
// connection open. Browsers cap HTTP/1.1 at six connections per origin, so six
// open tabs each holding an event stream starved every click and drag. There
// is no stream to hold any more.
func TestNoEventStreamRemainsToHoldAConnection(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	if res := get(h, "/events"); res.Code != http.StatusNotFound {
		t.Errorf("GET /events = %d, want 404 — a held stream starves the browser's connection pool", res.Code)
	}
}

// The read views carry the mark the live script refreshes, and the forms do
// not — a page being typed into must never be redrawn under the typist.
func TestOnlyReadViewsAreMarkedLive(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	for _, path := range []string{"/", "/issues", "/issues/" + target.ID} {
		if body := get(h, path).Body.String(); !strings.Contains(body, "data-live") {
			t.Errorf("read view %s is not marked live:\n%s", path, body)
		}
	}
	for _, path := range []string{"/issues/new", "/issues/" + target.ID + "/edit"} {
		if body := get(h, path).Body.String(); strings.Contains(body, "data-live") {
			t.Errorf("form page %s is marked live; forms never refresh under the user", path)
		}
	}
}

// The listener ships with the page — every view loads it, and it is served.
func TestEveryPageLoadsTheLiveListener(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	for _, path := range []string{"/", "/issues", "/nope"} {
		if body := get(h, path).Body.String(); !strings.Contains(body, "/assets/live.js") {
			t.Errorf("%s does not load the live listener", path)
		}
	}
	if code := get(h, "/assets/live.js").Code; code != http.StatusOK {
		t.Errorf("GET /assets/live.js = %d, want 200", code)
	}
}

// poll asks /changed the way live.js does: with the validator a previous
// answer handed back.
func poll(h http.Handler, validator string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/changed", nil)
	req.Header.Set("If-None-Match", validator)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

// etag digs the validator out of an answer, failing the test over an answer
// that carries none.
func etag(t *testing.T, res *httptest.ResponseRecorder) string {
	t.Helper()
	if res.Code == http.StatusOK {
		if got := res.Header().Get("ETag"); got != "" {
			return got
		}
		t.Fatal("a 200 from /changed carried no ETag")
	}
	return res.Header().Get("ETag")
}
