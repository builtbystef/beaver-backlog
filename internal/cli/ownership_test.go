package cli_test

import (
	"strings"
	"testing"
	"time"

	"beaver/internal/beavertest"
	"beaver/internal/issue"
)

// AC: claim sets `assignee` to the current actor and leaves `state` untouched —
// claiming reserves an issue, it does not start it (ADR 0009). The write bumps
// `updated`, and the change is on disk, not just echoed.
func TestClaimSetsAssigneeLeavesState(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Some work", issue.StateTodo, beavertest.DefaultNow)
	h.Clock.Advance(time.Hour)

	out := h.DecodeJSON(h.MustRun("claim", "iss001", "--as", "alice").Stdout)
	if out["assignee"] != "alice" {
		t.Errorf("assignee = %v, want alice", out["assignee"])
	}
	if out["state"] != "todo" {
		t.Errorf("state = %v, want unchanged todo (claim reserves, never starts)", out["state"])
	}

	shown := h.DecodeJSON(h.MustRun("show", "iss001").Stdout)
	if shown["assignee"] != "alice" || shown["state"] != "todo" {
		t.Errorf("persisted assignee/state = %v/%v, want alice/todo", shown["assignee"], shown["state"])
	}
	if shown["updated"] == "2026-06-27T18:30:00Z" {
		t.Errorf("updated = %v, want bumped by the claim write", shown["updated"])
	}
}

// AC: the guard refuses an issue already assigned to a different actor, naming the
// owner; --force steals it. The refusal writes nothing (best-effort local guard,
// ADR 0009), so the file is byte-for-byte unchanged.
func TestClaimRefusesAnothersIssueUnlessForce(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Owned work", issue.StateTodo, "bob")
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour) // a stray write would bump updated

	r := h.Run("claim", "iss001", "--as", "alice")
	if r.Code != 2 {
		t.Errorf("claim of another's issue exit = %d, want 2 (usage)", r.Code)
	}
	if !strings.Contains(r.Stderr, "bob") || !strings.Contains(r.Stderr, "--force") {
		t.Errorf("refusal should name the owner and mention --force:\n%s", r.Stderr)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("refused claim modified the file (the guard must not write):\nbefore:\n%s\nafter:\n%s", before, after)
	}

	out := h.DecodeJSON(h.MustRun("claim", "iss001", "--as", "alice", "--force").Stdout)
	if out["assignee"] != "alice" {
		t.Errorf("assignee after --force = %v, want alice (stolen)", out["assignee"])
	}
}

// AC: re-claiming one's own issue is an idempotent no-op — success, but no rewrite,
// so `updated` is not churned and the file bytes are unchanged.
func TestReclaimingOwnIsIdempotentNoop(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Mine already", issue.StateTodo, "alice")
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour) // a rewrite would bump updated and change bytes

	r := h.Run("claim", "iss001", "--as", "alice")
	if r.Code != 0 {
		t.Errorf("re-claiming own exit = %d, want 0 (idempotent)", r.Code)
	}
	out := h.DecodeJSON(r.Stdout)
	if out["assignee"] != "alice" {
		t.Errorf("assignee = %v, want alice", out["assignee"])
	}
	if out["updated"] != "2026-06-27T18:30:00Z" {
		t.Errorf("updated = %v, want unchanged (no bump on a no-op)", out["updated"])
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("re-claiming own rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// claim resolves the actor through the one identity chain every attributing command
// uses, so with no --as it honors BUSY_BEAVER_ACTOR (the agent/CI override).
func TestClaimResolvesActorFromEnv(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Env["BUSY_BEAVER_ACTOR"] = "ci-bot"
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("claim", "iss001").Stdout)
	if out["assignee"] != "ci-bot" {
		t.Errorf("assignee = %v, want ci-bot resolved from BUSY_BEAVER_ACTOR", out["assignee"])
	}
}

// AC: assign sets a named assignee. It is explicit delegation, so unlike claim it
// carries no guard — it reassigns already-owned work without --force — and never
// touches state. Assigning the current assignee is an idempotent no-op.
func TestAssignDelegatesWithoutGuard(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Held by bob", issue.StateInProgress, "bob")

	out := h.DecodeJSON(h.MustRun("assign", "iss001", "carol").Stdout)
	if out["assignee"] != "carol" {
		t.Errorf("assignee = %v, want carol (delegated over bob without --force)", out["assignee"])
	}
	if out["state"] != "in-progress" {
		t.Errorf("state = %v, want unchanged in-progress (assign never changes state)", out["state"])
	}

	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour)
	if r := h.Run("assign", "iss001", "carol"); r.Code != 0 {
		t.Errorf("re-assigning the same actor exit = %d, want 0 (idempotent)", r.Code)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("no-op assign rewrote the file")
	}
}

