package web_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/web"
)

// pollInterval is short enough that a test waits milliseconds for a change the
// reader would wait about a second for.
const pollInterval = 10 * time.Millisecond

// eventWait is how long a test gives a change to travel — many poll intervals,
// so a loaded machine slows the test down rather than failing it.
const eventWait = 5 * time.Second

// The point of the whole slice: a write nobody in the browser made — the CLI,
// an agent, a git pull — reaches the open page as one event.
func TestAStoreWriteAnnouncesItselfAsAChangedEvent(t *testing.T) {
	dir := newStore(t)
	srv := liveServer(t, dir)
	events := subscribe(t, srv.URL)

	create(t, open(t, dir), core.Draft{Title: "Written by someone else"})

	if got := next(t, events); got != "changed" {
		t.Errorf("event = %q, want changed", got)
	}
}

// One poll, one broadcast: every page open at the time hears about the change,
// not just whichever connected first.
func TestEveryConnectedClientHearsTheSameChange(t *testing.T) {
	dir := newStore(t)
	srv := liveServer(t, dir)
	first := subscribe(t, srv.URL)
	second := subscribe(t, srv.URL)

	create(t, open(t, dir), core.Draft{Title: "Written by someone else"})

	for i, events := range []<-chan string{first, second} {
		if got := next(t, events); got != "changed" {
			t.Errorf("subscriber %d got %q, want changed", i+1, got)
		}
	}
}

// A quiet store is silent: no heartbeat, no periodic re-render, nothing for a
// page to redraw itself over.
func TestAQuietStoreSendsNothing(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Settled long ago"})
	srv := liveServer(t, dir)
	events := subscribe(t, srv.URL)

	select {
	case got := <-events:
		t.Errorf("an untouched store sent %q, want silence", got)
	case <-time.After(20 * pollInterval):
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

// liveServer runs the handler over a real socket, since server-sent events are
// a streaming response rather than a recorded one.
func liveServer(t *testing.T, dir string) *httptest.Server {
	t.Helper()
	h, err := web.New(web.Config{WorkDir: dir, Actor: "tester", PollInterval: pollInterval})
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// subscribe opens the event stream and returns the event names arriving on it.
// It waits for the server's opening line before returning, so a write made by
// the test after this call is a change the poller has not already seen.
func subscribe(t *testing.T, base string) <-chan string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /events = %d, want 200", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	lines := bufio.NewScanner(res.Body)
	if !lines.Scan() {
		t.Fatalf("the stream said nothing to confirm it was open: %v", lines.Err())
	}
	events := make(chan string, 8)
	go func() {
		defer close(events)
		for lines.Scan() {
			if name, ok := strings.CutPrefix(lines.Text(), "event:"); ok {
				events <- strings.TrimSpace(name)
			}
		}
	}()
	return events
}

func next(t *testing.T, events <-chan string) string {
	t.Helper()
	select {
	case got, ok := <-events:
		if !ok {
			t.Fatal("the event stream closed before any event arrived")
		}
		return got
	case <-time.After(eventWait):
		t.Fatal("no event arrived before the timeout")
		return ""
	}
}
