package cli

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

// cmdDoctor reports the store's health and, with --fix, repairs the lint-class
// problems it is safe to repair on its own. It is the capstone integrity check
// (n9b4a7): where every other command tolerates and skips a broken file so it can
// keep serving the valid ones (ADR 0005), doctor is the one command whose whole job
// is to name what is wrong. It scans every file, reports the problems it finds, and
// exits non-zero when any remain, so a human — or CI — can tell a clean store from a
// degraded one without reading the output.
//
// The problem classes span the two halves of ADR 0005's contract. Hard validation
// errors — a file that is not a usable issue at all — are reported, never
// auto-"fixed": there is nothing safe to guess. The lint classes — a filename that
// has drifted from its authoritative frontmatter, a frontmatter key that looks like
// a typo of a known field (ADR 0014), a dangling depends_on/parent reference, a
// dependency or parent cycle, an issue stuck on a cancelled one, two files claiming
// one id — are reported, and the one that is mechanically safe to repair (filename
// drift) is what --fix repairs. --fix never removes an unknown key: removal is data
// loss, not tidying, and stays a human decision (ADR 0014). The unknown-key class
// is advisory besides: a resembling key is only *likely* a typo, so it is reported
// for a human's eye but never fails doctor — see category.advisory. Output
// auto-detects human vs JSON like every other command (ADR 0013).
func cmdDoctor(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "doctor")
	fixFlag := fs.Bool("fix", false, "repair lint-class problems (filename drift); never removes data")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "doctor takes no arguments")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	// Discover the store directly rather than through discover(): doctor reports
	// invalid files itself, in its report, so it must not also install the store's
	// stderr warning handler — that would double-report each broken file (and, in
	// JSON mode, splatter warnings beside the object an agent parses).
	st, err := store.Discover(env.WorkDir)
	if err != nil {
		return storeError(env, err)
	}

	rep, err := diagnose(env, st)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	if *fixFlag {
		rep.applyFixes(st)
	}

	if err := rep.render(env.Stdout, format, *fixFlag); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	// Non-zero when problems remain, so a script or agent can branch on store health
	// without parsing the report. A --fix run that repaired everything exits clean.
	if rep.remaining() > 0 {
		return exitError
	}
	return exitOK
}

// located pairs a valid, parsed issue with the absolute path it was read from — the
// two things doctor needs together to check a file against its authoritative
// frontmatter (filename drift) and to repair it in place.
type located struct {
	iss  issue.Issue
	path string
}

// diagnose scans every file in the store and builds the health report. It does its
// own tolerant read — List plus the single-file Read, the same "is this a usable
// issue" contract scan applies store-wide (ADR 0005) — so it sees both halves the
// normal readers hide from each other: the valid issues (with their paths, for the
// filename and graph checks) and the files that are not usable issues at all (with
// the reason, for the invalid-file findings).
func diagnose(env Env, st *store.Store) (*report, error) {
	files, err := st.List()
	if err != nil {
		return nil, err
	}

	var valid []located
	var findings []finding
	for _, f := range files {
		iss, rerr := st.Read(f)
		if rerr != nil {
			// A hard validation failure: reported, never auto-fixed (ADR 0005).
			findings = append(findings, invalidFinding(relPath(env.WorkDir, f), rerr))
			continue
		}
		valid = append(valid, located{iss: iss, path: f})
	}

	findings = append(findings, duplicateIDFindings(env, valid)...)
	findings = append(findings, lintFindings(env, valid)...)
	findings = append(findings, graphFindings(env, valid)...)

	sortFindings(findings)
	return &report{checked: len(valid), findings: findings}, nil
}

