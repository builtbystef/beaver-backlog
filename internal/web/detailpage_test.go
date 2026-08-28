package web_test

// One issue's page as it renders: the lifecycle actions beside its title, the
// field list, the prose, the derived relationships, and the note box. What is
// asserted here is structure a reader can observe: a button's word and where
// it posts, a field's answer, a heading the Markdown produced. Never the class
// names that draw any of it.

import (
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The actions beside the title are exactly the moves the lifecycle allows from
// where the issue stands, and pressing one still makes its move.
func TestDetailOffersExactlyTheMovesTheLifecycleAllows(t *testing.T) {
	cases := []struct {
		from  issue.State
		moves map[string]issue.State // the word on the button, and where it lands
	}{
		{issue.StateTodo, map[string]issue.State{
			"Start":  issue.StateInProgress,
			"Done":   issue.StateDone,
			"Cancel": issue.StateCancelled,
		}},
		{issue.StateInProgress, map[string]issue.State{
			"Done":   issue.StateDone,
			"Cancel": issue.StateCancelled,
		}},
		{issue.StateDone, map[string]issue.State{"Reopen": issue.StateTodo}},
		{issue.StateCancelled, map[string]issue.State{"Reopen": issue.StateTodo}},
	}
	for _, c := range cases {
		t.Run(string(c.from), func(t *testing.T) {
			dir := newStore(t)
			svc := open(t, dir)
			h := newHandler(t, dir)

			offered := offeredMoves(t, get(h, "/issues/"+inState(t, svc, "Fix flag parsing", c.from).ID).Body.String())
			if got, want := buttonWords(offered), buttonWords(c.moves); got != want {
				t.Fatalf("a %s issue offers %s, want %s", c.from, got, want)
			}

			// Each move gets an issue of its own, since taking one spends the
			// state it was offered from.
			for word, lands := range c.moves {
				target := inState(t, svc, "Fix flag parsing", c.from)
				m := offeredMoves(t, get(h, "/issues/"+target.ID).Body.String())[word]
				if res := post(h, m.action, m.posts); res.Code != http.StatusSeeOther {
					t.Fatalf("%s = %d, want 303:\n%s", word, res.Code, res.Body.String())
				}
				if got := fetch(t, svc, target.ID).State; got != lands {
					t.Errorf("%s from %s landed in %s, want %s", word, c.from, got, lands)
				}
			}
		})
	}
}

// The field list is the whole file made readable: every field the issue holds
// has a row of its own, and a field it leaves empty reads as answered rather
// than blank.
func TestDetailAnswersEveryFieldEvenTheEmptyOnes(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	bare := create(t, svc, core.Draft{Title: "Nothing filled in"})
	addCustomField(t, dir, bare.ID, "epic", "Q3 cleanup")

	fields := answeredFields(t, get(newHandler(t, dir), "/issues/"+bare.ID).Body.String())

	for _, name := range []string{"ID", "State", "Priority", "Labels", "Assignee", "Created", "Updated", "epic"} {
		if fields[name] == "" {
			t.Errorf("the page answers nothing for %q; it holds %v", name, fields)
		}
	}
	if fields["epic"] != "Q3 cleanup" {
		t.Errorf("the frontmatter key epic reads %q, want its value", fields["epic"])
	}
}

// The description is prose, not source: what the reader wrote in Markdown comes
// out as the elements it asked for. An issue with nothing written down says so
// rather than showing a hole.
func TestDetailRendersTheDescriptionAsMarkdown(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	written := create(t, svc, core.Draft{
		Title: "Fix flag parsing",
		Body:  "## Cause\n\nThe loop bound is off.\n\n- drops the last flag\n- only with a trailing space\n",
	})
	bare := create(t, svc, core.Draft{Title: "Nothing written down"})
	h := newHandler(t, dir)

	body := get(h, "/issues/"+written.ID).Body.String()
	for _, want := range []string{"<h2>Cause</h2>", "<li>drops the last flag</li>", "<p>The loop bound is off.</p>"} {
		if !strings.Contains(body, want) {
			t.Errorf("the description is not rendered as Markdown, missing %q:\n%s", want, body)
		}
	}

	if said := words(get(h, "/issues/"+bare.ID).Body.String()); !strings.Contains(said, "No description") {
		t.Errorf("an issue with no description says nothing about it:\n%s", said)
	}
}

// The relationships block says what the core derived about the issue's
// dependencies, and names every unmet one with the state it is sitting in.
func TestDetailSaysWhetherTheIssueIsReadyBlockedOrStuck(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	settled := create(t, svc, core.Draft{Title: "Groundwork"})
	transition(t, svc, settled.ID, issue.StateDone)
	going := create(t, svc, core.Draft{Title: "Still going"})
	abandoned := create(t, svc, core.Draft{Title: "Given up on"})
	transition(t, svc, abandoned.ID, issue.StateCancelled)

	cases := []struct {
		condition string
		on        issue.Issue
		waiting   []string // what the block must say it is waiting on
	}{
		{"ready", settled, nil},
		{"blocked", going, []string{going.ID, string(issue.StateTodo)}},
		{"stuck", abandoned, []string{abandoned.ID, string(issue.StateCancelled)}},
	}
	h := newHandler(t, dir)
	for _, c := range cases {
		t.Run(c.condition, func(t *testing.T) {
			target := create(t, svc, core.Draft{Title: "Waits on " + c.on.Title, DependsOn: []string{c.on.ID}})

			fields := answeredFields(t, get(h, "/issues/"+target.ID).Body.String())

			if !strings.Contains(fields["Status"], c.condition) {
				t.Errorf("status reads %q, want it to say %q", fields["Status"], c.condition)
			}
			for _, want := range c.waiting {
				if !strings.Contains(fields["Waiting on"], want) {
					t.Errorf("waiting on reads %q, missing %q", fields["Waiting on"], want)
				}
			}
		})
	}
}

// A refused note is this same page again with the words still in the box, so
// nobody retypes what the core would not take.
func TestARefusedNoteComesBackWithTheWordsStillInTheBox(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})

	// Whitespace is the one thing the core refuses a note for, so it is the
	// only way to ask the page what it does with a refusal.
	typed := "  \n  "
	res := post(newHandler(t, dir), "/issues/"+target.ID+"/notes", url.Values{"text": {typed}})

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422:\n%s", res.Code, res.Body.String())
	}
	if got := boxText(t, res.Body.String(), "text"); got != typed {
		t.Errorf("the note box came back holding %q, want %q", got, typed)
	}
}

