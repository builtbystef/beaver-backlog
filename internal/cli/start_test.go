package cli_test

import (
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// What start does to an issue is the core's, and is covered at the core seam.
// These pin the command surface: the happy path, the two refusals it maps to
// exit codes of its own, and what it renders to a human and to an agent.

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

// With no --as, the shared identity chain honors BEAVER_BACKLOG_ACTOR (the
// agent/CI override).
func TestStartResolvesActorFromEnv(t *testing.T) {
	h := beavertest.New(t).Init()
	h.Env["BEAVER_BACKLOG_ACTOR"] = "ci-bot"
	seed(t, h, "iss001", "Work", issue.StateTodo, beavertest.DefaultNow)

	out := h.DecodeJSON(h.MustRun("start", "iss001").Stdout)
	if out["assignee"] != "ci-bot" {
		t.Errorf("assignee = %v, want ci-bot resolved from BEAVER_BACKLOG_ACTOR", out["assignee"])
	}
}

func TestStartRefusesAnothersIssueUnlessForce(t *testing.T) {
	h := beavertest.New(t).Init()
	seedOwned(t, h, "iss001", "Bob's work", issue.StateTodo, "bob")
	file := "issues/" + h.IssueFiles()[0]
	before := h.ReadFile(file)
	h.Clock.Advance(time.Hour) // a stray write would bump updated

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

func TestStartRejectsClosedIssue(t *testing.T) {
	for _, st := range []issue.State{issue.StateDone, issue.StateCancelled} {
		t.Run(string(st), func(t *testing.T) {
			h := beavertest.New(t).Init()
			seed(t, h, "iss001", "Closed", st, beavertest.DefaultNow)

			r := h.Run("start", "iss001", "--as", "alice")
			if r.Code != 2 {
				t.Errorf("start of a %s issue exit = %d, want 2 (usage)", st, r.Code)
			}
			if !strings.Contains(r.Stderr, "iss001") || !strings.Contains(strings.ToLower(r.Stderr), "reopen") {
				t.Errorf("rejection should name the issue and suggest reopen:\n%s", r.Stderr)
			}
		})
	}
}

// The dependency check is advisory: start warns, naming the unmet dependencies,
// but never blocks.
func TestStartWarnsOnUnmetDependenciesButStarts(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Prerequisite", issue.StateInProgress, nil, "")
	seedDep(t, h, "iss001", "The dependent", issue.StateTodo, []string{"dep001", "gone99"}, "")

	r := h.Run("start", "iss001", "--as", "alice")
	if r.Code != 0 {
		t.Fatalf("start of a not-ready issue exit = %d, want 0 — the dependency warning is advisory, never fatal", r.Code)
	}
	out := h.DecodeJSON(r.Stdout)
	if out["state"] != "in-progress" || out["assignee"] != "alice" {
		t.Errorf("not-ready issue = %v/%v, want in-progress/alice (started despite the warning)", out["state"], out["assignee"])
	}
	for _, want := range []string{"warning", "not ready", "dep001", "gone99"} {
		if !strings.Contains(r.Stderr, want) {
			t.Errorf("dependency warning missing %q:\n%s", want, r.Stderr)
		}
	}
}

// The readiness signal also lands in start's JSON, so an agent that never reads
// stderr still sees the work it began was blocked and what it is waiting on.
func TestStartJSONCarriesReadinessSignal(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Prereq", issue.StateInProgress, nil, "")
	seedDep(t, h, "iss001", "Dependent", issue.StateTodo, []string{"dep001", "gone99"}, "")

	out := h.DecodeJSON(h.MustRun("start", "iss001", "--as", "alice").Stdout)
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

// The relationships object is always present — a ready start reports
// blocked=false with an empty blocked_on — so a consumer never special-cases a
// missing key, and there is nothing to warn about.
func TestStartReadyJSONShowsUnblockedAndDoesNotWarn(t *testing.T) {
	h := beavertest.New(t).Init()
	seedDep(t, h, "dep001", "Done prereq", issue.StateDone, nil, "")
	seedDep(t, h, "iss001", "Ready dependent", issue.StateTodo, []string{"dep001"}, "")

	r := h.MustRun("start", "iss001", "--as", "alice")
	if strings.Contains(strings.ToLower(r.Stderr), "warning") || strings.Contains(r.Stderr, "not ready") {
		t.Errorf("a ready issue must start without a dependency warning:\n%s", r.Stderr)
	}
	rel, ok := h.DecodeJSON(r.Stdout)["relationships"].(map[string]any)
	if !ok {
		t.Fatalf("start JSON has no relationships object:\n%s", r.Stdout)
	}
	if rel["blocked"] != false || rel["stuck"] != false {
		t.Errorf("relationships blocked/stuck = %v/%v, want false/false", rel["blocked"], rel["stuck"])
	}
	if bo, _ := rel["blocked_on"].([]any); len(bo) != 0 {
		t.Errorf("blocked_on = %v, want [] for a ready start", bo)
	}
}

// Blocked detail reaches humans via the stderr warning; stdout stays a single
// confirmation line, never a relationship dump.
func TestStartHumanStaysConciseWithWarning(t *testing.T) {
	h := beavertest.New(t).Init()
	h.IsTTY = true
	seedDep(t, h, "dep001", "Prereq", issue.StateInProgress, nil, "")
	seedDep(t, h, "iss001", "Dependent", issue.StateTodo, []string{"dep001"}, "")

	r := h.MustRun("start", "iss001", "--as", "alice")
	if strings.Contains(strings.TrimSpace(r.Stdout), "\n") {
		t.Errorf("human start should be a single confirmation line, not a relationship dump:\n%s", r.Stdout)
	}
	if !strings.Contains(strings.ToLower(r.Stdout), "started") || !strings.Contains(r.Stdout, "iss001") {
		t.Errorf("human start line should say started and name the issue:\n%s", r.Stdout)
	}
	if strings.Contains(r.Stdout, "waiting on") {
		t.Errorf("blocked detail belongs on stderr, not stdout:\n%s", r.Stdout)
	}
	if !strings.Contains(r.Stderr, "dep001") {
		t.Errorf("the stderr warning should name the blocker:\n%s", r.Stderr)
	}
}

func TestStartUsageErrors(t *testing.T) {
	h := beavertest.New(t).Init()
	seed(t, h, "iss001", "x", issue.StateTodo, beavertest.DefaultNow)
	cases := [][]string{
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

func TestStartNotFoundAndNoStore(t *testing.T) {
	h := beavertest.New(t).Init()
	if r := h.Run("start", "zzzzzz"); r.Code != 3 {
		t.Errorf("start of a missing issue exit = %d, want 3 (not-found)", r.Code)
	}

	noStore := beavertest.New(t) // no init
	r := noStore.Run("start", "zzzzzz")
	if r.Code != 3 {
		t.Errorf("start without a store exit = %d, want 3 (not-found)", r.Code)
	}
	if !strings.Contains(r.Stderr, "init") {
		t.Errorf("start without a store should suggest init:\n%s", r.Stderr)
	}
}

// --- helpers ---

// seedOwned writes an issue file with an assignee directly, so the guard test
// does not lean on the command that would set one.
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
