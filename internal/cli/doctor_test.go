package cli_test

import (
	"slices"
	"strings"
	"testing"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

// AC: a clean store reports healthy — exit 0, no findings, and (in JSON) an ok flag
// a consumer can read instead of the exit code.
func TestDoctorCleanStoreIsHealthy(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "good11", "First", issue.StateTodo, beavertest.DefaultNow)
	seed(t, h, "good22", "Second", issue.StateDone, beavertest.DefaultNow)

	rep, r := doctorJSON(t, h)
	if r.Code != 0 {
		t.Fatalf("doctor exit = %d, want 0 on a clean store\nstderr: %s", r.Code, r.Stderr)
	}
	if rep["ok"] != true {
		t.Errorf("ok = %v, want true", rep["ok"])
	}
	if got := findingCategories(t, rep); len(got) != 0 {
		t.Errorf("clean store reported findings: %v", got)
	}

	human := h.MustRun("doctor", "--format", "human")
	if !strings.Contains(human.Stdout, "No problems found") {
		t.Errorf("human clean output = %q, want a no-problems line", human.Stdout)
	}
}

// AC: doctor detects and reports every problem class, and exits non-zero when any
// exist. Each class is seeded so it stands alone (its issues produce only their own
// finding), and all coexist in one store, so this asserts both detection and that
// the classes do not mask one another.
func TestDoctorDetectsEachProblemClass(t *testing.T) {
	h := beavertest.New(t).Init()

	// A hard validation error: reported, never fixed (ADR 0005).
	h.WriteFile("issues/bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n"+stamps+"---\n")
	// Filename drift: a valid issue under a name that is not its canonical <id>-<slug>.
	seedAt(t, h, "drift-wrong.md", "drft01", "Drift Title", issue.StateTodo)
	// A frontmatter key that looks like a typo of a known field (ADR 0014).
	seedCustom(t, h, "unkey1", "Unknown Key", map[string]any{"assigne": "someone"})
	// Dangling references: a depends_on and a parent to an id no issue holds.
	seedDep(t, h, "dang01", "Dangling Depends", issue.StateTodo, []string{"ghost0"}, "")
	seedDep(t, h, "dang02", "Dangling Parent", issue.StateTodo, nil, "ghost0")
	// A dependency cycle: two issues each waiting on the other.
	seedDep(t, h, "cyc0aa", "Cycle A", issue.StateTodo, []string{"cyc0bb"}, "")
	seedDep(t, h, "cyc0bb", "Cycle B", issue.StateTodo, []string{"cyc0aa"}, "")
	// A parent cycle: two issues each claiming the other as parent.
	seedDep(t, h, "pcy0aa", "Parent Cycle A", issue.StateTodo, nil, "pcy0bb")
	seedDep(t, h, "pcy0bb", "Parent Cycle B", issue.StateTodo, nil, "pcy0aa")
	// Stuck on a cancelled dependency: an open issue whose dependency can never be met.
	seedDep(t, h, "cnl001", "Cancelled Dep", issue.StateCancelled, nil, "")
	seedDep(t, h, "stk001", "Stuck Issue", issue.StateTodo, []string{"cnl001"}, "")
	// Two files claiming one id — a half-merged clash validation cannot catch.
	seedAt(t, h, "dup-a.md", "dup001", "Dup A", issue.StateTodo)
	seedAt(t, h, "dup-b.md", "dup001", "Dup B", issue.StateTodo)

	rep, r := doctorJSON(t, h)
	if r.Code == 0 {
		t.Fatalf("doctor exit = 0, want non-zero when problems exist\nstdout: %s", r.Stdout)
	}
	if rep["ok"] != false {
		t.Errorf("ok = %v, want false", rep["ok"])
	}
	got := findingCategories(t, rep)
	for _, want := range []string{
		"invalid",
		"filename_drift",
		"unknown_key",
		"dangling_reference",
		"dependency_cycle",
		"parent_cycle",
		"stuck",
		"duplicate_id",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %q finding; got %v", want, got)
		}
	}
	// The two dangling edges (depends_on and parent) are each their own finding.
	if n := count(got, "dangling_reference"); n != 2 {
		t.Errorf("dangling_reference findings = %d, want 2 (a depends_on and a parent)", n)
	}
}

