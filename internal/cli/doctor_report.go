package cli

// This file holds doctor's report half: the finding and report types diagnosis
// produces, the --fix application over them, and the human and JSON renderings.
// The scanning that produces the findings lives in doctor.go.

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"beaver/internal/issue"
	"beaver/internal/output"
	"beaver/internal/store"
)

// category is the class of a health problem. The declaration order is also the
// report's severity order — invalid (a file Busy Beaver cannot use) first, filename
// drift (cosmetic, and the only auto-fixable class) last — so findings sort into a
// stable, most-serious-first list.
type category int

const (
	catInvalid category = iota
	catDuplicateID
	catCycle
	catParentCycle
	catDangling
	catStuck
	catUnknownKey
	catUnknownValue
	catNoTimestamp
	catDrift
)

// advisory reports whether findings of this class are informational rather than
// problems: still reported — a key resembling a known field is worth a look — but
// never counted toward the exit code or the ok flag. Unknown-key is advisory
// because resemblance is only ever a guess: a deliberate custom field like
// `status` sits within typo distance of `state`, and a first-class supported
// feature (ADR 0014) must not keep an otherwise healthy store permanently failing
// doctor. Every other class states a fact, not a guess, and stays a problem.
func (c category) advisory() bool { return c == catUnknownKey }

// slug is the stable machine name of a category, used in JSON so a consumer can
// branch on the problem class (ADR 0013).
func (c category) slug() string {
	switch c {
	case catInvalid:
		return "invalid"
	case catDuplicateID:
		return "duplicate_id"
	case catCycle:
		return "dependency_cycle"
	case catParentCycle:
		return "parent_cycle"
	case catDangling:
		return "dangling_reference"
	case catStuck:
		return "stuck"
	case catUnknownKey:
		return "unknown_key"
	case catUnknownValue:
		return "unknown_value"
	case catNoTimestamp:
		return "missing_timestamp"
	case catDrift:
		return "filename_drift"
	}
	return "unknown"
}

// label is the short human name of a category, the left column of the human report.
func (c category) label() string {
	switch c {
	case catInvalid:
		return "invalid"
	case catDuplicateID:
		return "duplicate id"
	case catCycle:
		return "cycle"
	case catParentCycle:
		return "parent cycle"
	case catDangling:
		return "dangling ref"
	case catStuck:
		return "stuck"
	case catUnknownKey:
		return "unknown key"
	case catUnknownValue:
		return "unknown value"
	case catNoTimestamp:
		return "missing time"
	case catDrift:
		return "filename drift"
	}
	return "problem"
}

// finding is one health problem. detail is a self-contained, human-readable summary
// (it embeds the file or ids it concerns), so it doubles as the JSON message and the
// human detail; file and ids expose the same anchors structurally for a machine
// consumer. Only filename drift is fixable, and only it carries the fix payload
// (want/fixSrc/fixIss); fixed flips to true once --fix has renamed it.
type finding struct {
	cat     category
	file    string   // relative path of the primary file, or "" when the finding spans files
	ids     []string // issue ids the finding concerns
	detail  string   // self-contained problem statement
	fixable bool
	fixed   bool

	// Filename-drift repair payload and display target.
	want   string      // canonical base name the file should have
	fixSrc string      // absolute path to rename
	fixIss issue.Issue // issue whose canonical name is the destination
}

// report is the whole health scan: how many valid issues were checked, and every
// finding, ordered most-serious-first.
type report struct {
	checked  int
	findings []finding
}

// remaining counts the problems still standing — every non-advisory finding not
// repaired this run — which is what the exit code and the "ok" flag turn on.
// Advisory findings are excluded by definition: they are reported, never held
// against the store's health.
func (r *report) remaining() int {
	n := 0
	for _, f := range r.findings {
		if !f.fixed && !f.cat.advisory() {
			n++
		}
	}
	return n
}

// advisoryCount counts the informational findings, reported apart from the
// problems in every summary.
func (r *report) advisoryCount() int {
	n := 0
	for _, f := range r.findings {
		if f.cat.advisory() {
			n++
		}
	}
	return n
}

func (r *report) fixedCount() int {
	n := 0
	for _, f := range r.findings {
		if f.fixed {
			n++
		}
	}
	return n
}

func (r *report) fixableCount() int {
	n := 0
	for _, f := range r.findings {
		if f.fixable && !f.fixed {
			n++
		}
	}
	return n
}

// applyFixes repairs the fixable findings — today, filename drift — by renaming each
// drifted file to its canonical name through the store, which refuses to overwrite
// another file (ErrNameCollision) so a fix never destroys data. It runs to a fixed
// point: a pass repairs every finding whose destination is currently free, and
// repeats while any pass makes progress, so a chain of drifts that free each other's
// names (A wants B's name, B wants C's) all resolve in one invocation. A finding
// whose destination stays occupied — the rare mutual swap — is simply left standing
// and reported, never forced.
func (r *report) applyFixes(st *store.Store) {
	for {
		progress := false
		for i := range r.findings {
			f := &r.findings[i]
			if !f.fixable || f.fixed {
				continue
			}
			newPath, err := st.Rename(f.fixSrc, f.fixIss)
			if err != nil {
				continue // destination taken (or a write error): leave it standing
			}
			f.fixed = true
			f.want = filepath.Base(newPath)
			progress = true
		}
		if !progress {
			return
		}
	}
}

// render writes the report in the resolved format.
func (r *report) render(w io.Writer, format output.Format, fix bool) error {
	if format == output.JSON {
		return r.renderJSON(w)
	}
	return r.renderHuman(w, fix)
}