// AC: release clears the assignee, leaving state untouched; releasing an
// already-unassigned issue is an idempotent no-op.
func TestReleaseClearsAssignee(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Owned", issue.StateTodo, "bob")

	out := h.DecodeJSON(h.MustRun("release", "iss001").Stdout)
	if out["assignee"] != nil {
		t.Errorf("assignee = %v, want null after release", out["assignee"])
	}
	if out["state"] != "todo" {
		t.Errorf("state = %v, want unchanged todo (release never changes state)", out["state"])
	}

	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour)
	if r := h.Run("release", "iss001"); r.Code != 0 {
		t.Errorf("releasing an unassigned issue exit = %d, want 0 (idempotent)", r.Code)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("no-op release rewrote the file")
	}
}

// AC: start moves a todo to in-progress and auto-claims it when unowned — the one
// implicit assignment (ADR 0009). The change is persisted.
func TestStartMovesToInProgressAndAutoClaims(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Unowned todo", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("start", "iss001", "--as", "alice").Stdout)
	if out["state"] != "in-progress" {
		t.Errorf("state = %v, want in-progress", out["state"])
	}
	if out["assignee"] != "alice" {
		t.Errorf("assignee = %v, want alice (auto-claimed as it started)", out["assignee"])
	}

	shown := h.DecodeJSON(h.MustRun("show", "iss001").Stdout)
	if shown["state"] != "in-progress" || shown["assignee"] != "alice" {
		t.Errorf("persisted = %v/%v, want in-progress/alice", shown["state"], shown["assignee"])
	}
}

// start auto-claims an unowned issue even when the state does not move: an
// already-in-progress but ownerless issue is claimed rather than left unattributed.
func TestStartClaimsUnownedInProgress(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Ownerless active", issue.StateInProgress, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("start", "iss001", "--as", "alice").Stdout)
	if out["assignee"] != "alice" || out["state"] != "in-progress" {
		t.Errorf("start of an unowned in-progress issue = %v/%v, want alice/in-progress", out["assignee"], out["state"])
	}
}

// AC: start guards ownership exactly as claim does — it refuses an issue held by a
// different actor unless --force steals it. A refusal moves neither the assignee nor
// the state (nothing is written).
func TestStartGuardsAndStealsLikeClaim(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Bob's work", issue.StateTodo, "bob")
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour)

	r := h.Run("start", "iss001", "--as", "alice")
	if r.Code != 2 {
		t.Errorf("start of another's issue exit = %d, want 2 (usage)", r.Code)
	}
	if !strings.Contains(r.Stderr, "bob") || !strings.Contains(r.Stderr, "--force") {
		t.Errorf("refusal should name the owner and mention --force:\n%s", r.Stderr)
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("refused start modified the file (the state must not move either):\n%s", after)
	}

	out := h.DecodeJSON(h.MustRun("start", "iss001", "--as", "alice", "--force").Stdout)
	if out["assignee"] != "alice" || out["state"] != "in-progress" {
		t.Errorf("after --force start = %v/%v, want alice/in-progress (stolen and started)", out["assignee"], out["state"])
	}
}

