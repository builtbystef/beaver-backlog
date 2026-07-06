package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/builtbystef/busy-beaver/internal/issue"
	"github.com/builtbystef/busy-beaver/internal/store"
)

// cmdDoctor scans every file in the store, reports problems, and exits non-zero
// while any remain. Hard validation errors are only reported; --fix repairs
// filename drift, the one mechanically safe class, and never removes data.
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

	// Bypass discover(): doctor reports invalid files itself, and the store's
	// stderr warning handler would double-report them.
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
	// Non-zero while problems remain, so scripts can branch on store health
	// without parsing the report.
	if rep.remaining() > 0 {
		return exitError
	}
	return exitOK
}

// located pairs a parsed issue with the absolute path it was read from.
type located struct {
	iss  issue.Issue
	path string
}

// diagnose scans every file in the store and builds the health report. It reads
// file by file so it sees both the valid issues (with their paths) and the files
// that are not usable issues at all (with the reason).
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

// duplicateIDFindings reports each id claimed by more than one file. A duplicate
// must be resolved by a human, so it is never auto-fixed; lintFindings also
// withholds filename-drift repairs for these files, since renaming one onto the
// contested canonical name would clobber the other.
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

// lintFindings reports the per-file lint of a valid issue: filename drift from the
// canonical <id>-<slug>, frontmatter keys that look like typos of known fields,
// unrecognized priority values, and missing created/updated timestamps. Only the
// drift finding is fixable, and it is withheld for a duplicated id, whose contested
// canonical name no rename can safely take.
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

// missingTimestamps names the timestamp fields iss lacks, or "" when both are set.
// A zero time means the field was absent from the file.
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

// graphFindings reports relationship problems across the whole valid set: dangling
// depends_on/parent references, dependency cycles, parent cycles (including an
// issue that is its own parent), and open issues stuck on a cancelled dependency.
// None is auto-fixed; each is a human's call.
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
		// Only still-open issues can be stuck; a done or cancelled dependent has
		// nothing left to wait for.
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

// danglingFinding reports a depends_on or parent edge whose target id names no
// issue in the store.
func danglingFinding(env Env, lc located, field, target string) finding {
	return finding{
		cat:    catDangling,
		file:   relPath(env.WorkDir, lc.path),
		ids:    []string{lc.iss.ID},
		detail: fmt.Sprintf("%s: %s %s, but no such issue exists", lc.iss.ID, field, target),
	}
}

// invalidFinding reports a file that is not a usable issue. It is anchored to the
// file, not an id, because an invalid file has no identity doctor can trust.
func invalidFinding(fileRel string, err error) finding {
	return finding{
		cat:    catInvalid,
		file:   fileRel,
		detail: fmt.Sprintf("%s: %v", fileRel, err),
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

// --- typo detection ---

// knownFields are the frontmatter keys Busy Beaver defines, in their on-disk
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