// duplicateIDFindings reports each id claimed by more than one file. It is a
// half-merged state validation cannot catch — every file is a valid issue on its
// own, but they disagree about identity (ADR 0002) — and it must be resolved by a
// human, so it is never auto-fixed. It is computed first because a duplicated id
// makes its files' canonical name contested: lintFindings suppresses filename-drift
// repairs for those files, since renaming one onto the other's canonical name would
// clobber it, and shuffling names does not resolve the real clash anyway.
func duplicateIDFindings(env Env, valid []located) []finding {
	byID := make(map[string][]string)
	for _, lc := range valid {
		byID[lc.iss.ID] = append(byID[lc.iss.ID], relPath(env.WorkDir, lc.path))
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []finding
	for _, id := range ids {
		if paths := byID[id]; len(paths) > 1 {
			sort.Strings(paths)
			out = append(out, finding{
				cat:    catDuplicateID,
				ids:    []string{id},
				detail: fmt.Sprintf("id %s is used by %d files: %s", id, len(paths), strings.Join(paths, ", ")),
			})
		}
	}
	return out
}

// lintFindings reports the per-file lint of a valid issue: a filename that has
// drifted from the canonical <id>-<slug> its frontmatter dictates, any frontmatter
// key that looks like a typo of a known field, a priority that is not one of the
// four levels, and a missing created/updated timestamp. The value and timestamp
// lints exist because both states are silent damage a hand-edit can cause — an
// unrecognized priority matches no --priority filter (not even none), and an issue
// with no created time sorts as the oldest in every list — yet neither is a load
// failure (ADR 0005), so only doctor ever surfaces them. Neither is fixable:
// guessing the intended level, or inventing a date, is exactly what --fix never
// does. Filename drift is the one class doctor can repair, so its finding carries
// the payload --fix needs; it is withheld (not merely marked unfixable) for a
// duplicated id, whose contested canonical name no rename can safely take.
func lintFindings(env Env, valid []located) []finding {
	dup := duplicatedIDs(valid)
	var out []finding
	for _, lc := range valid {
		canonical := issue.FileName(lc.iss.ID, issue.Slug(lc.iss.Title))
		if filepath.Base(lc.path) != canonical && !dup[lc.iss.ID] {
			out = append(out, finding{
				cat:     catDrift,
				file:    relPath(env.WorkDir, lc.path),
				ids:     []string{lc.iss.ID},
				detail:  fmt.Sprintf("%s: should be named %s", relPath(env.WorkDir, lc.path), canonical),
				fixable: true,
				want:    canonical,
				fixSrc:  lc.path,
				fixIss:  lc.iss,
			})
		}
		for _, key := range issue.CustomKeys(lc.iss.Custom) {
			if near, ok := nearestField(key); ok {
				out = append(out, finding{
					cat:    catUnknownKey,
					file:   relPath(env.WorkDir, lc.path),
					ids:    []string{lc.iss.ID},
					detail: fmt.Sprintf("%s: %q looks like a typo of %q", relPath(env.WorkDir, lc.path), key, near),
				})
			}
		}
		if !lc.iss.Priority.Valid() {
			out = append(out, finding{
				cat:    catUnknownValue,
				file:   relPath(env.WorkDir, lc.path),
				ids:    []string{lc.iss.ID},
				detail: fmt.Sprintf("%s: priority %q is not a recognized level (urgent|high|medium|low); no --priority filter will match it", lc.iss.ID, lc.iss.Priority),
			})
		}
		if missing := missingTimestamps(lc.iss); missing != "" {
			out = append(out, finding{
				cat:    catNoTimestamp,
				file:   relPath(env.WorkDir, lc.path),
				ids:    []string{lc.iss.ID},
				detail: fmt.Sprintf("%s: missing %s timestamp (an issue with no created time sorts as the oldest)", lc.iss.ID, missing),
			})
		}
	}
	return out
}

// missingTimestamps names the timestamp fields iss lacks ("created", "updated", or
// "created and updated"), or "" when both are set. A zero time means the field was
// absent from the file — Marshal never writes a zero — and stays absent on rewrite.
func missingTimestamps(iss issue.Issue) string {
	switch {
	case iss.Created.IsZero() && iss.Updated.IsZero():
		return "created and updated"
	case iss.Created.IsZero():
		return "created"
	case iss.Updated.IsZero():
		return "updated"
	default:
		return ""
	}
}

// graphFindings reports the relationship problems that only show up across the whole
// valid set: a depends_on/parent edge to an id no issue holds (a dangling
// reference), a dependency cycle, a parent cycle (a hierarchy loop no tree could
// render, including an issue that is its own parent), and an open issue stuck on a
// cancelled one. None is auto-fixed — each is a human's call to re-scope, drop an
// edge, or create the missing issue. Stuck and the cycles reuse the derived
// relationship engine (issue) so doctor and the rest of Busy Beaver share one
// definition of each condition.
func graphFindings(env Env, valid []located) []finding {
	idset := make(map[string]bool, len(valid))
	issues := make([]issue.Issue, len(valid))
	for i, lc := range valid {
		idset[lc.iss.ID] = true
		issues[i] = lc.iss
	}
	rel := issue.NewRelations(issues)

	var out []finding
	for _, lc := range valid {
		iss := lc.iss
		for _, dep := range dedupe(iss.DependsOn) {
			if !idset[dep] {
				out = append(out, danglingFinding(env, lc, "depends_on", dep))
			}
		}
		if iss.Parent != "" && !idset[iss.Parent] {
			out = append(out, danglingFinding(env, lc, "parent", iss.Parent))
		}
		// Stuck is scoped to the still-open issues: a done or cancelled dependent has
		// nothing left to wait for, so a cancelled dependency is only a live problem
		// for work that still needs to proceed.
		if iss.State == issue.StateTodo || iss.State == issue.StateInProgress {
			if cancelled := cancelledDeps(rel, iss); len(cancelled) > 0 {
				out = append(out, finding{
					cat:    catStuck,
					file:   relPath(env.WorkDir, lc.path),
					ids:    []string{iss.ID},
					detail: fmt.Sprintf("%s: waits on cancelled %s", iss.ID, strings.Join(cancelled, ", ")),
				})
			}
		}
	}
	for _, cycle := range rel.Cycles() {
		out = append(out, finding{
			cat:    catCycle,
			ids:    cycle,
			detail: "dependency cycle: " + strings.Join(cycle, ", "),
		})
	}
	for _, cycle := range rel.ParentCycles() {
		detail := "parent cycle: " + strings.Join(cycle, ", ")
		if len(cycle) == 1 {
			detail = fmt.Sprintf("%s: is its own parent", cycle[0])
		}
		out = append(out, finding{
			cat:    catParentCycle,
			ids:    cycle,
			detail: detail,
		})
	}
	return out
}

// danglingFinding reports one edge — a depends_on or parent — whose target id names
// no issue in the store.
func danglingFinding(env Env, lc located, field, target string) finding {
	return finding{
		cat:    catDangling,
		file:   relPath(env.WorkDir, lc.path),
		ids:    []string{lc.iss.ID},
		detail: fmt.Sprintf("%s: %s %s, but no such issue exists", lc.iss.ID, field, target),
	}
}

// invalidFinding reports a file that is not a usable issue, carrying the exact reason
// the store's read contract gave (ADR 0005): a malformed frontmatter, a missing or
// malformed id, an illegal state. It is anchored to the file, not an id, because an
// invalid file has no identity doctor can trust.
func invalidFinding(fileRel string, err error) finding {
	return finding{
		cat:    catInvalid,
		file:   fileRel,
		detail: fmt.Sprintf("%s: %v", fileRel, err),
	}
}

// cancelledDeps returns the ids of iss's dependencies that are cancelled — the edges
// that leave it stuck, since only done satisfies a dependency and cancellation never
// will (ADR 0011). The order follows the issue's stored depends_on.
func cancelledDeps(rel *issue.Relations, iss issue.Issue) []string {
	var out []string
	for _, b := range rel.BlockedOn(iss) {
		if b.Cancelled() {
			out = append(out, b.ID)
		}
	}
	return out
}

// duplicatedIDs is the set of ids held by more than one valid file.
func duplicatedIDs(valid []located) map[string]bool {
	count := make(map[string]int, len(valid))
	for _, lc := range valid {
		count[lc.iss.ID]++
	}
	dup := make(map[string]bool)
	for id, n := range count {
		if n > 1 {
			dup[id] = true
		}
	}
	return dup
}

// --- findings and the report ---

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

// --- typo detection ---

// knownFields are the frontmatter keys Busy Beaver defines, in the on-disk spelling a
// hand-edit would aim for. A custom key close to one of these is very likely a typo
// of it (ADR 0014).
var knownFields = []string{"id", "title", "state", "assignee", "priority", "labels", "depends_on", "parent", "created", "updated"}

// nearestField reports the known field a custom key most looks like a typo of, when
// one is within a small edit distance. This is the misspelled-key catch ADR 0014
// hands to doctor: a key like `assigne` is preserved verbatim by the serializer (it
// is never data loss to keep it), but it silently did nothing, so doctor flags it as
// a likely typo of `assignee` for a human to correct. It is deliberately narrow — a
// distance of one or two, and only for keys long enough that the match is meaningful
// — so a deliberate custom field like `sprint` or `estimate`, near no known field,
// is not flagged at all. Narrowness alone cannot clear every legitimate key, though
// (`status` sits two edits from `state`), which is why the whole class is advisory
// (category.advisory): flagged for a look, never a failed bill of health.
func nearestField(key string) (string, bool) {
	if len(key) < 3 {
		return "", false
	}
	best, bestDist := "", 0
	for _, f := range knownFields {
		d := levenshtein(key, f)
		if d == 0 {
			return "", false // an exact known key is not custom; nothing to flag
		}
		if best == "" || d < bestDist {
			best, bestDist = f, d
		}
	}
	if bestDist >= 1 && bestDist <= 2 {
		return best, true
	}
	return "", false
}

// levenshtein is the edit distance between two ASCII strings — the number of
// single-character insertions, deletions, or substitutions to turn one into the
// other. Frontmatter keys are short ASCII identifiers, so the plain two-row dynamic
// program is more than fast enough.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur := make([]int, lb+1)
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, min(cur[j-1]+1, prev[j-1]+cost))
		}
		prev = cur
	}
	return prev[lb]
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
