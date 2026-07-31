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
	"time"

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
	// PollInterval is how often the store is fingerprinted for the live view.
	// It belongs to the interface rather than to the core — how often a browser
	// is told to look again is a property of this UI — and zero means the
	// default second. Tests shorten it.
	PollInterval time.Duration
}

// New builds the handler serving the store above cfg.WorkDir. It returns
// core.ErrNoStore when there is none, so a caller can refuse to start a server
// over nothing.
func New(cfg Config) (http.Handler, error) {
	if _, err := core.Open(cfg.WorkDir, cfg.CoreOptions...); err != nil {
		return nil, err
	}
	s := &server{cfg: cfg}
	s.changes = newChanges(cfg.PollInterval, s.fingerprint)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.board)
	mux.HandleFunc("GET /issues", s.list)
	mux.HandleFunc("GET /graph", s.graph)
	mux.HandleFunc("GET /issues/new", s.createFormPage)
	mux.HandleFunc("POST /issues", s.create)
	mux.HandleFunc("GET /issues/{ref}", s.detail)
	mux.HandleFunc("GET /issues/{ref}/edit", s.editFormPage)
	mux.HandleFunc("POST /issues/{ref}", s.update)
	mux.HandleFunc("POST /issues/{ref}/state", s.setState)
	mux.HandleFunc("POST /issues/{ref}/start", s.start)
	mux.HandleFunc("POST /issues/{ref}/notes", s.addNote)
	mux.HandleFunc("POST /issues/{ref}/delete", s.remove)
	mux.HandleFunc("GET /doctor", s.doctor)
	mux.HandleFunc("POST /doctor/fix", s.fix)
	mux.HandleFunc("GET /search", s.search)
	mux.HandleFunc("GET /events", s.events)
	mux.HandleFunc("GET /assets/{path...}", s.asset)
	// ServeMux's bare "/" is the fallback for everything no other pattern
	// claimed, which is what makes an unknown path this interface's own 404 page
	// rather than the net/http default.
	mux.HandleFunc("/", s.notFound)
	return mux, nil
}

// server holds only what every request needs to open the application afresh —
// deliberately no issue data, which would be stale the moment a file changed.
// The one long-lived piece is the change feed, which holds no issue data
// either: only whether the files still look the way they did.
type server struct {
	cfg     Config
	changes *changes
}

// fingerprint is what the poller compares between ticks. It opens the store
// afresh like every other read, so a store that vanished and came back is
// simply a change like any other.
func (s *server) fingerprint() (string, error) {
	svc, err := s.open()
	if err != nil {
		return "", err
	}
	return svc.Fingerprint()
}

// open builds the core service one request works through. Every handler starts
// here, so a store that appeared, vanished, or changed since the last request is
// simply what this request sees.
func (s *server) open() (*core.Service, error) {
	return core.Open(s.cfg.WorkDir, s.cfg.CoreOptions...)
}

// board renders the whole backlog as four columns of cards — the home view,
// where a card is both a link to its issue and a handle to drag between states.
func (s *server) board(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	f := parseFilters(r.URL.Query())
	listing, refused, err := s.filtered(svc, f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p := s.page("Board", listing.Warnings)
	p.Live = true
	if id := r.URL.Query().Get("deleted"); id != "" {
		p.Notice = "Deleted issue " + id + "."
	}
	s.render(w, r, "board.html", http.StatusOK, boardPage{
		page:    p,
		Filters: f.bar("/", r.URL.Query(), refused),
		Columns: columns(listing.Issues, svc.Now(), r.URL),
	})
}

// list renders the issues the address selects, in the core's ordering — the
// same filters the board reads, over a table instead of columns.
func (s *server) list(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	f := parseFilters(r.URL.Query())
	listing, refused, err := s.filtered(svc, f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p := s.page("Issues", listing.Warnings)
	p.Live = true
	// The header's box and the bar's text field are one filter, so a list
	// reached by searching says what it was searched for in both places.
	p.Search = f.Search
	s.render(w, r, "list.html", http.StatusOK, listPage{
		page:    p,
		Filters: f.bar("/issues", r.URL.Query(), refused),
		Issues:  listing.Issues,
	})
}

// filtered runs an address's query, telling a reference that names no issue
// apart from a failure of the request. A typo in the parent box is the reader's
// to fix where they typed it, so the core's words come back for the bar over an
// empty view rather than as an error page.
func (s *server) filtered(svc *core.Service, f filters) (core.Listing, string, error) {
	listing, err := svc.List(f.query())
	if err == nil {
		return listing, "", nil
	}
	var (
		unknown   *core.UnknownRefError
		ambiguous *core.AmbiguousRefError
	)
	if errors.As(err, &unknown) || errors.As(err, &ambiguous) {
		return listing, err.Error(), nil
	}
	return listing, "", err
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
	Title string
	// Search is what the header's box shows, so a filtered list still says what
	// it was filtered by; empty everywhere the reader has not searched.
	Search string
	// Notice is a one-line confirmation of something that already happened —
	// what a redirect after a write has to say once the page it wrote about is
	// gone. Empty on a page that is simply being read.
	Notice string
	// Live marks a page the change feed may redraw: a view being read, never a
	// form being filled in. A page the reader is typing into is theirs until
	// they submit it, whatever the store does meanwhile.
	Live     bool
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
	Filters filterBar
	Issues  []issue.Issue
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
	if err := pages[name].ExecuteTemplate(&buf, entry(r), data); err != nil {
		http.Error(w, "rendering "+name+": "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// entry is the template a request enters the page through: a fragment request —
// htmx's swap, or the live listener's redraw — asks for the view's own markup
// and puts it into a page it already has, so answering with the chrome around it
// would nest a second copy of the whole document.
func entry(r *http.Request) string {
	if r.Header.Get("HX-Request") == "true" {
		return "view"
	}
	return "layout.html"
}

// path is the request's path without its leading slash — the form the embedded
// asset filesystem and the 404 message both want.
func path(r *http.Request) string { return strings.TrimPrefix(r.URL.Path, "/") }

// pages holds each view already parsed with the shared layout. Parsing at
// startup means a broken template fails the build's tests, never one request.
var pages = map[string]*template.Template{
	"board.html":   mustParse("board.html"),
	"new.html":     mustParse("new.html"),
	"edit.html":    mustParse("edit.html"),
	"list.html":    mustParse("list.html"),
	"graph.html":   mustParse("graph.html"),
	"detail.html":  mustParse("detail.html"),
	"doctor.html":  mustParse("doctor.html"),
	"matches.html": mustParse("matches.html"),
	"error.html":   mustParse("error.html"),
}

func mustParse(name string) *template.Template {
	return template.Must(template.ParseFS(templateFS,
		"templates/layout.html", "templates/filters.html", "templates/"+name))
}