var (
	moveForm    = regexp.MustCompile(`(?s)<form([^>]*method="post"[^>]*)>(.*?)</form>`)
	hiddenField = regexp.MustCompile(`<input[^>]*type="hidden"[^>]*>`)
	fieldPair   = regexp.MustCompile(`(?s)<dt[^>]*>(.*?)</dt>\s*<dd[^>]*>(.*?)</dd>`)
	moveAction  = regexp.MustCompile(`/(start|state)$`)
)

// pageMove is one lifecycle action the page offers: where its form posts and
// what it posts there.
type pageMove struct {
	action string
	posts  url.Values
}

// offeredMoves reads the lifecycle actions off a rendered issue page, keyed by
// the word on the button. Every other form on the page (the note box, the
// delete) posts somewhere else and is not one.
func offeredMoves(t *testing.T, body string) map[string]pageMove {
	t.Helper()
	moves := map[string]pageMove{}
	for _, m := range moveForm.FindAllStringSubmatch(body, -1) {
		action := attr(m[1], "action")
		if !moveAction.MatchString(action) {
			continue
		}
		posts := url.Values{}
		for _, f := range hiddenField.FindAllString(m[2], -1) {
			posts.Set(attr(f, "name"), attr(f, "value"))
		}
		word := words(m[2])
		if word == "" {
			t.Errorf("a move posting to %s says nothing on its button:\n%s", action, m[0])
			continue
		}
		moves[word] = pageMove{action: action, posts: posts}
	}
	return moves
}

// answeredFields reads every named field off a rendered page, the issue's own
// and the relationships alike, as the words a reader sees against each name.
func answeredFields(t *testing.T, body string) map[string]string {
	t.Helper()
	fields := map[string]string{}
	for _, m := range fieldPair.FindAllStringSubmatch(body, -1) {
		fields[words(m[1])] = words(m[2])
	}
	if len(fields) == 0 {
		t.Fatalf("the page names no fields at all:\n%s", body)
	}
	return fields
}

// inState creates an issue and puts it where the case needs it, taking the
// route the lifecycle offers: in-progress is start's, the terminal states are
// transitions.
func inState(t *testing.T, svc *core.Service, title string, state issue.State) issue.Issue {
	t.Helper()
	iss := create(t, svc, core.Draft{Title: title})
	switch state {
	case issue.StateTodo:
	case issue.StateInProgress:
		start(t, svc, iss.ID)
	default:
		transition(t, svc, iss.ID, state)
	}
	return iss
}

// buttonWords names the two sides of a move comparison the same way, so a
// mismatch reads as the words that differ.
func buttonWords[V any](moves map[string]V) string {
	return strings.Join(slices.Sorted(maps.Keys(moves)), ", ")
}