// A closed issue cannot be started; start refuses it (exit 2) and points at reopen,
// the same graceful refusal the transition verbs give, and writes nothing.
func TestStartRejectsClosedIssue(t *testing.T) {
	for _, st := range []issue.State{issue.StateDone, issue.StateCancelled} {
		t.Run(string(st), func(t *testing.T) {
			h := beavertest.New(t).Init()
			seed(t, h, "iss001", "Closed", st, beavertest.DefaultNow)
			file := "issues/" + h.IssueFiles()[0]
			before := h.ReadFile(file)
			h.Clock.Advance(time.Hour)

			r := h.Run("start", "iss001", "--as", "alice")
			if r.Code != 2 {
				t.Errorf("start of a %s issue exit = %d, want 2 (usage)", st, r.Code)
			}
			if !strings.Contains(r.Stderr, "iss001") || !strings.Contains(strings.ToLower(r.Stderr), "reopen") {
				t.Errorf("rejection should name the issue and suggest reopen:\n%s", r.Stderr)
			}
			if after := h.ReadFile(file); after != before {
				t.Errorf("rejected start modified the %s file", st)
			}
		})
	}
}

// Starting one's own in-progress issue is an idempotent no-op: success, no rewrite,
// no `updated` churn.
func TestStartOnOwnInProgressIsNoop(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Already going", issue.StateInProgress, "alice")
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour) // a rewrite would bump updated

	r := h.Run("start", "iss001", "--as", "alice")
	if r.Code != 0 {
		t.Errorf("starting own in-progress issue exit = %d, want 0 (idempotent)", r.Code)
	}
	out := h.DecodeJSON(r.Stdout)
	if out["state"] != "in-progress" || out["assignee"] != "alice" {
		t.Errorf("state/assignee = %v/%v, want in-progress/alice", out["state"], out["assignee"])
	}
	if out["updated"] != "2026-06-27T18:30:00Z" {
		t.Errorf("updated = %v, want unchanged (no bump on a no-op)", out["updated"])
	}
	if after := h.ReadFile(file); after != before {
		t.Errorf("no-op start rewrote the file")
	}
}

// AC: the assignee survives `done` — retained as the record of who completed the
// work (ADR 0009). claim, start, then done, and the assignee is still there.
func TestAssigneeSurvivesDone(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "Work to finish", issue.StateTodo, beavertest.DefaultNow)

	h.MustRun("claim", "iss001", "--as", "alice")
	h.MustRun("start", "iss001", "--as", "alice")
	out := h.DecodeJSON(h.MustRun("done", "iss001").Stdout)
	if out["state"] != "done" {
		t.Fatalf("state = %v, want done", out["state"])
	}
	if out["assignee"] != "alice" {
		t.Errorf("assignee after done = %v, want alice retained (ADR 0009)", out["assignee"])
	}

	shown := h.DecodeJSON(h.MustRun("show", "iss001").Stdout)
	if shown["assignee"] != "alice" {
		t.Errorf("persisted assignee after done = %v, want alice", shown["assignee"])
	}
}

// AC: start is dependency-aware but never gated by it. When it moves a not-ready
// todo to in-progress, it warns (non-fatally), naming the unmet dependencies, and
// still starts the issue — the advisory stance of ADR 0011.
func TestStartWarnsOnUnmetDependenciesButStarts(t *testing.T) {
	h := beavertest.New(t).Init()
	// A todo dependent waiting on one in-progress and one missing dependency.
	seedDep(t, h, "dep001", "Prerequisite", issue.StateInProgress, nil, "")
	seedDep(t, h, "iss001", "The dependent", issue.StateTodo, []string{"dep001", "gone99"}, "")

	r := h.Run("start", "iss001", "--as", "alice")
	if r.Code != 0 {
		t.Fatalf("start of a not-ready issue exit = %d, want 0 — the dependency warning is advisory, never fatal (ADR 0011)", r.Code)
	}
	out := h.DecodeJSON(r.Stdout)
	if out["state"] != "in-progress" || out["assignee"] != "alice" {
		t.Errorf("not-ready issue = %v/%v, want in-progress/alice (started despite the warning)", out["state"], out["assignee"])
	}
	// The warning is loud on stderr and names both unmet dependencies.
	for _, want := range []string{"warning", "not ready", "dep001", "gone99"} {
		if !strings.Contains(r.Stderr, want) {
			t.Errorf("dependency warning missing %q:\n%s", want, r.Stderr)
		}
	}
}

