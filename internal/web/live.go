package web

// This file holds liveness: the store is polled for a change and every open
// page is told, once, that something happened. Nothing about what changed
// travels — the page re-fetches its own view, so the answer is a fresh render
// of the files rather than a patch the server had to reason about.
//
// Polling beats watching the filesystem here: a fingerprint of the issues
// directory costs one directory read, works the same over a git checkout, an
// editor's atomic rename, or a network share, and needs no dependency
// (ADR 0006).

import (
	"net/http"
	"sync"
	"time"
)

// defaultPollInterval is how often an unattended store is fingerprinted. About
// a second is under the threshold where a reader would reach for the reload
// button, and cheap enough to leave running all day.
const defaultPollInterval = time.Second

// changes broadcasts one signal — the store is not what it was — to every
// connected reader. It holds no issue data and no history: a subscriber that
// misses nothing learns the same thing as one that missed ten changes, which is
// to go and look again.
//
// The poller runs only while someone is listening: the first subscriber starts
// it, the last one to leave stops it. A server nobody has a page open on does
// no work at all, and a test that connects to nothing leaves no goroutine
// behind.
type changes struct {
	interval time.Duration
	// read reports the store's current fingerprint. A failure — the store was
	// deleted mid-session, a directory turned unreadable — is not a change:
	// the last good fingerprint stands until the store answers again.
	read func() (string, error)

	mu   sync.Mutex
	subs map[chan struct{}]struct{}
	stop chan struct{} // closed to end the poller; nil while none is running
}

func newChanges(interval time.Duration, read func() (string, error)) *changes {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	return &changes{interval: interval, read: read, subs: map[chan struct{}]struct{}{}}
}

// subscribe registers a listener and returns it with the function that removes
// it. The channel holds one pending signal: a reader still redrawing from the
// last change does not need to be told twice that it is behind.
func (c *changes) subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subs[ch] = struct{}{}
	if c.stop == nil {
		// The fingerprint is taken here, before the first subscriber is told
		// anything, so the poller's first comparison is against the store as it
		// stood when the page was opened.
		stop := make(chan struct{})
		c.stop = stop
		last, _ := c.read()
		go c.poll(last, stop)
	}
	return ch, func() { c.unsubscribe(ch) }
}

func (c *changes) unsubscribe(ch chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subs, ch)
	if len(c.subs) == 0 && c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
}

// poll compares fingerprints on the interval and announces every difference.
// An unchanged store announces nothing — silence is the signal that the page on
// screen is still the truth.
func (c *changes) poll(last string, stop <-chan struct{}) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			current, err := c.read()
			if err != nil || current == last {
				continue
			}
			last = current
			c.broadcast()
		}
	}
}

func (c *changes) broadcast() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for ch := range c.subs {
		select {
		case ch <- struct{}{}:
		default: // already told, and not yet caught up
		}
	}
}

// events is the change feed as server-sent events: one `changed` event per
// store change, no payload, for as long as the page stays open. The connection
// is the subscription — closing the page unsubscribes it, and the last page to
// close stops the polling.
func (s *server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	signals, done := s.changes.subscribe()
	defer done()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	// An opening comment, sent once: it is not an event, so nothing redraws
	// over it, but it settles the connection through any buffering between here
	// and the browser.
	_, _ = w.Write([]byte(": watching the store\n\n"))
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-signals:
			// A `data` line is what makes this a dispatched event rather than a
			// comment; its content is deliberately nothing, because the page
			// asks the server what it now looks like.
			if _, err := w.Write([]byte("event: changed\ndata:\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
