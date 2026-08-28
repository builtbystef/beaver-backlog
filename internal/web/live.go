package web

// This file holds liveness: every open page asks, about once a second, whether
// the store still looks the way it did, and re-fetches its own view when the
// answer is no. Nothing about what changed travels: the page re-renders from
// the files rather than applying a patch the server had to reason about.
//
// The asking is a short client-side poll of one tiny endpoint, deliberately
// not a held stream. A browser allows only about six plain-HTTP connections
// per origin, so the previous design, a server-sent-events stream each tab
// held for its lifetime, starved every click and drag once six tabs were
// open (rpliqf). A poll is over in a millisecond, so no number of open tabs
// can occupy the pool. Polling the store also beats watching the filesystem:
// a fingerprint of the issues directory costs one directory read, works the
// same over a git checkout, an editor's atomic rename, or a network share,
// and needs no dependency (ADR 0006).

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// changed answers whether the store still looks the way the asker last saw it.
// The store's fingerprint travels as an ETag, and a poll carrying that ETag
// back in If-None-Match is answered 304: nothing to redraw. The comparison is
// this handler's own, between validators it minted itself, so caching in the
// browser is switched off rather than reasoned about.
func (s *server) changed(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err == nil {
		var fp string
		if fp, err = svc.Fingerprint(); err == nil {
			tag := validator(fp)
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("ETag", tag)
			if r.Header.Get("If-None-Match") == tag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	// A store that cannot answer, deleted mid-session or with a directory turned
	// unreadable, is an outage rather than a change: the page keeps its last
	// validator and shows the disconnected notice until the store answers
	// again. This is a machine endpoint, so the answer is a status, not a page.
	http.Error(w, err.Error(), http.StatusServiceUnavailable)
}

// validator folds a fingerprint into ETag form. The fingerprint itself is
// file names, sizes, and times around control characters, which are not legal
// in a header, so what travels is a digest, quoted as ETags are.
func validator(fingerprint string) string {
	sum := sha256.Sum256([]byte(fingerprint))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}
