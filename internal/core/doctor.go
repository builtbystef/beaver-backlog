package core

// This file holds the health engine: the classes of problem a store can hold,
// the scan that finds every one of them, and the single repair that is
// mechanically safe. A finding is stated as the facts behind it rather than as a
// sentence, so an interface words it, and renders the files it names, its own
// way.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// Category is the class of a health problem. Declaration order is severity
// order, most serious first, so a report sorts into a stable list.
type Category int

const (
	CategoryInvalid Category = iota
	CategoryDuplicateID
	CategoryDependencyCycle
	CategoryParentCycle
	CategoryDanglingRef
	CategoryStuck
	CategoryUnknownKey
	CategoryUnknownValue
	CategoryMissingTimestamp
	CategoryFilenameDrift
)

// String is the category's stable machine name, the form a machine consumer
// matches on.
func (c Category) String() string {
	switch c {
	case CategoryInvalid:
		return "invalid"
	case CategoryDuplicateID:
		return "duplicate_id"
	case CategoryDependencyCycle:
		return "dependency_cycle"
	case CategoryParentCycle:
		return "parent_cycle"
	case CategoryDanglingRef:
		return "dangling_reference"
	case CategoryStuck:
		return "stuck"
	case CategoryUnknownKey:
		return "unknown_key"
	case CategoryUnknownValue:
		return "unknown_value"
	case CategoryMissingTimestamp:
		return "missing_timestamp"
	case CategoryFilenameDrift:
		return "filename_drift"
	}
	return "unknown"
}

// Advisory reports whether findings of this class are informational: reported,
// but never counted as problems. Unknown-key is advisory because resemblance is
// only a guess: a deliberate custom field like `status` sits within typo
// distance of `state` and must not make a healthy store read as unhealthy.
func (c Category) Advisory() bool { return c == CategoryUnknownKey }

// Finding is one health problem, given as the anchors it concerns and the facts
// behind it. Which facts are set depends on the category, and each is named
// below by the class that carries it; a caller phrases the problem from them
// rather than parsing a message.
type Finding struct {
	Category Category
	// Paths holds the files at fault, in path order. Exactly one for a
	// per-file problem, several for one that spans files (an id two files
	// claim), none for a problem in the graph alone (a cycle).
	Paths   []string
	IDs     []string // the issue ids the finding concerns
	Fixable bool     // whether a repairing scan can fix it at all
	Fixed   bool     // whether this scan did

	Err       error    // Invalid: why the file is not a usable issue
	Canonical string   // FilenameDrift: the <id>-<slug> name the file should hold, and once fixed the name it does
	Key       string   // UnknownKey: the frontmatter key that looks like a typo
	Resembles string   // UnknownKey: the known field it resembles
	Value     string   // UnknownValue: the value no level recognizes
	Missing   []string // MissingTimestamp: the timestamp fields the issue lacks
	Field     string   // DanglingRef: the frontmatter field holding the edge
	Target    string   // DanglingRef: the id it names, which no issue holds
	Cancelled []string // Stuck: the cancelled dependencies it waits on
}

// Path is the one file a finding is anchored to, or "" when it spans several
// (a duplicate id) or names none (a cycle).
func (f Finding) Path() string {
	if len(f.Paths) == 1 {
		return f.Paths[0]
	}
	return ""
}

// Report is one health scan: how many usable issues it checked and every
// finding, most serious first.
type Report struct {
	Checked  int
	Findings []Finding
}

// Problems counts the findings that still stand, neither repaired by this scan
// nor advisory. It is what a caller calling the store unhealthy turns on.
func (r Report) Problems() int {
	return r.count(func(f Finding) bool { return !f.Fixed && !f.Category.Advisory() })
}

// Advisories counts the informational findings, which never count as problems.
func (r Report) Advisories() int {
	return r.count(func(f Finding) bool { return f.Category.Advisory() })
}

// Fixed counts the findings this scan repaired.
func (r Report) Fixed() int {
	return r.count(func(f Finding) bool { return f.Fixed })
}

// Fixable counts the findings still standing that a repairing scan could fix.
func (r Report) Fixable() int {
	return r.count(func(f Finding) bool { return f.Fixable && !f.Fixed })
}

func (r Report) count(pred func(Finding) bool) int {
	n := 0
	for _, f := range r.Findings {
		if pred(f) {
			n++
		}
	}
	return n
}

