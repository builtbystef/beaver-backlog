package web_test

// The board's write surface: dropping a card on a column. What is asserted here
// is the mapping from a drop to a core call and from the core's refusal to a
// status — the lifecycle itself (what a move writes, who may claim what) is the
// core's own tests. Drag mechanics are demoed by hand; the endpoints they post
// to are tested here.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestDropOnAColumnTransitionsThroughTheCore(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	h := newHandler(t, dir)

	for _, c := range []struct {
		state string
		want  issue.State
	}{
		{"done", issue.StateDone},
		{"cancelled", issue.StateCancelled},
	} {
		t.Run(c.state, func(t *testing.T) {
			target := create(t, svc, core.Draft{Title: "Fix flag parsing " + c.state})

			res := post(h, "/issues/"+target.ID+"/state", url.Values{"state": {c.state}})

			if res.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
			}
			if to := res.Header().Get("Location"); to != "/" {
				t.Errorf("redirected to %q, want the board", to)
			}
			if file := readIssueFile(t, dir, target.ID); !strings.Contains(file, "state: "+string(c.want)) {
				t.Errorf("file on disk is not %s:\n%s", c.want, file)
			}
		})
	}

	// Reopening is a drop on todo, the same as the CLI's reopen.
	closed := create(t, svc, core.Draft{Title: "Closed for now"})
	transition(t, svc, closed.ID, issue.StateDone)

	if res := post(h, "/issues/"+closed.ID+"/state", url.Values{"state": {"todo"}}); res.Code != http.StatusSeeOther {
		t.Fatalf("reopening drop = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	if file := readIssueFile(t, dir, closed.ID); !strings.Contains(file, "state: todo") {
		t.Errorf("file on disk was not reopened:\n%s", file)
	}
}

func TestDropOnInProgressClaimsForTheLaunchActor(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})

	res := post(newHandler(t, dir), "/issues/"+target.ID+"/start", nil)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	if to := res.Header().Get("Location"); to != "/" {
		t.Errorf("redirected to %q, want the board", to)
	}
	file := readIssueFile(t, dir, target.ID)
	for _, want := range []string{"state: " + string(issue.StateInProgress), "assignee: tester"} {
		if !strings.Contains(file, want) {
			t.Errorf("file on disk missing %q:\n%s", want, file)
		}
	}
}

// Every move between columns is legal — reclassifying a closed issue between
// done and cancelled included. The card goes where it was dropped.
func TestDropReclassifiesAClosedIssue(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	target := create(t, svc, core.Draft{Title: "All finished"})
	transition(t, svc, target.ID, issue.StateDone)

	res := post(newHandler(t, dir), "/issues/"+target.ID+"/state", url.Values{"state": {"cancelled"}})

	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303:\n%s", res.Code, res.Body.String())
	}
	if file := readIssueFile(t, dir, target.ID); !strings.Contains(file, "state: cancelled") {
		t.Errorf("file on disk is not cancelled:\n%s", file)
	}
}

// Starting an issue is refused the same way it is at the CLI, and the web never
// steals: the refusal names the holder and says where stealing lives.
func TestDropOnInProgressForAnIssueHeldByAnotherActorIsRefusedWith409(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	target := create(t, svc, core.Draft{Title: "Someone else's"})
	if _, err := svc.Start(target.ID, "stefan", false); err != nil {
		t.Fatalf("claim as another actor: %v", err)
	}
	before := readIssueFile(t, dir, target.ID)

	res := post(newHandler(t, dir), "/issues/"+target.ID+"/start", nil)

	if res.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409:\n%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, want := range []string{target.ID, "stefan", "--force"} {
		if !strings.Contains(body, want) {
			t.Errorf("refusal missing %q:\n%s", want, body)
		}
	}
	if after := readIssueFile(t, dir, target.ID); after != before {
		t.Errorf("the refused claim wrote to the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// A card dropped back where it came from is not a change, and an idempotent
// call must not look like one on disk.
func TestDroppingACardWhereItAlreadyIsWritesNothing(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	h := newHandler(t, dir)
	waiting := create(t, svc, core.Draft{Title: "Waiting to start"})
	mine := create(t, svc, core.Draft{Title: "Already mine"})
	start(t, svc, mine.ID) // as tester, the same actor the handler writes as

	for _, c := range []struct {
		name string
		id   string
		path string
		form url.Values
	}{
		{"same column", waiting.ID, "/issues/" + waiting.ID + "/state", url.Values{"state": {"todo"}}},
		{"re-claiming one's own", mine.ID, "/issues/" + mine.ID + "/start", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := readIssueFile(t, dir, c.id)

			res := post(h, c.path, c.form)

			if res.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303 — a no-op is not an error:\n%s", res.Code, res.Body.String())
			}
			if after := readIssueFile(t, dir, c.id); after != before {
				t.Errorf("a no-op drop rewrote the file:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestDropRoutesRefuseWhatTheyCannotAct(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	cases := []struct {
		name string
		path string
		form url.Values
		want int
	}{
		{"unknown reference", "/issues/no-such-issue/state", url.Values{"state": {"done"}}, http.StatusNotFound},
		{"unknown reference on start", "/issues/no-such-issue/start", nil, http.StatusNotFound},
		{"no column at all", "/issues/" + target.ID + "/state", nil, http.StatusUnprocessableEntity},
		{"not a column", "/issues/" + target.ID + "/state", url.Values{"state": {"sideways"}}, http.StatusUnprocessableEntity},
		{"in-progress is start's", "/issues/" + target.ID + "/state", url.Values{"state": {"in-progress"}}, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if res := post(h, c.path, c.form); res.Code != c.want {
				t.Errorf("POST %s = %d, want %d:\n%s", c.path, res.Code, c.want, res.Body.String())
			}
		})
	}
}

// The board ships the drag surface: cards are draggable handles, columns are
// drop targets addressed by what a drop means, and the script that wires them
// is served.
func TestBoardCardsAreDragHandlesAndColumnsAreDropTargets(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})
	h := newHandler(t, dir)

	body := get(h, "/").Body.String()

	for _, want := range []string{
		`draggable="true"`,
		`data-issue="` + target.ID + `"`,
		`data-state-url="/issues/` + target.ID + `/state"`,
		`data-start-url="/issues/` + target.ID + `/start"`,
		`src="/assets/drag.js"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("board missing %q:\n%s", want, body)
		}
	}
	if res := get(h, "/assets/drag.js"); res.Code != http.StatusOK {
		t.Errorf("GET /assets/drag.js = %d, want 200", res.Code)
	}
}