// renderHuman writes the report as a human reads it: a headline, an aligned
// class/detail table of the findings (ones repaired this run marked "fixed"), and a
// closing summary that either points at --fix or accounts for what it repaired and
// what still needs a person.
func (r *report) renderHuman(w io.Writer, fix bool) error {
	if len(r.findings) == 0 {
		_, err := fmt.Fprintf(w, "No problems found (checked %d issue%s).\n", r.checked, plural(r.checked))
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %s (checked %d issue%s):\n\n", foundClause(len(r.findings)-r.advisoryCount(), r.advisoryCount()), r.checked, plural(r.checked))
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, f := range r.findings {
		label, detail := f.cat.label(), f.detail
		if f.fixed {
			label, detail = "fixed", "renamed to "+f.want
		}
		fmt.Fprintf(tw, "  %s\t%s\n", label, detail)
	}
	tw.Flush()

	b.WriteByte('\n')
	writeSummary(&b, r, fix)
	_, err := io.WriteString(w, b.String())
	return err
}

// foundClause words the headline's count: problems, advisory notes, or both. The
// two are kept apart so an advisory-only report does not read as an unhealthy
// store.
func foundClause(problems, advisories int) string {
	switch {
	case advisories == 0:
		return fmt.Sprintf("%d problem%s", problems, plural(problems))
	case problems == 0:
		return fmt.Sprintf("%d advisory note%s", advisories, plural(advisories))
	default:
		return fmt.Sprintf("%d problem%s and %d advisory note%s", problems, plural(problems), advisories, plural(advisories))
	}
}

// advisoryOnlyLine closes a report whose only findings are advisory: nothing is
// wrong, doctor just has something worth a look.
const advisoryOnlyLine = "Advisory notes are informational and do not fail doctor; the store is otherwise healthy."

// writeSummary writes the closing line: before --fix, an offer to repair the fixable
// findings; after --fix, an accounting of what was repaired and what still needs a
// human. A report of only advisory notes closes by saying so, in both modes, rather
// than talking about problems that are not there.
func writeSummary(b *strings.Builder, r *report, fix bool) {
	remaining := r.remaining()
	if fix {
		fixed := r.fixedCount()
		switch {
		case fixed == 0 && remaining == 0:
			fmt.Fprintln(b, advisoryOnlyLine)
		case fixed == 0:
			fmt.Fprintf(b, "Nothing here could be fixed automatically; %d problem%s need%s a human.\n", remaining, plural(remaining), verbS(remaining))
		case remaining == 0:
			fmt.Fprintf(b, "Fixed %d problem%s; the store is clean.\n", fixed, plural(fixed))
		default:
			fmt.Fprintf(b, "Fixed %d problem%s; %d problem%s still need%s a human.\n", fixed, plural(fixed), remaining, plural(remaining), verbS(remaining))
		}
		return
	}
	if n := r.fixableCount(); n > 0 {
		fmt.Fprintf(b, "%d of these can be fixed automatically — run `beaver doctor --fix`.\n", n)
		return
	}
	if remaining == 0 {
		fmt.Fprintln(b, advisoryOnlyLine)
		return
	}
	fmt.Fprintln(b, "None of these can be fixed automatically; each needs a human.")
}

// reportView is the report's stable JSON shape (ADR 0013): the summary counts, an
// ok flag a consumer can read instead of the exit code, and every finding. ok is
// true exactly when no problems remain.
type reportView struct {
	OK       bool          `json:"ok"`
	Checked  int           `json:"checked"`
	Problems int           `json:"problems"`
	Fixed    int           `json:"fixed"`
	Findings []findingView `json:"findings"`
}

// findingView is one finding in JSON. Every key is always present (file is null when
// the finding spans files; ids is [] never null), so a consumer never special-cases
// a missing key (ADR 0013). advisory marks the informational classes a consumer may
// filter out: an advisory finding never counts toward problems or flips ok.
type findingView struct {
	Category string   `json:"category"`
	File     *string  `json:"file"`
	IDs      []string `json:"ids"`
	Message  string   `json:"message"`
	Fixable  bool     `json:"fixable"`
	Fixed    bool     `json:"fixed"`
	Advisory bool     `json:"advisory"`
}

func (r *report) renderJSON(w io.Writer) error {
	views := make([]findingView, len(r.findings))
	for i, f := range r.findings {
		views[i] = findingView{
			Category: f.cat.slug(),
			File:     nilIfEmpty(f.file),
			IDs:      strsOrEmpty(f.ids),
			Message:  f.detail,
			Fixable:  f.fixable,
			Fixed:    f.fixed,
			Advisory: f.cat.advisory(),
		}
	}
	return output.WriteJSON(w, reportView{
		OK:       r.remaining() == 0,
		Checked:  r.checked,
		Problems: r.remaining(),
		Fixed:    r.fixedCount(),
		Findings: views,
	})
}

// sortFindings orders findings most-serious-first (by category), then by file, then
// by first id, then detail — a total order, so the report is byte-for-byte
// deterministic regardless of file iteration order.
func sortFindings(fs []finding) {
	sort.Slice(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.cat != b.cat {
			return a.cat < b.cat
		}
		if a.file != b.file {
			return a.file < b.file
		}
		if ai, bi := firstID(a.ids), firstID(b.ids); ai != bi {
			return ai < bi
		}
		return a.detail < b.detail
	})
}

// --- small helpers ---

func firstID(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strsOrEmpty(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// plural returns the plural suffix for a count, so a message can read "1 issue" and
// "2 issues" from one format string.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// verbS returns the present-tense verb suffix agreeing with a count ("1 problem
// needs" vs "2 problems need").
func verbS(n int) string {
	if n == 1 {
		return "s"
	}
	return ""
}