// AC: doctor --fix repairs filename drift — the one mechanically safe lint. The file
// is renamed to its canonical name on disk, the finding is marked fixed, and with no
// other problem the store comes out clean (exit 0).
func TestDoctorFixRepairsFilenameDrift(t *testing.T) {
	h := beavertest.New(t).Init()
	seedAt(t, h, "drift-wrong.md", "drft01", "New Title", issue.StateTodo)

	rep, r := doctorJSON(t, h, "--fix")
	if r.Code != 0 {
		t.Fatalf("doctor --fix exit = %d, want 0 after repairing the only problem\nstderr: %s", r.Code, r.Stderr)
	}
	if rep["ok"] != true || rep["fixed"] != float64(1) {
		t.Errorf("ok=%v fixed=%v, want true and 1", rep["ok"], rep["fixed"])
	}
	drift := findingWhere(t, rep, "filename_drift")
	if drift["fixed"] != true {
		t.Errorf("drift finding fixed = %v, want true", drift["fixed"])
	}
	if files := h.IssueFiles(); !slices.Contains(files, "drft01-new-title.md") || slices.Contains(files, "drift-wrong.md") {
		t.Errorf("file not renamed to canonical: %v", files)
	}
}

// AC: --fix repairs the lint but reports validation errors rather than auto-fixing
// them. The drifted file is renamed; the invalid file is left untouched and still
// reported, so the run exits non-zero for the problem a human must resolve.
func TestDoctorFixLeavesValidationErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n"+stamps+"---\n")
	seedAt(t, h, "drift-wrong.md", "drft01", "New Title", issue.StateTodo)

	rep, r := doctorJSON(t, h, "--fix")
	if r.Code == 0 {
		t.Fatalf("doctor --fix exit = 0, want non-zero while an invalid file remains")
	}
	if findingWhere(t, rep, "filename_drift")["fixed"] != true {
		t.Errorf("drift should have been fixed")
	}
	if findingWhere(t, rep, "invalid")["fixed"] != false {
		t.Errorf("a validation error must never be auto-fixed")
	}
	files := h.IssueFiles()
	if !slices.Contains(files, "drft01-new-title.md") {
		t.Errorf("drift not repaired: %v", files)
	}
	if !slices.Contains(files, "bad-state.md") {
		t.Errorf("invalid file was touched: %v", files)
	}
}

// AC / ADR 0014: --fix never removes an unknown frontmatter key — removal is data
// loss, not tidying. The typo'd key is reported (never fixed) and remains in the file
// after --fix. The finding is advisory: resemblance to a known field is only ever a
// guess, so it does not fail the run.
func TestDoctorFixNeverRemovesUnknownKey(t *testing.T) {
	h := beavertest.New(t).Init()
	seedCustom(t, h, "unkey1", "Typo Key", map[string]any{"assigne": "bob"})

	rep, r := doctorJSON(t, h, "--fix")
	if r.Code != 0 {
		t.Fatalf("doctor --fix exit = %d, want 0: an unknown key is advisory, not a problem\nstdout: %s", r.Code, r.Stdout)
	}
	key := findingWhere(t, rep, "unknown_key")
	if key["fixable"] != false || key["fixed"] != false {
		t.Errorf("unknown key finding fixable=%v fixed=%v, want both false", key["fixable"], key["fixed"])
	}
	if body := h.ReadFile("issues/" + issue.FileName("unkey1", issue.Slug("Typo Key"))); !strings.Contains(body, "assigne") {
		t.Errorf("--fix removed the unknown key; file:\n%s", body)
	}
}

// A likely-typo'd key is worth a human's eye but is only ever a guess — a deliberate
// custom key like `status` sits within typo distance of `state` — so the unknown-key
// class is advisory: reported (with the advisory flag in JSON, and worded as a note
// in the human report) while the store still counts as healthy (ok, exit 0, zero
// problems). A first-class custom field must never fail doctor forever (ADR 0014).
func TestDoctorUnknownKeyIsAdvisory(t *testing.T) {
	h := beavertest.New(t).Init()
	seedCustom(t, h, "note01", "Deliberate Custom Key", map[string]any{"status": "shipping"})

	rep, r := doctorJSON(t, h)
	if r.Code != 0 {
		t.Fatalf("doctor exit = %d, want 0: an advisory note is not a problem\nstdout: %s", r.Code, r.Stdout)
	}
	if rep["ok"] != true || rep["problems"] != float64(0) {
		t.Errorf("ok=%v problems=%v, want true and 0", rep["ok"], rep["problems"])
	}
	key := findingWhere(t, rep, "unknown_key")
	if key["advisory"] != true {
		t.Errorf("unknown key advisory = %v, want true", key["advisory"])
	}
	if msg, _ := key["message"].(string); !strings.Contains(msg, "status") {
		t.Errorf("message should name the suspect key: %q", msg)
	}

	human := h.MustRun("doctor", "--format", "human")
	if !strings.Contains(human.Stdout, "1 advisory note") || strings.Contains(human.Stdout, "problem") {
		t.Errorf("human output should headline an advisory note and no problems:\n%s", human.Stdout)
	}
}