// A ready issue — todo with every dependency done — starts with no dependency
// warning, so the advisory noise appears only when it is warranted.
func TestStartReadyIssueDoesNotWarn(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Done prerequisite", issue.StateDone, nil, "")
	seedDep(t, h, "iss001", "Ready dependent", issue.StateTodo, []string{"dep001"}, "")

	r := h.MustRun("start", "iss001", "--as", "alice")
	if strings.Contains(strings.ToLower(r.Stderr), "warning") || strings.Contains(r.Stderr, "not ready") {
		t.Errorf("a ready issue must start without a dependency warning:\n%s", r.Stderr)
	}
}

// A cancelled dependency leaves the dependent stuck, but start still begins it —
// advisory, not a gate — while warning that it is not ready.
func TestStartStuckDependencyWarnsButStarts(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Abandoned prerequisite", issue.StateCancelled, nil, "")
	seedDep(t, h, "iss001", "Stuck dependent", issue.StateTodo, []string{"dep001"}, "")

	r := h.Run("start", "iss001", "--as", "alice")
	if r.Code != 0 {
		t.Fatalf("start of a stuck issue exit = %d, want 0 (advisory, non-fatal)", r.Code)
	}
	out := h.DecodeJSON(r.Stdout)
	if out["state"] != "in-progress" {
		t.Errorf("state = %v, want in-progress (started despite the cancelled dependency)", out["state"])
	}
	// The machine-visible signal flags the stuck condition (a cancelled dependency).
	if rel, _ := out["relationships"].(map[string]any); rel["stuck"] != true || rel["blocked"] != true {
		t.Errorf("relationships = %v, want stuck=true blocked=true", out["relationships"])
	}
	if !strings.Contains(r.Stderr, "dep001") || !strings.Contains(strings.ToLower(r.Stderr), "not ready") {
		t.Errorf("start should warn about the cancelled dependency dep001:\n%s", r.Stderr)
	}
}

// The dependency signal start warns humans about (on stderr) is also machine-visible
// in the JSON result: the same additive "relationships" object show emits, so an
// agent sees — in the start output itself — that the work it just began was blocked
// and exactly what it is waiting on. This keeps start advisory (it started, exit 0)
// while making the readiness legible to a machine that never reads the stderr line.
func TestStartJSONCarriesReadinessSignal(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Prereq", issue.StateInProgress, nil, "")
	seedDep(t, h, "iss001", "Dependent", issue.StateTodo, []string{"dep001", "gone99"}, "")

	out := h.DecodeJSON(h.MustRun("start", "iss001", "--as", "alice").Stdout)
	// The base issue fields keep their shape alongside the additive object.
	if out["state"] != "in-progress" || out["assignee"] != "alice" {
		t.Fatalf("base fields = %v/%v, want in-progress/alice", out["state"], out["assignee"])
	}
	rel, ok := out["relationships"].(map[string]any)
	if !ok {
		t.Fatalf("start JSON has no relationships object: %v", out)
	}
	if rel["blocked"] != true || rel["ready"] != false {
		t.Errorf("relationships blocked/ready = %v/%v, want true/false", rel["blocked"], rel["ready"])
	}
	blockedOn, _ := rel["blocked_on"].([]any)
	if len(blockedOn) != 2 {
		t.Fatalf("blocked_on = %v, want the two unmet dependencies", blockedOn)
	}
	byID := map[string]map[string]any{}
	for _, b := range blockedOn {
		m := b.(map[string]any)
		byID[m["id"].(string)] = m
	}
	if byID["dep001"]["state"] != "in-progress" || byID["dep001"]["missing"] != false {
		t.Errorf("blocked_on[dep001] = %v, want state in-progress, missing false", byID["dep001"])
	}
	if byID["gone99"]["missing"] != true || byID["gone99"]["state"] != nil {
		t.Errorf("blocked_on[gone99] = %v, want missing true, state null", byID["gone99"])
	}
}

