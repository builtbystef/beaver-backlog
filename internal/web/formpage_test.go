package web_test

// The write forms as they render: every field reaching its own control, the
// completions the reference boxes offer, what a refused submission still holds,
// and the guard in front of a delete. What is asserted here is structure a
// reader can observe: a label's target, an option's value, a dialog a form
// waits on. Never the class names that draw any of it.

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// Every control the form posts is labelled, and every label reaches a control
// of its own: a field nobody can name is a field nobody can fill in.
func TestEveryFormFieldReachesItsOwnControl(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	blocker := create(t, svc, core.Draft{Title: "Groundwork"})
	target := create(t, svc, core.Draft{
		Title:     "Fix flag parsing",
		Labels:    []string{"bug"},
		DependsOn: []string{blocker.ID},
	})
	h := newHandler(t, dir)

	cases := []struct {
		path   string
		fields []string
	}{
		{"/issues/new", []string{"title", "description", "priority", "labels", "depends_on", "parent"}},
		{"/issues/" + target.ID + "/edit", []string{
			"title", "description", "assignee", "priority",
			"keep_labels", "add_labels", "keep_depends_on", "add_depends_on", "parent",
		}},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			form := issueForm(t, get(h, c.path).Body.String())
			for _, name := range c.fields {
				if namedControl(t, form, name) == "" {
					t.Errorf("the form posts no %q field:\n%s", name, form)
				}
			}

			// A label pointing at nothing, or at two controls at once, reaches
			// no control of its own.
			ids := map[string]int{}
			for _, m := range controlTag.FindAllStringSubmatch(form, -1) {
				if id := attr(m[2], "id"); id != "" {
					ids[id]++
				}
			}
			named := map[string]bool{}
			var wrapping string
			for _, m := range labelTag.FindAllStringSubmatch(form, -1) {
				wrapping += m[2]
				target := attr(m[1], "for")
				if target == "" {
					continue
				}
				named[target] = true
				if ids[target] != 1 {
					t.Errorf("label for %q reaches %d controls, want 1:\n%s", target, ids[target], form)
				}
				if words(m[2]) == "" {
					t.Errorf("the label for %q says nothing", target)
				}
			}
			for _, m := range controlTag.FindAllStringSubmatch(form, -1) {
				name, kind := attr(m[2], "name"), attr(m[2], "type")
				if name == "" || kind == "hidden" {
					continue
				}
				// A control is named either by a label pointing at it or by one
				// wrapped around it; the checkbox groups use the second.
				if named[attr(m[2], "id")] || strings.Contains(wrapping, m[0]) {
					continue
				}
				t.Errorf("the %q field carries no label:\n%s", name, m[0])
			}
		})
	}
}

// Nobody types an id from memory: every reference box offers the store's issues
// as id-plus-title, minus the issue the form is about.
func TestReferenceFieldsOfferEveryIssueAsACompletion(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	first := create(t, svc, core.Draft{Title: "Groundwork"})
	second := create(t, svc, core.Draft{Title: "The spec"})
	h := newHandler(t, dir)

	body := get(h, "/issues/new").Body.String()
	offered := completions(t, body)
	for _, iss := range []issue.Issue{first, second} {
		if offered[iss.ID] != iss.Title {
			t.Errorf("the create form offers %s as %q, want %q", iss.ID, offered[iss.ID], iss.Title)
		}
	}
	assertReadsTheCompletions(t, body, "/issues/new", "depends_on", "parent")

	edit := get(h, "/issues/"+first.ID+"/edit").Body.String()
	offered = completions(t, edit)
	if _, self := offered[first.ID]; self {
		t.Error("the edit form offers the issue as its own reference")
	}
	if offered[second.ID] != second.Title {
		t.Errorf("the edit form offers %s as %q, want %q", second.ID, offered[second.ID], second.Title)
	}
	assertReadsTheCompletions(t, edit, "the edit form", "add_depends_on", "parent")
}