// A genuine problem and an advisory note coexist without muddying each other: the
// problem alone drives the exit code and counts, while the note stays visible in the
// findings and the headline names both.
func TestDoctorAdvisoryDoesNotMaskProblems(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n"+stamps+"---\n")
	seedCustom(t, h, "note01", "Custom Key", map[string]any{"status": "shipping"})

	rep, r := doctorJSON(t, h)
	if r.Code == 0 {
		t.Fatalf("doctor exit = 0, want non-zero: a real problem remains")
	}
	if rep["ok"] != false || rep["problems"] != float64(1) {
		t.Errorf("ok=%v problems=%v, want false and 1 (the advisory note not counted)", rep["ok"], rep["problems"])
	}
	human := h.Run("doctor", "--format", "human")
	if !strings.Contains(human.Stdout, "1 problem and 1 advisory note") {
		t.Errorf("human headline should count them apart:\n%s", human.Stdout)
	}
}

// A priority that is not one of the four levels loads fine (validation stays
// narrow, ADR 0005) but silently matches no --priority filter, so doctor flags it
// as an unknown value — and never fixes it, since mapping it to a real level
// would be guessing.
func TestDoctorFlagsUnknownPriorityValue(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/pri001-typo-priority.md",
		"---\nid: pri001\ntitle: Typo Priority\nstate: todo\npriority: critical\n"+stamps+"---\n")

	rep, r := doctorJSON(t, h, "--fix")
	if r.Code == 0 {
		t.Fatalf("doctor --fix exit = 0, want non-zero: an unknown priority is not auto-fixable")
	}
	f := findingWhere(t, rep, "unknown_value")
	if f["fixable"] != false || f["fixed"] != false {
		t.Errorf("unknown value finding fixable=%v fixed=%v, want both false", f["fixable"], f["fixed"])
	}
	msg, _ := f["message"].(string)
	if !strings.Contains(msg, "critical") || !strings.Contains(msg, "priority") {
		t.Errorf("message should name the bad priority value: %q", msg)
	}
	if body := h.ReadFile("issues/pri001-typo-priority.md"); !strings.Contains(body, "priority: critical") {
		t.Errorf("--fix must not touch the priority; file:\n%s", body)
	}
}

// An issue with no created/updated timestamp is usable (ADR 0005) but sorts as the
// oldest in every list, so doctor surfaces it — and never invents a date.
func TestDoctorFlagsMissingTimestamps(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/nots01-no-stamps.md", "---\nid: nots01\ntitle: No Stamps\nstate: todo\n---\n")

	rep, r := doctorJSON(t, h)
	if r.Code == 0 {
		t.Fatalf("doctor exit = 0, want non-zero for a missing timestamp")
	}
	f := findingWhere(t, rep, "missing_timestamp")
	if f["fixable"] != false {
		t.Errorf("missing timestamp finding fixable=%v, want false (no guessed dates)", f["fixable"])
	}
	if msg, _ := f["message"].(string); !strings.Contains(msg, "created and updated") {
		t.Errorf("message should name the missing fields: %q", msg)
	}
}

