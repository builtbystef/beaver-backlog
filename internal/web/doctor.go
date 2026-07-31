package web

// This file holds the doctor page: the store's health report as a reader sees
// it, and the one repair that is mechanically safe. What is wrong, which class
// each wrong belongs to, and which of them a machine may fix are all the core's
// — this file only words the findings and renders the files they name.
//
// Unusable files are findings here rather than the banner they are everywhere
// else: the whole point of the scan is to report them, so a broken file is
// named once, in the view whose job it is to say so (ADR 0003).

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/core"
)

// doctorPage is one health scan rendered: the headline counts, every finding,
// and whether anything is left that a repair could take on.
type doctorPage struct {
	page
	Checked    int
	Problems   int
	Advisories int
	Fixable    int
	Fixed      int
	Findings   []findingView
	// Repaired marks the render that answers a repair, so the page can account
	// for what it just did rather than describe a store it only read.
	Repaired bool
}

// findingView is one finding worded for a reader: what class it belongs to,
// which files and issues it concerns, and the fact behind it.
type findingView struct {
	Category string // the core's machine name, which the markup carries
	Label    string // the class in words
	Class    string // "problem", "advisory", or "fixed" — how it reads on the page
	Paths    []string
	IDs      []string
	Detail   string
}

// doctor renders the store's health without touching it.
func (s *server) doctor(w http.ResponseWriter, r *http.Request) {
	s.scan(w, r, false)
}

// fix repairs what is mechanically safe and renders the report the repair
// produced, so the answer to pressing the button is an account of what it did.
func (s *server) fix(w http.ResponseWriter, r *http.Request) {
	s.scan(w, r, true)
}

// scan runs one health scan and renders it. The repaired page is deliberately
// not live: it is a receipt for something the reader just did, and a redraw
// from the change their own repair caused would take it away before they read
// it.
func (s *server) scan(w http.ResponseWriter, r *http.Request, repair bool) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// No warnings to carry: the scan hands unusable files back as findings, so
	// nothing is skipped silently and nothing is named twice.
	rep, err := svc.Doctor(repair)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p := s.page("Doctor", nil)
	p.Live = !repair
	s.render(w, r, "doctor.html", http.StatusOK, doctorPage{
		page:       p,
		Checked:    rep.Checked,
		Problems:   rep.Problems(),
		Advisories: rep.Advisories(),
		Fixable:    rep.Fixable(),
		Fixed:      rep.Fixed(),
		Findings:   s.findingViews(rep.Findings),
		Repaired:   repair,
	})
}

func (s *server) findingViews(findings []core.Finding) []findingView {
	out := make([]findingView, len(findings))
	for i, f := range findings {
		out[i] = findingView{
			Category: f.Category.String(),
			Label:    findingLabel(f.Category),
			Class:    findingClass(f),
			Paths:    s.relPaths(f.Paths),
			IDs:      f.IDs,
			Detail:   findingDetail(f),
		}
	}
	return out
}

// findingClass is how a finding reads on the page. A repaired finding is
// neither a problem nor a note but a receipt, so it gets its own class; the
// rest follow the core's classification, where resemblance to a known field is
// only a guess and so never makes a store read as unhealthy.
func findingClass(f core.Finding) string {
	switch {
	case f.Fixed:
		return "fixed"
	case f.Category.Advisory():
		return "advisory"
	default:
		return "problem"
	}
}

// findingLabel is the class of a finding in words — the badge the page leads
// each finding with.
func findingLabel(c core.Category) string {
	switch c {
	case core.CategoryInvalid:
		return "invalid file"
	case core.CategoryDuplicateID:
		return "duplicate id"
	case core.CategoryDependencyCycle:
		return "dependency cycle"
	case core.CategoryParentCycle:
		return "parent cycle"
	case core.CategoryDanglingRef:
		return "dangling reference"
	case core.CategoryStuck:
		return "stuck"
	case core.CategoryUnknownKey:
		return "likely typo"
	case core.CategoryUnknownValue:
		return "unknown value"
	case core.CategoryMissingTimestamp:
		return "missing timestamp"
	case core.CategoryFilenameDrift:
		return "filename drift"
	}
	return "problem"
}

// findingDetail states the fact a finding turns on, in this interface's words.
// The core hands back the anchors rather than a sentence, so the paths and ids
// are rendered where the page already shows them and only what is particular to
// the class is said here.
func findingDetail(f core.Finding) string {
	if f.Fixed {
		return "renamed to " + f.Canonical
	}
	switch f.Category {
	case core.CategoryInvalid:
		return fmt.Sprintf("not a usable issue: %v", f.Err)
	case core.CategoryDuplicateID:
		return fmt.Sprintf("%d files claim this id; only a human can say which keeps it", len(f.Paths))
	case core.CategoryDependencyCycle:
		return "these issues depend on each other in a loop, so none of them can ever be ready"
	case core.CategoryParentCycle:
		// An issue that is its own parent reads as nonsense described as a loop.
		if len(f.IDs) == 1 {
			return "is its own parent"
		}
		return "these issues are each other's parents"
	case core.CategoryDanglingRef:
		return fmt.Sprintf("%s names %s, but no issue holds that id", f.Field, f.Target)
	case core.CategoryStuck:
		return "waits on cancelled " + strings.Join(f.Cancelled, ", ") + ", which will never be done"
	case core.CategoryUnknownKey:
		return fmt.Sprintf("%q looks like a typo of %q", f.Key, f.Resembles)
	case core.CategoryUnknownValue:
		return fmt.Sprintf("priority %q is not a recognized level (urgent, high, medium, or low); no priority filter will match it", f.Value)
	case core.CategoryMissingTimestamp:
		return fmt.Sprintf("missing %s (an issue with no created time sorts as the oldest)", strings.Join(f.Missing, " and "))
	case core.CategoryFilenameDrift:
		return "should be named " + f.Canonical
	}
	return f.Category.String()
}

// relPaths renders every path the way the page names files: relative to where
// the server was launched.
func (s *server) relPaths(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = s.relPath(p)
	}
	return out
}
