package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"beaver/internal/issue"
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
