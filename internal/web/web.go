// Package web is the local web interface over Beaver Backlog's core — the
// second interface after the CLI, and no more privileged than it. It turns an
// HTTP request into a call on the core and renders the result as HTML; it
// decides nothing about the rules, and it never reaches past the core to the
// store or a file.
//
// Pages are server-rendered html/template with every template and static asset
// embedded in the binary, so serving needs no build step and no network
// (ADR 0006). The design system's stylesheet is compiled from styles/ by a
// pinned Tailwind CLI at dev time and its output committed, so building needs
// nothing beyond Go either. A core service is opened per request — a scan is
// cheap and the files change underneath the browser — so no issue data outlives
// a response.
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

// templateFS holds the layout shell, one file per page under pages/, and the
// fragments they share under partials/.
//
//go:embed templates/layout.html templates/pages/*.html templates/partials/*.html
var templateFS embed.FS

// assets holds the stylesheets and htmx 2.0.4 (vendored, pinned, unmodified from
// unpkg; BSD-0). htmx earns its place because fragment refresh and inline form
// posts become declarative attributes on server-rendered HTML instead of
// hand-written fetch-and-swap JavaScript.
//
// tailwind.css is generated — the committed output of scripts/build-css.sh over
// styles/tailwind.css. Edit the source, never this copy.
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
	mux.HandleFunc("GET /changed", s.changed)
	mux.HandleFunc("GET /assets/{path...}", s.asset)
	// ServeMux's bare "/" is the fallback for everything no other pattern
	// claimed, which is what makes an unknown path this interface's own 404 page
	// rather than the net/http default.
	mux.HandleFunc("/", s.notFound)
	return mux, nil
}

// server holds only what every request needs to open the application afresh —
// deliberately no issue data, which would be stale the moment a file changed.
type server struct {
	cfg Config
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
	p.Section = "board"
	if id := r.URL.Query().Get("deleted"); id != "" {
		p.Notice = "Deleted issue " + id + "."
	}
	s.render(w, r, "board.html", http.StatusOK, boardPage{
		page:    p,
		Filters: f.bar("/", r.URL.Query(), refused),
		Columns: columns(listing.Issues, s.relations(svc), svc.Now(), r.URL),
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
	p.Section = "issues"
	// The sidebar's box and the bar's text field are one filter, so a list
	// reached by searching says what it was searched for in both places.
	p.Search = f.Search
	order := parseOrder(r.URL.Query())
	order.apply(listing.Issues)
	s.render(w, r, "list.html", http.StatusOK, listPage{
		page:    p,
		Filters: f.bar("/issues", r.URL.Query(), refused),
		Rows:    rows(listing.Issues, s.relations(svc)),
		Columns: order.headers(r.URL),
	})
}

// relations is the derived-condition index over the whole store, not the
// filtered view, so a card's "blocked" never depends on whether its blocker
// made it past the filter bar. A store that cannot answer yields nil, which
// issue.Relations treats as an index over nothing — the marks simply stay off,
// never costing the page (ADR 0003).
func (s *server) relations(svc *core.Service) *issue.Relations {
	all, err := svc.List(core.Query{})
	if err != nil {
		return issue.NewRelations(nil)
	}
	return issue.NewRelations(all.Issues)
}

// rows pairs each listed issue with the derived conditions its row shows.
func rows(issues []issue.Issue, rel *issue.Relations) []row {
	out := make([]row, len(issues))
	for i, iss := range issues {
		out[i] = row{Issue: iss, Conditions: conditions(iss, rel)}
	}
	return out
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
			Message: "No Beaver Backlog store here any more; it may return with the next checkout.",
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
	// Section names the sidebar entry this page belongs under, so the nav can
	// say where the reader is; empty on a page that is nowhere in particular,
	// like a form or an error.
	Section string
	// Search is what the sidebar's box shows, so a filtered list still says
	// what it was filtered by; empty everywhere the reader has not searched.
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

// navItem is one entry in the shell's sidebar navigation: where it goes, what
// it says, whether the reader is already there, and the count it wears — only
// Doctor wears one, and only when there is something to count.
type navItem struct {
	Href    string
	Label   string
	Current bool
	Badge   int
}

// Nav is the sidebar's navigation, the same four views from every page. It is a
// method on page rather than a field each handler fills, so a view that says
// which section it belongs to has said everything the shell needs.
//
// Doctor carries the number of files the scan skipped, which makes store health
// an ambient fact rather than a page to remember; a page that did not scan
// simply shows no badge.
func (p page) Nav() []navItem {
	items := []navItem{
		{Href: "/", Label: "Board"},
		{Href: "/issues", Label: "Issues"},
		{Href: "/graph", Label: "Graph"},
		{Href: "/doctor", Label: "Doctor", Badge: len(p.Warnings)},
	}
	for i, section := range []string{"board", "issues", "graph", "doctor"} {
		items[i].Current = section == p.Section
	}
	return items
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
	Rows    []row
	Columns []header
}

// row is one issue in the table with the derived conditions its row shows.
type row struct {
	Issue      issue.Issue
	Conditions conditionMarks
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

// mustParse builds one page's template set: the layout shell, every shared
// partial, and the page itself. Each page gets all the partials rather than a
// curated list, so using a fragment on a new page is a template call away.
func mustParse(name string) *template.Template {
	return template.Must(template.ParseFS(templateFS,
		"templates/layout.html", "templates/partials/*.html", "templates/pages/"+name))
}
