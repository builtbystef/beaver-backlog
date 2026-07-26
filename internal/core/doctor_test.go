package core_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestDoctorFindsNothingWrongWithACleanStore(t *testing.T) {
	root := newStore(t)
	seed(t, root, mkIssue("good11", "First"))
	seed(t, root, withState(mkIssue("good22", "Second"), issue.StateDone))

	rep := doctor(t, root, false)
	if len(rep.Findings) != 0 {
		t.Errorf("clean store reported %v", categories(rep))
	}
	if rep.Checked != 2 {
		t.Errorf("Checked = %d, want the 2 usable issues", rep.Checked)
	}
	if rep.Problems() != 0 || rep.Fixable() != 0 {
		t.Errorf("problems=%d fixable=%d, want both 0", rep.Problems(), rep.Fixable())
	}
}

// Every class of problem coexists in one store, so this pins both detection and
// that no class masks another.
func TestDoctorDetectsEveryProblemClass(t *testing.T) {
	root := newStore(t)
	// A file that is not a usable issue at all.
	writeRaw(t, root, "bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n---\n")
	// A valid issue under a name its frontmatter does not imply.
	seedAt(t, root, "drift-wrong.md", mkIssue("drft01", "Drift Title"))
	// A frontmatter key that looks like a typo of a known field.
	seed(t, root, withCustom(mkIssue("unkey1", "Unknown Key"), map[string]any{"assigne": "someone"}))
	// A priority no level recognizes, and an issue with no timestamps at all.
	seed(t, root, withPriority(mkIssue("pri001", "Typo Priority"), "critical"))
	seed(t, root, undated(mkIssue("nots01", "No Stamps")))
	// Edges naming an id no issue holds.
	seed(t, root, withDeps(mkIssue("dang01", "Dangling Depends"), "ghost0"))
	seed(t, root, withParent(mkIssue("dang02", "Dangling Parent"), "ghost0"))
	// Two issues each waiting on the other, and two each claiming the other as parent.
	seed(t, root, withDeps(mkIssue("cyc0aa", "Cycle A"), "cyc0bb"))
	seed(t, root, withDeps(mkIssue("cyc0bb", "Cycle B"), "cyc0aa"))
	seed(t, root, withParent(mkIssue("pcy0aa", "Parent Cycle A"), "pcy0bb"))
	seed(t, root, withParent(mkIssue("pcy0bb", "Parent Cycle B"), "pcy0aa"))
	// An open issue whose only dependency was cancelled: waiting can never clear it.
	seed(t, root, withState(mkIssue("cnl001", "Cancelled Dep"), issue.StateCancelled))
	seed(t, root, withDeps(mkIssue("stk001", "Stuck Issue"), "cnl001"))
	// Two files claiming one id — a clash validation cannot catch.
	seedAt(t, root, "dup-a.md", mkIssue("dup001", "Dup A"))
	seedAt(t, root, "dup-b.md", mkIssue("dup001", "Dup B"))

	rep := doctor(t, root, false)
	got := categories(rep)
	for _, want := range []core.Category{
		core.CategoryInvalid,
		core.CategoryDuplicateID,
		core.CategoryDependencyCycle,
		core.CategoryParentCycle,
		core.CategoryDanglingRef,
		core.CategoryStuck,
		core.CategoryUnknownKey,
		core.CategoryUnknownValue,
		core.CategoryMissingTimestamp,
		core.CategoryFilenameDrift,
	} {
		if !slices.Contains(got, want) {
			t.Errorf("missing a %s finding; got %v", want, got)
		}
	}
	// The two dangling edges — a depends_on and a parent — are each their own finding.
	if n := countCategory(got, core.CategoryDanglingRef); n != 2 {
		t.Errorf("dangling findings = %d, want 2 (a depends_on and a parent)", n)
	}
	// Findings arrive in severity order, whatever order the directory was read in.
	if !slices.IsSortedFunc(got, func(a, b core.Category) int { return int(a) - int(b) }) {
		t.Errorf("findings are not in severity order: %v", got)
	}
}