// A mutating command on a timestamp-less issue must not bake in the year-1 zero
// value: created stays honestly absent (doctor keeps flagging it), updated is set
// by the mutation, and JSON renders the absent created as null (ADR 0013).
func TestMutationLeavesMissingCreatedAbsent(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/nots01-no-stamps.md", "---\nid: nots01\ntitle: No Stamps\nstate: todo\n---\n")

	h.MustRun("label", "nots01", "sprint-7")

	file := h.ReadFile("issues/nots01-no-stamps.md")
	if strings.Contains(file, "0001-01-01") || strings.Contains(file, "created:") {
		t.Errorf("mutation baked a timestamp into a created-less issue:\n%s", file)
	}
	if !strings.Contains(file, "updated: ") {
		t.Errorf("mutation should set updated:\n%s", file)
	}
	out := h.DecodeJSON(h.MustRun("show", "nots01", "--format", "json").Stdout)
	if out["created"] != nil {
		t.Errorf("JSON created = %v, want null for an absent timestamp", out["created"])
	}
	rep, _ := doctorJSON(t, h)
	if msg, _ := findingWhere(t, rep, "missing_timestamp")["message"].(string); !strings.Contains(msg, "created") || strings.Contains(msg, "updated") {
		t.Errorf("doctor should now flag only created as missing: %q", msg)
	}
}

// A deliberate custom field is a first-class, supported feature (ADR 0014), not a
// problem: it is near no known field, so doctor does not flag it and a store that
// uses custom fields can still be perfectly healthy.
func TestDoctorAllowsDeliberateCustomFields(t *testing.T) {
	h := beavertest.New(t).Init()
	seedCustom(t, h, "cust01", "Has Custom Fields", map[string]any{"sprint": 7, "estimate": "3d"})

	rep, r := doctorJSON(t, h)
	if r.Code != 0 {
		t.Fatalf("doctor exit = %d, want 0: deliberate custom fields are not problems\nstdout: %s", r.Code, r.Stdout)
	}
	if got := findingCategories(t, rep); len(got) != 0 {
		t.Errorf("deliberate custom fields were flagged: %v", got)
	}
}

// A parent chain that loops back on itself has no root, so no hierarchy could ever
// render it — the degenerate case being an issue that names itself as its own
// parent, which show would list among its own children. Doctor reports it (never
// fixes it: which edge to drop is a human's call), like a dependency cycle.
func TestDoctorFlagsParentCycles(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "self01", "Own Parent", issue.StateTodo, nil, "self01")

	rep, r := doctorJSON(t, h)
	if r.Code == 0 {
		t.Fatalf("doctor exit = 0, want non-zero for a parent cycle")
	}
	f := findingWhere(t, rep, "parent_cycle")
	if f["fixable"] != false {
		t.Errorf("parent cycle fixable = %v, want false", f["fixable"])
	}
	if msg, _ := f["message"].(string); !strings.Contains(msg, "self01") || !strings.Contains(msg, "own parent") {
		t.Errorf("message should call out the self-parent: %q", msg)
	}
}

// A duplicate id is reported and never auto-fixed: renaming one file onto the other's
// canonical name would clobber it, so --fix leaves both files exactly where they are
// and the clash is left for a human. Because the id is contested, no filename-drift
// repair is offered for those files either.
func TestDoctorDuplicateIDIsReportedNotFixed(t *testing.T) {
	h := beavertest.New(t).Init()
	seedAt(t, h, "dup-a.md", "dup001", "Dup A", issue.StateTodo)
	seedAt(t, h, "dup-b.md", "dup001", "Dup B", issue.StateTodo)

	rep, r := doctorJSON(t, h, "--fix")
	if r.Code == 0 {
		t.Fatalf("doctor --fix exit = 0, want non-zero while a duplicate id stands")
	}
	dup := findingWhere(t, rep, "duplicate_id")
	if dup["fixable"] != false || dup["fixed"] != false {
		t.Errorf("duplicate id fixable=%v fixed=%v, want both false", dup["fixable"], dup["fixed"])
	}
	if got := findingCategories(t, rep); slices.Contains(got, "filename_drift") {
		t.Errorf("filename drift should be suppressed for a contested id, got %v", got)
	}
	if files := h.IssueFiles(); !slices.Contains(files, "dup-a.md") || !slices.Contains(files, "dup-b.md") {
		t.Errorf("--fix must not move a duplicate-id file (no clobber): %v", files)
	}
}

