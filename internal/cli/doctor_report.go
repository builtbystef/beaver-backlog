package cli

// This file holds doctor's report half: the finding and report types, the --fix
// application, and the human and JSON renderings. The scanning lives in doctor.go.

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

// category is the class of a health problem. Declaration order is the report's
// severity order, most serious first, so findings sort into a stable list.
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

// advisory reports whether findings of this class are informational: reported, but
// never counted toward the exit code or the ok flag. Unknown-key is advisory
// because resemblance is only a guess — a deliberate custom field like `status`
// sits within typo distance of `state` and must not fail an otherwise healthy
// store.
func (c category) advisory() bool { return c == catUnknownKey }

// slug is the stable machine name of a category, used in JSON.
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
// that doubles as the JSON message; file and ids expose the same anchors
// structurally. Only filename drift is fixable and carries the fix payload; fixed
// flips to true once --fix has renamed it.
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

// report is the whole health scan: how many valid issues were checked and every
// finding, most serious first.
type report struct {
	checked  int
	findings []finding
}

// remaining counts the non-advisory findings not repaired this run, which is what
// the exit code and the ok flag turn on.
func (r *report) remaining() int {
	n := 0
	for _, f := range r.findings {
		if !f.fixed && !f.cat.advisory() {
			n++
		}
	}
	return n
}

// advisoryCount counts the informational findings.
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

// applyFixes renames each drifted file to its canonical name through the store,
// which refuses to overwrite another file, so a fix never destroys data. Passes
// repeat while any makes progress, so chained drifts that free each other's names
// all resolve; a destination that stays occupied (a mutual swap) is left standing
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

// renderHuman writes a headline, an aligned class/detail table of the findings,
// and a closing summary.
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

// foundClause words the headline's count, keeping problems and advisory notes
// apart so an advisory-only report does not read as an unhealthy store.
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

// advisoryOnlyLine closes a report whose only findings are advisory.
const advisoryOnlyLine = "Advisory notes are informational and do not fail doctor; the store is otherwise healthy."

// writeSummary writes the closing line: before --fix, an offer to repair the
// fixable findings; after --fix, an accounting of what was repaired and what still
// needs a human.
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

// reportView is the report's stable JSON shape. ok is true exactly when no
// problems remain, so a consumer can read it instead of the exit code.
type reportView struct {
	OK       bool          `json:"ok"`
	Checked  int           `json:"checked"`
	Problems int           `json:"problems"`
	Fixed    int           `json:"fixed"`
	Findings []findingView `json:"findings"`
}

// findingView is one finding in JSON. Every key is always present (file is null
// when the finding spans files; ids is [] never null), so a consumer never
// special-cases a missing key. advisory marks the informational classes, which
// never count toward problems or flip ok.
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

// sortFindings orders findings by category (most serious first), then file, first
// id, and detail — a total order, so the report is deterministic regardless of
// file iteration order.
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

// plural returns the plural suffix for a count ("1 issue", "2 issues").
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// verbS returns the verb suffix agreeing with a count ("1 problem needs" vs
// "2 problems need").
func verbS(n int) string {
	if n == 1 {
		return "s"
	}
	return ""
}
