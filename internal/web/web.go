// Package web is the local web interface over Beaver Backlog's core — the
// second interface after the CLI, and no more privileged than it. It turns an
// HTTP request into a call on the core and renders the result as HTML; it
// decides nothing about the rules, and it never reaches past the core to the
// store or a file.
//
// Pages are server-rendered html/template with every template and static asset
// embedded in the binary, so serving needs no build step and no network
// (ADR 0006). A core service is opened per request — a scan is cheap and the
// files change underneath the browser — so no issue data outlives a response.
package web

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

//go:embed templates/*.html
var templateFS embed.FS

// assets holds the stylesheet and htmx 2.0.4 (vendored, pinned, unmodified from
// unpkg; BSD-0). htmx earns its place because fragment refresh, inline form
// posts, and SSE-triggered updates become declarative attributes on
// server-rendered HTML instead of hand-written fetch-and-swap JavaScript.
//
//go:embed assets
var assetFS embed.FS

// Config is what the launching interface hands the web module: where the store
// is, who the writes belong to, and the seams the core reads from.
type Config struct {
	WorkDir     string        // the store is resolved from here, walking up, via the core
	Actor       string        // launch-resolved; attributed to every write
	CoreOptions []core.Option // clock and ID source travel to the core, not as Config fields
}

// New builds the handler serving the store above cfg.WorkDir. It returns
// core.ErrNoStore when there is none, so a caller can refuse to start a server
// over nothing.
func New(cfg Config) (http.Handler, error) {
	if _, err := core.Open(cfg.WorkDir, cfg.CoreOptions...); err != nil {
		return nil, err
	}
	s := &server{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.board)
	mux.HandleFunc("GET /issues", s.list)
	mux.HandleFunc("GET /assets/{path...}", s.asset)
	// ServeMux's bare "/" is the fallback for everything no other pattern
	// claimed, which is what makes an unknown path this interface's own 404 page
	// rather than the net/http default.
	mux.HandleFunc("/", s.notFound)
	return mux, nil
}

// server holds only what every request needs to open the application afresh —
// deliberately no issue data, which would be stale the moment a file changed.
type server struct{ cfg Config }

// open builds the core service one request works through. Every handler starts
// here, so a store that appeared, vanished, or changed since the last request is
// simply what this request sees.
func (s *server) open() (*core.Service, error) {
	return core.Open(s.cfg.WorkDir, s.cfg.CoreOptions...)
}

// board renders the whole backlog as four columns of cards — the home view, and
// read-only here: the cards are links, not yet handles to drag.
func (s *server) board(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	listing, err := svc.List(core.Query{})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, "board.html", http.StatusOK, boardPage{
		page:    s.page("Board", listing.Warnings),
		Columns: columns(listing.Issues, svc.Now(), r.URL),
	})
}

// list renders every issue in the core's ordering — the skeleton's one real
// read view, unfiltered.
func (s *server) list(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	listing, err := svc.List(core.Query{})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, "list.html", http.StatusOK, listPage{
		page:   s.page("Issues", listing.Warnings),
		Issues: listing.Issues,
	})
}

// asset serves an embedded static file. A missing one is an unknown path like
// any other, so it gets the same 404 page rather than net/http's bare text.
func (s *server) asset(w http.ResponseWriter, r *http.Request) {
	name := path(r)
	if _, err := fs.Stat(assetFS, name); err != nil {
		s.notFound(w, r)
		return
	}
	http.ServeFileFS(w, r, assetFS, name)
}

func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "error.html", http.StatusNotFound, errorPage{
		page:    s.page("Not found", nil),
		Message: "No page at " + path(r) + ".",
	})
}

// fail words a core failure as a page. This slice serves reads only, so the one
// distinction that matters is "the store or issue isn't there" against anything
// else; the spec's full error table arrives with the routes that need it.
func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, core.ErrNoStore) {
		s.render(w, r, "error.html", http.StatusNotFound, errorPage{
			page:    s.page("No store", nil),
			Message: "No Beaver Backlog store here any more — it may return with the next checkout.",
		})
		return
	}
	if errors.Is(err, core.ErrNotFound) {
		s.notFound(w, r)
		return
	}
	s.render(w, r, "error.html", http.StatusInternalServerError, errorPage{
		page:    s.page("Error", nil),
		Message: err.Error(),
	})
}

// page is the shell every view fills: what the browser titles the tab, and the
// files the scan skipped. Warnings ride on the page itself because a broken file
// must never cost the reader the rest of the store (ADR 0003).
type page struct {
	Title    string
	Warnings []skipped
}

// skipped is one invalid file named for a reader: its path relative to where the
// server was launched, and what is wrong with it.
type skipped struct {
	Path   string
	Reason string
}

type listPage struct {
	page
	Issues []issue.Issue
}

type errorPage struct {
	page
	Message string
}

func (s *server) page(title string, warnings []core.Warning) page {
	p := page{Title: title}
	for _, w := range warnings {
		p.Warnings = append(p.Warnings, skipped{Path: s.relPath(w.Path), Reason: w.Err.Error()})
	}
	return p
}

// relPath renders path relative to the directory the server was launched from
// when it sits inside it, falling back to the absolute path otherwise.
func (s *server) relPath(p string) string {
	if rel, err := filepath.Rel(s.cfg.WorkDir, p); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return p
}

// render writes one page, building it in full before any of it reaches the
// browser so a template failure cannot leave a half-drawn page behind a 200.
func (s *server) render(w http.ResponseWriter, r *http.Request, name string, status int, data any) {
	var buf bytes.Buffer
	if err := pages[name].ExecuteTemplate(&buf, "layout.html", data); err != nil {
		http.Error(w, "rendering "+name+": "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// path is the request's path without its leading slash — the form the embedded
// asset filesystem and the 404 message both want.
func path(r *http.Request) string { return strings.TrimPrefix(r.URL.Path, "/") }

// pages holds each view already parsed with the shared layout. Parsing at
// startup means a broken template fails the build's tests, never one request.
var pages = map[string]*template.Template{
	"board.html": mustParse("board.html"),
	"list.html":  mustParse("list.html"),
	"error.html": mustParse("error.html"),
}

func mustParse(name string) *template.Template {
	return template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/"+name))
}