// A refusal is the form again, and the form still holds what was typed into it:
// a reader fixes one field rather than writing the issue twice.
func TestARefusedSubmissionComesBackHoldingEverythingTyped(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	blocker := create(t, svc, core.Draft{Title: "Groundwork"})
	target := create(t, svc, core.Draft{
		Title:     "Fix flag parsing",
		Labels:    []string{"bug", "stale"},
		DependsOn: []string{blocker.ID},
	})
	h := newHandler(t, dir)

	res := post(h, "/issues", url.Values{
		"title":       {"Fix flag parsing properly"},
		"description": {"The parser drops the last flag."},
		"priority":    {"high"},
		"labels":      {"bug, cli"},
		"depends_on":  {"no-such-issue"},
		"parent":      {blocker.ID},
	})
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422:\n%s", res.Code, res.Body.String())
	}
	form := issueForm(t, res.Body.String())
	for field, want := range map[string]string{
		"title":      "Fix flag parsing properly",
		"labels":     "bug, cli",
		"depends_on": "no-such-issue",
		"parent":     blocker.ID,
	} {
		if got := attr(namedControl(t, form, field), "value"); got != want {
			t.Errorf("the refused form holds %q = %q, want %q", field, got, want)
		}
	}
	if got := boxText(t, form, "description"); got != "The parser drops the last flag." {
		t.Errorf("the refused form holds description %q, want what was typed", got)
	}
	if got := selected(t, form); got != "high" {
		t.Errorf("the refused form has %q selected, want high", got)
	}

	// The edit form's checkbox groups come back as they were answered: the box
	// that was unticked is still unticked, so a refusal never quietly restores
	// what the reader dropped.
	res = post(h, "/issues/"+target.ID, url.Values{
		"title":           {"Fix flag parsing"},
		"priority":        {"none"},
		"keep_labels":     {"bug"}, // "stale" was dropped
		"add_labels":      {"cli"},
		"keep_depends_on": {blocker.ID},
		"add_depends_on":  {"no-such-issue"},
	})
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422:\n%s", res.Code, res.Body.String())
	}
	form = issueForm(t, res.Body.String())
	for value, want := range map[string]bool{"bug": true, "stale": false} {
		if got := ticked(t, form, "keep_labels", value); got != want {
			t.Errorf("the refused form has %q ticked = %v, want %v", value, got, want)
		}
	}
	if got := ticked(t, form, "keep_depends_on", blocker.ID); !got {
		t.Errorf("the refused form dropped the dependency the reader kept")
	}
	for field, want := range map[string]string{"add_labels": "cli", "add_depends_on": "no-such-issue"} {
		if got := attr(namedControl(t, form, field), "value"); got != want {
			t.Errorf("the refused form holds %q = %q, want %q", field, got, want)
		}
	}
}

// Deleting is the one act with no undo, so it waits on an answer: the form
// names a dialog, and the dialog offers both ways out.
func TestDeletingAsksFirst(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Junk"})
	h := newHandler(t, dir)
	body := get(h, "/issues/"+target.ID).Body.String()

	m := deleteForm.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("the issue offers no delete:\n%s", body)
	}
	guard := attr(m[1], "data-confirm")
	if guard == "" {
		t.Fatalf("delete posts without asking first:\n%s", m[0])
	}
	dialog := regexp.MustCompile(`(?s)<dialog id="` + regexp.QuoteMeta(guard) + `"[^>]*>(.*?)</dialog>`).FindStringSubmatch(body)
	if dialog == nil {
		t.Fatalf("the form waits on a dialog %q that is not on the page:\n%s", guard, body)
	}
	if !strings.Contains(dialog[1], `method="dialog"`) {
		t.Errorf("the confirmation cannot close itself:\n%s", dialog[0])
	}
	// Cancelling is a way out of its own: the script submits on "confirm" and
	// on nothing else, so the other button has to be there to be pressed.
	for _, want := range []string{`value="cancel"`, `value="confirm"`} {
		if !strings.Contains(dialog[1], want) {
			t.Errorf("the confirmation offers no %s:\n%s", want, dialog[0])
		}
	}
	if !strings.Contains(body, "/assets/confirm.js") {
		t.Errorf("the page does not load the script the guard needs:\n%s", body)
	}
	if code := get(h, "/assets/confirm.js").Code; code != http.StatusOK {
		t.Errorf("GET /assets/confirm.js = %d, want 200", code)
	}
}

