package cli

// This file holds doctor's presentation: the words each class of finding is
// stated in, the paths rendered relative to where the command was run, and the
// human and JSON shapes of the report. The findings themselves are the core's.

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/output"
)

// renderReport writes a health report in the resolved format.
func renderReport(env Env, rep core.Report, format output.Format, fix bool) error {
	if format == output.JSON {
		return renderReportJSON(env, rep)
	}
	return renderReportHuman(env, rep, fix)
}

// findingLabel is the short human name of a category, the left column of the
// human report.
func findingLabel(c core.Category) string {
	switch c {
	case core.CategoryInvalid:
		return "invalid"
	case core.CategoryDuplicateID:
		return "duplicate id"
	case core.CategoryDependencyCycle:
		return "cycle"
	case core.CategoryParentCycle:
		return "parent cycle"
	case core.CategoryDanglingRef:
		return "dangling ref"
	case core.CategoryStuck:
		return "stuck"
	case core.CategoryUnknownKey:
		return "unknown key"
	case core.CategoryUnknownValue:
		return "unknown value"
	case core.CategoryMissingTimestamp:
		return "missing time"
	case core.CategoryFilenameDrift:
		return "filename drift"
	}
	return "problem"
}

// findingMessage states a finding in one self-contained line, which doubles as
// the JSON message. It is built from the facts the core hands back rather than
// carried with them, so the file names in it can be rendered relative to the
// working directory and the wording stays this interface's.
func findingMessage(env Env, f core.Finding) string {
	id, path := firstID(f.IDs), relPath(env.WorkDir, f.Path())
	switch f.Category {
	case core.CategoryInvalid:
		return fmt.Sprintf("%s: %v", path, f.Err)
	case core.CategoryDuplicateID:
		return fmt.Sprintf("id %s is used by %d files: %s",
			id, len(f.Paths), strings.Join(relPaths(env, f.Paths), ", "))
	case core.CategoryDependencyCycle:
		return "dependency cycle: " + strings.Join(f.IDs, ", ")
	case core.CategoryParentCycle:
		// A single-issue cycle is an issue that is its own parent, which reads as
		// nonsense described as a loop.
		if len(f.IDs) == 1 {
			return fmt.Sprintf("%s: is its own parent", id)
		}
		return "parent cycle: " + strings.Join(f.IDs, ", ")
	case core.CategoryDanglingRef:
		return fmt.Sprintf("%s: %s %s, but no such issue exists", id, f.Field, f.Target)
	case core.CategoryStuck:
		return fmt.Sprintf("%s: waits on cancelled %s", id, strings.Join(f.Cancelled, ", "))
	case core.CategoryUnknownKey:
		return fmt.Sprintf("%s: %q looks like a typo of %q", path, f.Key, f.Resembles)
	case core.CategoryUnknownValue:
		return fmt.Sprintf("%s: priority %q is not a recognized level (urgent|high|medium|low); no --priority filter will match it", id, f.Value)
	case core.CategoryMissingTimestamp:
		return fmt.Sprintf("%s: missing %s timestamp (an issue with no created time sorts as the oldest)",
			id, strings.Join(f.Missing, " and "))
	case core.CategoryFilenameDrift:
		return fmt.Sprintf("%s: should be named %s", path, f.Canonical)
	}
	return f.Category.String()
}

// renderReportHuman writes a headline, an aligned class/detail table of the
// findings, and a closing summary.
func renderReportHuman(env Env, rep core.Report, fix bool) error {
	if len(rep.Findings) == 0 {
		_, err := fmt.Fprintf(env.Stdout, "No problems found (checked %d issue%s).\n", rep.Checked, plural(rep.Checked))
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %s (checked %d issue%s):\n\n",
		foundClause(len(rep.Findings)-rep.Advisories(), rep.Advisories()), rep.Checked, plural(rep.Checked))
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, f := range rep.Findings {
		class, detail := findingLabel(f.Category), findingMessage(env, f)
		if f.Fixed {
			class, detail = "fixed", "renamed to "+f.Canonical
		}
		fmt.Fprintf(tw, "  %s\t%s\n", class, detail)
	}
	tw.Flush()

	b.WriteByte('\n')
	writeSummary(&b, rep, fix)
	_, err := io.WriteString(env.Stdout, b.String())
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
func writeSummary(b *strings.Builder, rep core.Report, fix bool) {
	remaining := rep.Problems()
	if fix {
		fixed := rep.Fixed()
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
	if n := rep.Fixable(); n > 0 {
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

func renderReportJSON(env Env, rep core.Report) error {
	views := make([]findingView, len(rep.Findings))
	for i, f := range rep.Findings {
		views[i] = findingView{
			Category: f.Category.String(),
			File:     nilIfEmpty(relPath(env.WorkDir, f.Path())),
			IDs:      strsOrEmpty(f.IDs),
			Message:  findingMessage(env, f),
			Fixable:  f.Fixable,
			Fixed:    f.Fixed,
			Advisory: f.Category.Advisory(),
		}
	}
	return output.WriteJSON(env.Stdout, reportView{
		OK:       rep.Problems() == 0,
		Checked:  rep.Checked,
		Problems: rep.Problems(),
		Fixed:    rep.Fixed(),
		Findings: views,
	})
}

// --- small helpers ---

func firstID(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

// relPaths renders every path relative to the working directory, in the order
// given.
func relPaths(env Env, paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = relPath(env.WorkDir, p)
	}
	return out
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