// Doctor scans the store and reports everything wrong with it: files that are
// not usable issues, ids claimed twice, relationship anomalies, per-file lint,
// and filenames drifted from the canonical <id>-<slug>. With fix set it repairs
// the drift, the one class a machine can resolve without guessing, and never
// removes data or touches a validation error.
//
// Unlike every other read, an unusable file is a finding here rather than a
// warning: reporting them is the whole point of the scan, so nothing is skipped
// silently and nothing is reported twice.
func (s *Service) Doctor(fix bool) (Report, error) {
	files, err := s.store.List()
	if err != nil {
		return Report{}, err
	}

	var valid []located
	var findings []Finding
	for _, f := range files {
		iss, rerr := s.store.Read(f)
		if rerr != nil {
			findings = append(findings, Finding{Category: CategoryInvalid, Paths: []string{f}, Err: rerr})
			continue
		}
		valid = append(valid, located{iss: iss, path: f})
	}

	findings = append(findings, duplicateIDFindings(valid)...)
	findings = append(findings, lintFindings(valid)...)
	findings = append(findings, graphFindings(valid)...)
	sortFindings(findings)

	if fix {
		s.repair(findings, byPath(valid))
	}
	return Report{Checked: len(valid), Findings: findings}, nil
}

// located pairs a usable issue with the file it was read from.
type located struct {
	iss  issue.Issue
	path string
}

// byPath indexes the usable issues by the file they came from, so a repair can
// reach the issue whose frontmatter dictates its file's canonical name.
func byPath(valid []located) map[string]issue.Issue {
	out := make(map[string]issue.Issue, len(valid))
	for _, lc := range valid {
		out[lc.path] = lc.iss
	}
	return out
}

// repair renames each drifted file to the canonical name its frontmatter
// implies, through the store, which refuses to overwrite another file, so a
// repair never destroys data. Passes repeat while any makes progress, so chained
// drifts that free each other's names all resolve; a destination that stays
// occupied (a mutual swap) is left standing and reported, never forced.
//
// A repaired finding keeps the path it was found at: what the file was called is
// the fact worth reporting, and Canonical already names what it is called now.
func (s *Service) repair(findings []Finding, issues map[string]issue.Issue) {
	for {
		progress := false
		for i := range findings {
			f := &findings[i]
			if !f.Fixable || f.Fixed {
				continue
			}
			if _, err := s.store.Rename(f.Path(), issues[f.Path()]); err != nil {
				continue // destination taken (or a write error): leave it standing
			}
			f.Fixed = true
			progress = true
		}
		if !progress {
			return
		}
	}
}