// AC: output is human or JSON per the standard auto-detection. The human report
// headlines the count, lists each problem by class, and points at --fix for the
// fixable ones.
func TestDoctorHumanOutput(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/bad-state.md", "---\nid: bad222\ntitle: Bad\nstate: archived\n"+stamps+"---\n")
	seedAt(t, h, "drift-wrong.md", "drft01", "New Title", issue.StateTodo)

	r := h.Run("doctor", "--format", "human")
	if r.Code == 0 {
		t.Fatalf("doctor exit = 0, want non-zero")
	}
	for _, want := range []string{"Found 2 problems", "invalid", "filename drift", "1 of these can be fixed automatically — run `beaver doctor --fix`"} {
		if !strings.Contains(r.Stdout, want) {
			t.Errorf("human output missing %q:\n%s", want, r.Stdout)
		}
	}
}

// doctor needs a store, like every other command: run outside one and it fails with
// the not-found exit code and the init hint, rather than pretending all is well.
func TestDoctorRequiresAStore(t *testing.T) {
	h := beavertest.New(t) // not initialized
	r := h.Run("doctor")
	if r.Code != 3 { // exitNotFound
		t.Fatalf("doctor exit = %d, want 3 (no store)\nstderr: %s", r.Code, r.Stderr)
	}
	if !strings.Contains(r.Stderr, "init") {
		t.Errorf("stderr should suggest `beaver init`:\n%s", r.Stderr)
	}
}

// The JSON report is a clean object on stdout — nothing leaks onto it (doctor reports
// invalid files in the report itself, not through the store's stderr warnings), so an
// agent parses stdout directly (ADR 0013).
func TestDoctorJSONStaysCleanOnStdout(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/broken.md", "not an issue file\n")

	rep, r := doctorJSON(t, h)
	if r.Code == 0 {
		t.Fatalf("doctor exit = 0, want non-zero: an invalid file is a problem")
	}
	if !strings.Contains(rep["findings"].([]any)[0].(map[string]any)["message"].(string), "broken.md") {
		t.Errorf("invalid file not reported in the JSON report: %v", rep["findings"])
	}
	if strings.Contains(r.Stdout, "skipping") {
		t.Errorf("a store warning leaked onto the JSON stdout:\n%s", r.Stdout)
	}
}

// --- helpers ---

// doctorJSON runs doctor (JSON, since the harness is non-interactive) and returns the
// decoded report object together with the raw result for exit-code assertions.
func doctorJSON(t *testing.T, h *beavertest.Harness, args ...string) (map[string]any, beavertest.Result) {
	t.Helper()
	r := h.Run(append([]string{"doctor"}, args...)...)
	return h.DecodeJSON(r.Stdout), r
}

// findingCategories lists the category of every finding in a decoded report, in order.
func findingCategories(t *testing.T, rep map[string]any) []string {
	t.Helper()
	raw, ok := rep["findings"].([]any)
	if !ok {
		t.Fatalf("report has no findings array: %v", rep)
	}
	var cats []string
	for _, f := range raw {
		m, ok := f.(map[string]any)
		if !ok {
			t.Fatalf("finding is not an object: %v", f)
		}
		cats = append(cats, m["category"].(string))
	}
	return cats
}

// findingWhere returns the first finding of the given category, failing if none.
func findingWhere(t *testing.T, rep map[string]any, cat string) map[string]any {
	t.Helper()
	raw, _ := rep["findings"].([]any)
	for _, f := range raw {
		if m, ok := f.(map[string]any); ok && m["category"] == cat {
			return m
		}
	}
	t.Fatalf("no %q finding in report: %v", cat, rep["findings"])
	return nil
}

func count[T comparable](xs []T, want T) int {
	n := 0
	for _, x := range xs {
		if x == want {
			n++
		}
	}
	return n
}

// seedAt writes a valid issue under an arbitrary file name, so a test can produce a
// file whose name has drifted from the canonical <id>-<slug> its frontmatter dictates
// (or two files that share an id).
func seedAt(t *testing.T, h *beavertest.Harness, filename, id, title string, state issue.State) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: state,
		Created: beavertest.DefaultNow, Updated: beavertest.DefaultNow,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+filename, string(data))
}

// seedCustom writes a valid issue carrying user-added (custom) frontmatter keys at its
// canonical file name, so a test can exercise doctor's handling of unknown keys
// without also tripping the filename-drift check.
func seedCustom(t *testing.T, h *beavertest.Harness, id, title string, custom map[string]any) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: issue.StateTodo,
		Created: beavertest.DefaultNow, Updated: beavertest.DefaultNow,
		Custom: custom,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+issue.FileName(id, issue.Slug(title)), string(data))
}