// Each class carries the facts its problem is made of, so a caller can word it
// (or link the file) without parsing a sentence.
func TestDoctorFindingsCarryTheirFacts(t *testing.T) {
	root := newStore(t)
	writeRaw(t, root, "broken.md", "this is not an issue file\n")
	seedAt(t, root, "drift-wrong.md", mkIssue("drft01", "New Title"))
	seed(t, root, withCustom(mkIssue("unkey1", "Unknown Key"), map[string]any{"assigne": "someone"}))
	seed(t, root, withPriority(mkIssue("pri001", "Typo Priority"), "critical"))
	seed(t, root, undated(mkIssue("nots01", "No Stamps")))
	seed(t, root, withDeps(mkIssue("dang01", "Dangling Depends"), "ghost0"))
	seed(t, root, withState(mkIssue("cnl001", "Cancelled Dep"), issue.StateCancelled))
	seed(t, root, withDeps(mkIssue("stk001", "Stuck Issue"), "cnl001"))
	seed(t, root, withParent(mkIssue("self01", "Own Parent"), "self01"))

	rep := doctor(t, root, false)

	invalid := finding(t, rep, core.CategoryInvalid)
	if filepath.Base(invalid.Path()) != "broken.md" {
		t.Errorf("invalid finding path = %q, want broken.md", invalid.Path())
	}
	if invalid.Err == nil || !strings.Contains(invalid.Err.Error(), "frontmatter") {
		t.Errorf("invalid finding Err = %v, want the frontmatter problem", invalid.Err)
	}
	if len(invalid.IDs) != 0 {
		t.Errorf("invalid finding IDs = %v; an unusable file carries no identity to trust", invalid.IDs)
	}

	drift := finding(t, rep, core.CategoryFilenameDrift)
	if drift.Canonical != "drft01-new-title.md" {
		t.Errorf("drift Canonical = %q, want drft01-new-title.md", drift.Canonical)
	}
	if !drift.Fixable {
		t.Error("filename drift should be the fixable class")
	}

	if key := finding(t, rep, core.CategoryUnknownKey); key.Key != "assigne" || key.Resembles != "assignee" {
		t.Errorf("unknown key = %q resembling %q, want assigne resembling assignee", key.Key, key.Resembles)
	}
	if val := finding(t, rep, core.CategoryUnknownValue); val.Value != "critical" {
		t.Errorf("unknown value = %q, want critical", val.Value)
	}
	if miss := finding(t, rep, core.CategoryMissingTimestamp); !slices.Equal(miss.Missing, []string{"created", "updated"}) {
		t.Errorf("missing timestamps = %v, want both fields", miss.Missing)
	}
	dang := finding(t, rep, core.CategoryDanglingRef)
	if dang.Field != "depends_on" || dang.Target != "ghost0" {
		t.Errorf("dangling finding = %s %s, want depends_on ghost0", dang.Field, dang.Target)
	}
	if stuck := finding(t, rep, core.CategoryStuck); !slices.Equal(stuck.Cancelled, []string{"cnl001"}) {
		t.Errorf("stuck on %v, want the cancelled dependency", stuck.Cancelled)
	}
	// An issue that is its own parent is a one-issue cycle; a caller words it
	// from the single id rather than from a loop of two.
	if pc := finding(t, rep, core.CategoryParentCycle); !slices.Equal(pc.IDs, []string{"self01"}) {
		t.Errorf("parent cycle ids = %v, want just self01", pc.IDs)
	}
	// Nothing here is repairable but the drift, and nothing was written.
	if rep.Fixable() != 1 || rep.Fixed() != 0 {
		t.Errorf("fixable=%d fixed=%d, want 1 and 0 for a scan that does not repair", rep.Fixable(), rep.Fixed())
	}
}