// duplicateIDFindings reports each id claimed by more than one file. A duplicate
// must be resolved by a human, so it is never auto-fixed; lintFindings also
// withholds filename-drift repairs for these files, since renaming one onto the
// contested canonical name would clobber the other.
func duplicateIDFindings(valid []located) []Finding {
	byID := make(map[string][]string)
	for _, lc := range valid {
		byID[lc.iss.ID] = append(byID[lc.iss.ID], lc.path)
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []Finding
	for _, id := range ids {
		if paths := byID[id]; len(paths) > 1 {
			sort.Strings(paths)
			out = append(out, Finding{Category: CategoryDuplicateID, IDs: []string{id}, Paths: paths})
		}
	}
	return out
}

// lintFindings reports the per-file lint of a usable issue: a filename drifted
// from the canonical <id>-<slug>, frontmatter keys that look like typos of known
// fields, an unrecognized priority value, and missing created/updated
// timestamps. Only the drift is fixable, and it is withheld for a duplicated id,
// whose contested canonical name no rename can safely take.
func lintFindings(valid []located) []Finding {
	dup := duplicatedIDs(valid)
	var out []Finding
	for _, lc := range valid {
		canonical := issue.FileName(lc.iss.ID, issue.Slug(lc.iss.Title))
		if filepath.Base(lc.path) != canonical && !dup[lc.iss.ID] {
			out = append(out, Finding{
				Category:  CategoryFilenameDrift,
				Paths:     []string{lc.path},
				IDs:       []string{lc.iss.ID},
				Fixable:   true,
				Canonical: canonical,
			})
		}
		for _, key := range issue.CustomKeys(lc.iss.Custom) {
			if near, ok := nearestField(key); ok {
				out = append(out, Finding{
					Category:  CategoryUnknownKey,
					Paths:     []string{lc.path},
					IDs:       []string{lc.iss.ID},
					Key:       key,
					Resembles: near,
				})
			}
		}
		if !lc.iss.Priority.Valid() {
			out = append(out, Finding{
				Category: CategoryUnknownValue,
				Paths:    []string{lc.path},
				IDs:      []string{lc.iss.ID},
				Value:    string(lc.iss.Priority),
			})
		}
		if missing := missingTimestamps(lc.iss); len(missing) > 0 {
			out = append(out, Finding{
				Category: CategoryMissingTimestamp,
				Paths:    []string{lc.path},
				IDs:      []string{lc.iss.ID},
				Missing:  missing,
			})
		}
	}
	return out
}

// missingTimestamps names the timestamp fields iss lacks, in frontmatter order,
// or nothing when both are set. A zero time means the field was absent from the
// file.
func missingTimestamps(iss issue.Issue) []string {
	var out []string
	if iss.Created.IsZero() {
		out = append(out, "created")
	}
	if iss.Updated.IsZero() {
		out = append(out, "updated")
	}
	return out
}

// graphFindings reports relationship problems across the whole usable set:
// dangling depends_on/parent references, dependency cycles, parent cycles
// (including an issue that is its own parent), and open issues stuck on a
// cancelled dependency. None is auto-fixed; each is a human's call.
func graphFindings(valid []located) []Finding {
	idset := make(map[string]bool, len(valid))
	issues := make([]issue.Issue, len(valid))
	for i, lc := range valid {
		idset[lc.iss.ID] = true
		issues[i] = lc.iss
	}
	rel := issue.NewRelations(issues)

	var out []Finding
	for _, lc := range valid {
		iss := lc.iss
		for _, dep := range dedupe(iss.DependsOn) {
			if !idset[dep] {
				out = append(out, danglingFinding(lc, "depends_on", dep))
			}
		}
		if iss.Parent != "" && !idset[iss.Parent] {
			out = append(out, danglingFinding(lc, "parent", iss.Parent))
		}
		// Only still-open issues can be stuck; a done or cancelled dependent has
		// nothing left to wait for.
		if iss.State == issue.StateTodo || iss.State == issue.StateInProgress {
			if cancelled := cancelledDeps(rel, iss); len(cancelled) > 0 {
				out = append(out, Finding{
					Category:  CategoryStuck,
					Paths:     []string{lc.path},
					IDs:       []string{iss.ID},
					Cancelled: cancelled,
				})
			}
		}
	}
	for _, cycle := range rel.Cycles() {
		out = append(out, Finding{Category: CategoryDependencyCycle, IDs: cycle})
	}
	for _, cycle := range rel.ParentCycles() {
		out = append(out, Finding{Category: CategoryParentCycle, IDs: cycle})
	}
	return out
}

// danglingFinding reports a depends_on or parent edge whose target id names no
// issue in the store.
func danglingFinding(lc located, field, target string) Finding {
	return Finding{
		Category: CategoryDanglingRef,
		Paths:    []string{lc.path},
		IDs:      []string{lc.iss.ID},
		Field:    field,
		Target:   target,
	}
}

// cancelledDeps returns the ids of iss's cancelled dependencies, in stored
// depends_on order. Only done satisfies a dependency, so these leave iss stuck.
func cancelledDeps(rel *issue.Relations, iss issue.Issue) []string {
	var out []string
	for _, b := range rel.BlockedOn(iss) {
		if b.Cancelled() {
			out = append(out, b.ID)
		}
	}
	return out
}

// duplicatedIDs is the set of ids held by more than one usable file.
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

// sortFindings orders findings by category (most serious first), then by the
// files and ids they name, then by the facts that tell two findings of one class
// on one file apart, such as the two dangling edges out of a single issue. The
// order is total, so a report never depends on the order the directory was read
// in.
func sortFindings(fs []Finding) {
	sort.Slice(fs, func(i, j int) bool { return sortKey(fs[i]) < sortKey(fs[j]) })
}

// sortKey renders a finding as the string its position is decided by. The
// category is fixed-width so it compares as a number, and the parts are joined
// on a byte no field can hold, so no part can bleed into the next.
func sortKey(f Finding) string {
	return strings.Join([]string{
		fmt.Sprintf("%02d", f.Category),
		strings.Join(f.Paths, ","),
		strings.Join(f.IDs, ","),
		f.Field, f.Target, f.Key, f.Value,
		strings.Join(f.Missing, ","),
		strings.Join(f.Cancelled, ","),
	}, "\x00")
}

// --- typo detection ---

// knownFields are the frontmatter keys Beaver Backlog defines, in their on-disk
// spelling.
var knownFields = []string{"id", "title", "state", "assignee", "priority", "labels", "depends_on", "parent", "created", "updated"}

// nearestField reports the known field a custom key most looks like a typo of, when
// one is within an edit distance of one or two. The threshold is deliberately
// narrow so a deliberate custom field like `sprint` is not flagged; even so,
// resemblance is only a guess (`status` sits two edits from `state`), which is why
// the whole class is advisory.
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

// levenshtein is the edit distance between two ASCII strings. Frontmatter keys are
// short, so the plain two-row dynamic program is fast enough.
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