// The relationships object is a uniform, always-present shape: a ready start (every
// dependency done) reports blocked=false with an empty blocked_on, so a consumer
// never special-cases a missing key to learn "this one was fine".
func TestStartReadyJSONShowsUnblocked(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Done prereq", issue.StateDone, nil, "")
	seedDep(t, h, "iss001", "Ready dependent", issue.StateTodo, []string{"dep001"}, "")

	out := h.DecodeJSON(h.MustRun("start", "iss001", "--as", "alice").Stdout)
	rel, ok := out["relationships"].(map[string]any)
	if !ok {
		t.Fatalf("start JSON has no relationships object: %v", out)
	}
	if rel["blocked"] != false || rel["stuck"] != false {
		t.Errorf("relationships blocked/stuck = %v/%v, want false/false", rel["blocked"], rel["stuck"])
	}
	if bo, _ := rel["blocked_on"].([]any); len(bo) != 0 {
		t.Errorf("blocked_on = %v, want [] for a ready start", bo)
	}
}

// The human/machine split is deliberate: a human's start output stays the concise
// one-line confirmation (the blocked detail reaches them via the stderr warning),
// while the structured readiness lives in JSON. Starting a blocked issue at a TTY
// must not dump the relationship section onto stdout.
func TestStartHumanStaysConciseWithWarning(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seedDep(t, h, "dep001", "Prereq", issue.StateInProgress, nil, "")
	seedDep(t, h, "iss001", "Dependent", issue.StateTodo, []string{"dep001"}, "")

	r := h.MustRun("start", "iss001", "--as", "alice")
	if strings.Contains(strings.TrimSpace(r.Stdout), "\n") {
		t.Errorf("human start should be a single confirmation line, not a relationship dump:\n%s", r.Stdout)
	}
	if !strings.Contains(strings.ToLower(r.Stdout), "started") {
		t.Errorf("human start line should say started:\n%s", r.Stdout)
	}
	if strings.Contains(r.Stdout, "waiting on") {
		t.Errorf("blocked detail belongs on stderr, not stdout:\n%s", r.Stdout)
	}
	if !strings.Contains(r.Stderr, "dep001") {
		t.Errorf("the stderr warning should name the blocker:\n%s", r.Stderr)
	}
}

// Human output is a concise confirmation line (not JSON) for each ownership verb.
func TestOwnershipHumanOutput(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)

	claim := h.MustRun("claim", "iss001", "--as", "alice").Stdout
	if strings.HasPrefix(strings.TrimSpace(claim), "{") {
		t.Errorf("expected a human line, got JSON:\n%s", claim)
	}
	if !strings.Contains(claim, "iss001") || !strings.Contains(strings.ToLower(claim), "claim") {
		t.Errorf("claim line missing id or verb:\n%s", claim)
	}

	if assign := h.MustRun("assign", "iss001", "carol").Stdout; !strings.Contains(assign, "carol") {
		t.Errorf("assign line should name the actor:\n%s", assign)
	}
	if release := h.MustRun("release", "iss001").Stdout; !strings.Contains(strings.ToLower(release), "released") {
		t.Errorf("release line should say released:\n%s", release)
	}
	start := h.MustRun("start", "iss001", "--as", "alice").Stdout
	if !strings.Contains(strings.ToLower(start), "started") || !strings.Contains(start, "iss001") {
		t.Errorf("start line should say started and name the issue:\n%s", start)
	}
}