// Filename drift is the one class a machine can resolve without guessing.
func TestDoctorFixRepairsFilenameDrift(t *testing.T) {
	root := newStore(t)
	seedAt(t, root, "drift-wrong.md", mkIssue("drft01", "New Title"))
	before := issueFile(t, root, "drift-wrong.md")

	rep := doctor(t, root, true)
	if rep.Problems() != 0 || rep.Fixed() != 1 {
		t.Errorf("problems=%d fixed=%d, want 0 and 1", rep.Problems(), rep.Fixed())
	}
	if !finding(t, rep, core.CategoryFilenameDrift).Fixed {
		t.Error("the drift finding should report itself fixed")
	}
	if got := issueFiles(t, root); !slices.Equal(got, []string{"drft01-new-title.md"}) {
		t.Errorf("files after --fix = %v, want only the canonical name", got)
	}
	// A rename moves the file; it never rewrites what the human wrote in it.
	if after := issueFile(t, root, "drft01-new-title.md"); after != before {
		t.Errorf("the repair rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	// The finding still names where the file was found: what it was called is the
	// fact worth reporting, and Canonical says what it is called now.
	if got := filepath.Base(finding(t, rep, core.CategoryFilenameDrift).Path()); got != "drift-wrong.md" {
		t.Errorf("fixed finding path = %q, want the name the drift was found under", got)
	}
}

// A repair frees the name the next drifted file wants, so passes repeat until
// none makes progress — a chain a single pass could only half fix.
func TestDoctorFixResolvesChainedDrift(t *testing.T) {
	root := newStore(t)
	// The first file the repair reaches wants a name the second still holds; only
	// once the second has moved can the first follow.
	seedAt(t, root, "aaa111-first.md", mkIssue("ccc333", "Third"))
	seedAt(t, root, "ccc333-third.md", mkIssue("bbb222", "Second"))

	rep := doctor(t, root, true)
	if rep.Fixed() != 2 || rep.Problems() != 0 {
		t.Errorf("fixed=%d problems=%d, want both drifts repaired", rep.Fixed(), rep.Problems())
	}
	if got := issueFiles(t, root); !slices.Equal(got, []string{"bbb222-second.md", "ccc333-third.md"}) {
		t.Errorf("files after --fix = %v, want both at their canonical names", got)
	}
}

// Two files each holding the name the other must take cannot both be repaired,
// and forcing either would destroy an issue. Both are left standing and reported.
func TestDoctorFixLeavesAMutualSwapStanding(t *testing.T) {
	root := newStore(t)
	seedAt(t, root, "bbb222-second.md", mkIssue("aaa111", "First"))
	seedAt(t, root, "aaa111-first.md", mkIssue("bbb222", "Second"))

	rep := doctor(t, root, true)
	if rep.Fixed() != 0 || rep.Problems() != 2 {
		t.Errorf("fixed=%d problems=%d, want nothing fixed and both still reported", rep.Fixed(), rep.Problems())
	}
	if got := issueFiles(t, root); !slices.Equal(got, []string{"aaa111-first.md", "bbb222-second.md"}) {
		t.Errorf("files after --fix = %v, want both untouched", got)
	}
}

// A validation error is a human's to resolve: --fix repairs the lint beside it
// and never touches the file itself.
func TestDoctorFixNeverTouchesAnUnusableFile(t *testing.T) {
	root := newStore(t)
	writeRaw(t, root, "bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n---\n")
	seedAt(t, root, "drift-wrong.md", mkIssue("drft01", "New Title"))
	before := issueFile(t, root, "bad-state.md")

	rep := doctor(t, root, true)
	if rep.Problems() != 1 {
		t.Errorf("problems = %d, want the 1 unusable file still standing", rep.Problems())
	}
	invalid := finding(t, rep, core.CategoryInvalid)
	if invalid.Fixable || invalid.Fixed {
		t.Errorf("invalid finding fixable=%v fixed=%v, want both false", invalid.Fixable, invalid.Fixed)
	}
	if after := issueFile(t, root, "bad-state.md"); after != before {
		t.Errorf("--fix modified an unusable file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// Renaming either file of a duplicated id onto the contested canonical name would
// clobber the other, so the drift repair is withheld for both and the clash is
// reported as the one problem a human must settle.
func TestDoctorReportsADuplicateIDAndFixesNothing(t *testing.T) {
	root := newStore(t)
	seedAt(t, root, "dup-a.md", mkIssue("dup001", "Dup A"))
	seedAt(t, root, "dup-b.md", mkIssue("dup001", "Dup B"))

	rep := doctor(t, root, true)
	dup := finding(t, rep, core.CategoryDuplicateID)
	if dup.Fixable || dup.Fixed {
		t.Errorf("duplicate id fixable=%v fixed=%v, want both false", dup.Fixable, dup.Fixed)
	}
	if dup.Path() != "" {
		t.Errorf("Path() = %q, want empty: the finding spans files", dup.Path())
	}
	if got := baseNames(dup.Paths); !slices.Equal(got, []string{"dup-a.md", "dup-b.md"}) {
		t.Errorf("duplicate id paths = %v, want both files sorted", got)
	}
	if got := categories(rep); slices.Contains(got, core.CategoryFilenameDrift) {
		t.Errorf("drift repair should be withheld for a contested id, got %v", got)
	}
	if got := issueFiles(t, root); !slices.Equal(got, []string{"dup-a.md", "dup-b.md"}) {
		t.Errorf("--fix moved a duplicate-id file: %v", got)
	}
}

// Resemblance to a known field is only ever a guess — a deliberate `status` key
// sits two edits from `state` — so the class is reported and never counted or
// removed.
func TestDoctorTreatsAnUnknownKeyAsAdvisory(t *testing.T) {
	root := newStore(t)
	seed(t, root, withCustom(mkIssue("note01", "Deliberate Custom Key"), map[string]any{"status": "shipping"}))
	before := issueFile(t, root, "note01-deliberate-custom-key.md")

	rep := doctor(t, root, true)
	if rep.Problems() != 0 || rep.Advisories() != 1 {
		t.Errorf("problems=%d advisories=%d, want 0 and 1", rep.Problems(), rep.Advisories())
	}
	key := finding(t, rep, core.CategoryUnknownKey)
	if !key.Category.Advisory() || key.Fixable {
		t.Errorf("unknown key advisory=%v fixable=%v, want true and false", key.Category.Advisory(), key.Fixable)
	}
	if after := issueFile(t, root, "note01-deliberate-custom-key.md"); after != before {
		t.Error("--fix removed a custom key; that is data loss, not tidying")
	}
}

// An advisory note never hides a real problem, nor softens the count beside it.
func TestDoctorCountsAdvisoriesApartFromProblems(t *testing.T) {
	root := newStore(t)
	writeRaw(t, root, "bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n---\n")
	seed(t, root, withCustom(mkIssue("note01", "Custom Key"), map[string]any{"status": "shipping"}))

	rep := doctor(t, root, false)
	if rep.Problems() != 1 || rep.Advisories() != 1 || len(rep.Findings) != 2 {
		t.Errorf("problems=%d advisories=%d findings=%d, want 1, 1 and 2",
			rep.Problems(), rep.Advisories(), len(rep.Findings))
	}
}

// A key far from every known field is a supported feature, not a typo.
func TestDoctorAllowsDeliberateCustomFields(t *testing.T) {
	root := newStore(t)
	seed(t, root, withCustom(mkIssue("cust01", "Has Custom Fields"), map[string]any{"sprint": 7, "estimate": "3d"}))

	if rep := doctor(t, root, false); len(rep.Findings) != 0 {
		t.Errorf("deliberate custom fields were flagged: %v", categories(rep))
	}
}

// A closed issue has nothing left to wait for, so a cancelled dependency does not
// leave it stuck.
func TestDoctorReportsStuckOnlyForOpenIssues(t *testing.T) {
	root := newStore(t)
	seed(t, root, withState(mkIssue("cnl001", "Cancelled Dep"), issue.StateCancelled))
	seed(t, root, withState(withDeps(mkIssue("done01", "Finished"), "cnl001"), issue.StateDone))
	seed(t, root, withState(withDeps(mkIssue("prog01", "In Flight"), "cnl001"), issue.StateInProgress))

	rep := doctor(t, root, false)
	var stuck []string
	for _, f := range rep.Findings {
		if f.Category == core.CategoryStuck {
			stuck = append(stuck, f.IDs...)
		}
	}
	if !slices.Equal(stuck, []string{"prog01"}) {
		t.Errorf("stuck findings for %v, want only the in-progress issue", stuck)
	}
}

func TestDoctorNeedsAStore(t *testing.T) {
	if _, err := core.Open(t.TempDir()); err == nil {
		t.Error("Open outside a store should fail before any scan")
	}
}

// --- helpers ---

// doctor runs a health scan over the store at root, failing the test if the scan
// itself cannot run.
func doctor(t *testing.T, root string, fix bool) core.Report {
	t.Helper()
	rep, err := openAt(t, root).Doctor(fix)
	if err != nil {
		t.Fatalf("Doctor(fix=%v): %v", fix, err)
	}
	return rep
}

// finding returns the first finding of a category, failing the test if there is
// none.
func finding(t *testing.T, rep core.Report, cat core.Category) core.Finding {
	t.Helper()
	for _, f := range rep.Findings {
		if f.Category == cat {
			return f
		}
	}
	t.Fatalf("no %s finding in %v", cat, categories(rep))
	return core.Finding{}
}

// categories lists the category of every finding, in report order.
func categories(rep core.Report) []core.Category {
	out := make([]core.Category, len(rep.Findings))
	for i, f := range rep.Findings {
		out[i] = f.Category
	}
	return out
}

func countCategory(cats []core.Category, want core.Category) int {
	n := 0
	for _, c := range cats {
		if c == want {
			n++
		}
	}
	return n
}

// seedAt writes an issue under an arbitrary file name, so a test can produce a
// file whose name has drifted from what its frontmatter dictates — or two files
// that claim one id.
func seedAt(t *testing.T, root, name string, iss issue.Issue) {
	t.Helper()
	data, err := issue.Marshal(iss)
	if err != nil {
		t.Fatalf("marshal seed %s: %v", iss.ID, err)
	}
	writeRaw(t, root, name, string(data))
}

func withCustom(iss issue.Issue, custom map[string]any) issue.Issue {
	iss.Custom = custom
	return iss
}

// undated strips both timestamps, the shape of an issue authored by hand.
func undated(iss issue.Issue) issue.Issue {
	iss.Created, iss.Updated = time.Time{}, time.Time{}
	return iss
}

// issueFile returns the bytes of an issue file by name, so a test can prove a
// scan left it untouched.
func issueFile(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".beaver", "issues", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func baseNames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}