var (
	postForm   = regexp.MustCompile(`(?s)<form[^>]*method="post"[^>]*>(.*?)</form>`)
	deleteForm = regexp.MustCompile(`(?s)<form([^>]*action="[^"]*/delete"[^>]*)>.*?</form>`)
	labelTag   = regexp.MustCompile(`(?s)<label([^>]*)>(.*?)</label>`)
	controlTag = regexp.MustCompile(`<(input|select|textarea)\b([^>]*)>`)
	refList    = regexp.MustCompile(`(?s)<datalist id="issue-refs">(.*?)</datalist>`)
	optionTag  = regexp.MustCompile(`<option value="([^"]*)"[^>]*>([^<]*)</option>`)
	tagOrSpace = regexp.MustCompile(`(?s)<[^>]*>|\s+`)
)

// issueForm is the write form on a page: the one thing on it that posts.
func issueForm(t *testing.T, body string) string {
	t.Helper()
	m := postForm.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no form posts on this page:\n%s", body)
	}
	return m[1]
}

// namedControl is the control posting under name, empty when the form has none.
func namedControl(t *testing.T, form, name string) string {
	t.Helper()
	for _, m := range controlTag.FindAllStringSubmatch(form, -1) {
		if attr(m[2], "name") == name {
			return m[2]
		}
	}
	return ""
}

// attr reads one attribute off a tag's attribute text.
func attr(tag, name string) string {
	m := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `="([^"]*)"`).FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	return m[1]
}

// boxText is what a named textarea holds.
func boxText(t *testing.T, form, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?s)<textarea[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]*>(.*?)</textarea>`).FindStringSubmatch(form)
	if m == nil {
		t.Fatalf("the form holds no %q box:\n%s", name, form)
	}
	return m[1]
}

// selected is the value the form's dropdown is showing.
func selected(t *testing.T, form string) string {
	t.Helper()
	m := regexp.MustCompile(`<option value="([^"]*)" selected>`).FindStringSubmatch(form)
	if m == nil {
		t.Fatalf("nothing is selected in the form's dropdown:\n%s", form)
	}
	return m[1]
}

// ticked reports whether the checkbox posting value under name is checked.
func ticked(t *testing.T, form, name, value string) bool {
	t.Helper()
	pattern := `<input[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]*value="` + regexp.QuoteMeta(value) + `"([^>]*)>`
	m := regexp.MustCompile(pattern).FindStringSubmatch(form)
	if m == nil {
		t.Fatalf("the form holds no %q box for %q:\n%s", name, value, form)
	}
	return strings.Contains(m[1], "checked")
}

// completions is the reference list the form offers, keyed by the id each entry
// fills in.
func completions(t *testing.T, body string) map[string]string {
	t.Helper()
	m := refList.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("the form offers no completions:\n%s", body)
	}
	out := map[string]string{}
	for _, o := range optionTag.FindAllStringSubmatch(m[1], -1) {
		out[o[1]] = o[2]
	}
	return out
}

// assertReadsTheCompletions checks that each named reference box is wired to
// the shared list rather than left as plain text.
func assertReadsTheCompletions(t *testing.T, body, where string, fields ...string) {
	t.Helper()
	form := issueForm(t, body)
	for _, name := range fields {
		if got := attr(namedControl(t, form, name), "list"); got != "issue-refs" {
			t.Errorf("%s: the %q box reads completions from %q, want the shared list", where, name, got)
		}
	}
}

// words is a fragment's visible text, tags and spacing removed.
func words(fragment string) string {
	return strings.TrimSpace(tagOrSpace.ReplaceAllString(fragment, " "))
}