// Each ownership verb is a read-modify-write of the whole file, so it carries the
// human-owned body and any custom frontmatter keys through untouched (ADR 0014).
func TestOwnershipPreservesBodyAndCustomFields(t *testing.T) {
	h := beavertest.New(t).Init()
	h.WriteFile("issues/pre111-keep-me.md", `---
id: pre111
title: Keep me
state: todo
sprint: 7
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

Body with **markdown**.
`)
	h.MustRun("claim", "pre111", "--as", "alice")

	shown := h.DecodeJSON(h.MustRun("show", "pre111").Stdout)
	if shown["assignee"] != "alice" {
		t.Errorf("assignee = %v, want alice", shown["assignee"])
	}
	if custom, _ := shown["custom"].(map[string]any); custom["sprint"] != float64(7) {
		t.Errorf("custom field sprint not preserved: %v", shown["custom"])
	}
	if body, _ := shown["body"].(string); !strings.Contains(body, "**markdown**") {
		t.Errorf("body not preserved verbatim: %q", body)
	}
}

// Misuse of an ownership verb is a usage error (exit 2): a missing or extra ref, a
// missing or blank actor for assign, or an invalid format.
func TestOwnershipUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "x", issue.StateTodo, beavertest.DefaultNow)
	cases := [][]string{
		{"claim"},                              // missing ref
		{"claim", "a", "b"},                    // too many args
		{"claim", "iss001", "--format", "xml"}, // invalid format
		{"assign"},                             // missing ref and actor
		{"assign", "iss001"},                   // missing actor
		{"assign", "iss001", "  "},             // blank actor
		{"assign", "iss001", "a", "b"},         // too many args
		{"release"},                            // missing ref
		{"release", "a", "b"},                  // too many args
		{"start"},                              // missing ref
		{"start", "a", "b"},                    // too many args
		{"start", "iss001", "--format", "xml"}, // invalid format
	}
	for _, args := range cases {
		if r := h.Run(args...); r.Code != 2 {
			t.Errorf("%v exit = %d, want 2 (usage)", args, r.Code)
		}
	}
}

// An ownership verb targeting a missing issue exits 3 (not-found); without a store
// at all it exits 3 and points at init, like every other command.
func TestOwnershipNotFoundAndNoStore(t *testing.T) {
	for _, args := range [][]string{
		{"claim", "zzzzzz"},
		{"assign", "zzzzzz", "alice"},
		{"release", "zzzzzz"},
		{"start", "zzzzzz"},
	} {
		h := beavertest.New(t).Init()
		if r := h.Run(args...); r.Code != 3 {
			t.Errorf("%v on a missing issue exit = %d, want 3 (not-found)", args, r.Code)
		}

		noStore := beavertest.New(t) // no init
		r := noStore.Run(args...)
		if r.Code != 3 {
			t.Errorf("%v without a store exit = %d, want 3 (not-found)", args, r.Code)
		}
		if !strings.Contains(r.Stderr, "init") {
			t.Errorf("%v without a store should suggest init:\n%s", args, r.Stderr)
		}
	}
}

// --- helpers ---

// seedOwned writes an issue file directly with an assignee, so the guard tests can
// stage an issue owned by a specific actor without depending on claim/assign.
func seedOwned(t *testing.T, h *beavertest.Harness, id, title string, state issue.State, assignee string) {
	t.Helper()
	data, err := issue.Marshal(issue.Issue{
		ID: id, Title: title, State: state, Assignee: assignee,
		Created: beavertest.DefaultNow, Updated: beavertest.DefaultNow,
	})
	if err != nil {
		t.Fatalf("marshal seed %s: %v", id, err)
	}
	h.WriteFile("issues/"+issue.FileName(id, issue.Slug(title)), string(data))
}
