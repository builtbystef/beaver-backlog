package web

// This file holds the write surface: create, edit, note, and delete as HTML
// forms. Every handler here does the same three things — read the fields a
// browser posted, hand them to one core operation, and turn the answer into
// either a redirect or the form again with the core's refusal on it. No rule
// about what a change means lives here; a form is a way of phrasing a call.

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// createPage is the create form: the fields as they stand and, when the core
// refused them, what it said.
type createPage struct {
	page
	Form       createForm
	Priorities []option
	Refs       []refOption
	Error      string
}

// refOption is one issue the reference fields can offer as a completion: the
// id a datalist fills in, and the title that tells the reader which issue the
// id is.
type refOption struct {
	Value string
	Label string
}

// createForm is what the create form posts. Every field is text as the browser
// sent it — the core does the interpreting, so a refusal can hand back exactly
// what was typed rather than a cleaned-up version of it.
type createForm struct {
	Title       string
	Description string
	Priority    string
	Labels      string
	DependsOn   string
	Parent      string
}

// editPage is the edit form over one issue. The issue travels with it because
// the form's checkbox groups are the issue's own labels and dependencies.
type editPage struct {
	page
	Issue      issue.Issue
	Form       editForm
	Priorities []option
	Refs       []refOption
	Error      string
}

// editForm is the whole change set an edit can make. The labels and
// dependencies the issue already carries are checkboxes — unchecking one is how
// the form says "remove" — and the text fields beside them are how it says
// "add", which is exactly the add-and-remove shape the core's change set takes.
type editForm struct {
	Title        string
	Description  string
	Assignee     string
	Priority     string
	Labels       []choice
	AddLabels    string
	DependsOn    []choice
	AddDependsOn string
	Parent       string
}

// choice is one member of a checkbox group: a value the issue currently carries
// and whether the posted form kept it.
type choice struct {
	Value string
	Keep  bool
}

// option is one entry of the priority select, pre-marked so the template needs
// no comparison of its own.
type option struct {
	Value    string
	Label    string
	Selected bool
}

// noteForm is the detail page's note box: the text in it and the core's refusal
// when there was one.
type noteForm struct {
	Text  string
	Error string
}

// priorityLevels are the priority select's entries, the four levels plus the
// unprioritized value the core spells "none".
var priorityLevels = []string{"none", "urgent", "high", "medium", "low"}

// createForm renders an empty create form — the one page reached from every
// other, so a new issue is always one click away.
func (s *server) createFormPage(w http.ResponseWriter, r *http.Request) {
	s.renderCreate(w, r, createForm{}, "", http.StatusOK)
}

// create mints the issue the form describes and sends the reader to it. A
// refusal — an empty title, a bad priority, a reference naming no issue — is
// the form again with the core's words on it, still holding what was typed.
func (s *server) create(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, err)
		return
	}
	f := createForm{
		Title:       r.PostForm.Get("title"),
		Description: text(r.PostForm.Get("description")),
		Priority:    r.PostForm.Get("priority"),
		Labels:      r.PostForm.Get("labels"),
		DependsOn:   r.PostForm.Get("depends_on"),
		Parent:      r.PostForm.Get("parent"),
	}
	priority, err := core.ParsePriority(f.Priority)
	if err != nil {
		s.renderCreate(w, r, f, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	created, err := svc.Create(core.Draft{
		Title:     f.Title,
		Body:      f.Description,
		Labels:    values(f.Labels),
		Priority:  priority,
		DependsOn: values(f.DependsOn),
		Parent:    strings.TrimSpace(f.Parent),
	})
	if err != nil {
		if msg, refused := refusal(err); refused {
			s.renderCreate(w, r, f, msg, http.StatusUnprocessableEntity)
			return
		}
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/issues/"+created.Issue.ID, http.StatusSeeOther)
}

// editFormPage renders the edit form over the issue the URL names, with every
// label and dependency it holds checked — the form as "change nothing".
func (s *server) editFormPage(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	got, err := svc.Get(r.PathValue("ref"))
	if err != nil {
		s.failRef(w, r, r.PathValue("ref"), got.Warnings, err)
		return
	}
	iss := got.Issue
	s.renderEdit(w, r, iss, editForm{
		Title:       iss.Title,
		Description: issue.Description(iss.Body),
		Assignee:    iss.Assignee,
		Priority:    string(iss.Priority),
		Labels:      keeping(iss.Labels, iss.Labels),
		DependsOn:   keeping(iss.DependsOn, iss.DependsOn),
		Parent:      iss.Parent,
	}, "", http.StatusOK)
}

// update applies the posted change set. The issue is resolved first, so a
// reference naming nothing in the URL is a missing page while a reference
// naming nothing in a field is the form refusing what was typed.
func (s *server) update(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, err)
		return
	}
	ref := r.PathValue("ref")
	got, err := svc.Get(ref)
	if err != nil {
		s.failRef(w, r, ref, got.Warnings, err)
		return
	}
	iss := got.Issue

	f := editForm{
		Title:        r.PostForm.Get("title"),
		Description:  text(r.PostForm.Get("description")),
		Assignee:     r.PostForm.Get("assignee"),
		Priority:     r.PostForm.Get("priority"),
		Labels:       keeping(iss.Labels, r.PostForm["keep_labels"]),
		AddLabels:    r.PostForm.Get("add_labels"),
		DependsOn:    keeping(iss.DependsOn, r.PostForm["keep_depends_on"]),
		AddDependsOn: r.PostForm.Get("add_depends_on"),
		Parent:       r.PostForm.Get("parent"),
	}
	changes, err := f.changes(r)
	if err != nil {
		s.renderEdit(w, r, iss, f, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if _, err := svc.Update(iss.ID, changes); err != nil {
		if msg, refused := refusal(err); refused {
			s.renderEdit(w, r, iss, f, msg, http.StatusUnprocessableEntity)
			return
		}
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/issues/"+iss.ID, http.StatusSeeOther)
}

// changes turns the posted form into the core's change set. A field the form
// did not carry is left alone rather than cleared, so a partial form — the note
// box's page, a future inline editor — can never blank what it never showed.
func (f editForm) changes(r *http.Request) (core.Changes, error) {
	c := core.Changes{}
	if r.PostForm.Has("title") {
		c.Title = &f.Title
	}
	if r.PostForm.Has("description") {
		c.Body = &f.Description
	}
	if r.PostForm.Has("assignee") {
		assignee := strings.TrimSpace(f.Assignee)
		c.Assignee = &assignee
	}
	if r.PostForm.Has("priority") {
		priority, err := core.ParsePriority(f.Priority)
		if err != nil {
			return core.Changes{}, err
		}
		c.Priority = &priority
	}
	if governs(r, "labels") {
		c.AddLabels, c.RemoveLabels = values(f.AddLabels), dropped(f.Labels)
	}
	if governs(r, "depends_on") {
		c.AddDependsOn, c.RemoveDependsOn = values(f.AddDependsOn), dropped(f.DependsOn)
	}
	if r.PostForm.Has("parent") {
		parent := strings.TrimSpace(f.Parent)
		c.Parent = &parent
	}
	return c, nil
}

// governs reports whether the posted form speaks for a checkbox group. A group
// with everything unchecked posts none of its checkboxes, so the hidden marker
// beside it is what distinguishes "remove them all" from "this form never
// showed them".
func governs(r *http.Request, group string) bool {
	return r.PostForm.Has(group+"_form") || r.PostForm.Has("keep_"+group) || r.PostForm.Has("add_"+group)
}

// addNote appends the posted text to the issue's log, attributed to the actor
// the server was launched as. An empty note is the detail page again with the
// core's refusal by the box.
func (s *server) addNote(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, err)
		return
	}
	ref := r.PathValue("ref")
	got, err := svc.Get(ref)
	if err != nil {
		s.failRef(w, r, ref, got.Warnings, err)
		return
	}
	entry := text(r.PostForm.Get("text"))
	if _, err := svc.Note(got.Issue.ID, s.cfg.Actor, entry); err != nil {
		if msg, refused := refusal(err); refused {
			s.renderDetail(w, r, got, noteForm{Text: entry, Error: msg}, http.StatusUnprocessableEntity)
			return
		}
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/issues/"+got.Issue.ID, http.StatusSeeOther)
}

// remove deletes the issue's file outright and returns the reader to the board,
// which says what went — the file is gone and version control is the only undo,
// so the confirmation is the whole receipt.
func (s *server) remove(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ref := r.PathValue("ref")
	deleted, err := svc.Delete(ref)
	if err != nil {
		s.failRef(w, r, ref, deleted.Warnings, err)
		return
	}
	http.Redirect(w, r, "/?"+url.Values{"deleted": {deleted.Issue.ID}}.Encode(), http.StatusSeeOther)
}

func (s *server) renderCreate(w http.ResponseWriter, r *http.Request, f createForm, msg string, status int) {
	p := s.page("New issue", nil)
	p.Section = "new"
	s.render(w, r, "new.html", status, createPage{
		page:       p,
		Form:       f,
		Priorities: priorityOptions(f.Priority),
		Refs:       s.refOptions(""),
		Error:      msg,
	})
}

func (s *server) renderEdit(w http.ResponseWriter, r *http.Request, iss issue.Issue, f editForm, msg string, status int) {
	p := s.page("Edit "+iss.Title, nil)
	p.Section = "issues"
	s.render(w, r, "edit.html", status, editPage{
		page:       p,
		Issue:      iss,
		Form:       f,
		Priorities: priorityOptions(f.Priority),
		Refs:       s.refOptions(iss.ID),
		Error:      msg,
	})
}

// refOptions is every issue a reference field could name, offered as
// completions, minus the issue the form is about — an issue never depends on
// or parents itself. A store that cannot answer offers nothing: the fields
// stay plain text, never costing the form (ADR 0003).
func (s *server) refOptions(except string) []refOption {
	svc, err := s.open()
	if err != nil {
		return nil
	}
	listing, err := svc.List(core.Query{})
	if err != nil {
		return nil
	}
	var out []refOption
	for _, iss := range listing.Issues {
		if iss.ID == except {
			continue
		}
		out = append(out, refOption{Value: iss.ID, Label: iss.Title})
	}
	return out
}

// failRef words a failed reference for a route that takes one: several matches
// are a choice to offer, anything else the shared failure page.
func (s *server) failRef(w http.ResponseWriter, r *http.Request, ref string, warnings []core.Warning, err error) {
	var ambiguous *core.AmbiguousRefError
	if errors.As(err, &ambiguous) {
		p := s.page(ref, warnings)
		p.Section = "issues"
		s.render(w, r, "matches.html", http.StatusOK, matchesPage{
			page:    p,
			Ref:     ref,
			Matches: ambiguous.Matches,
		})
		return
	}
	s.fail(w, r, err)
}

// refusal reports what to show inline when the core refused the form's content,
// and false when the failure is not the form's to fix. Content is anything the
// reader typed — a field the rules reject, a reference naming no issue or
// several, a relationship that would close a cycle — so all of it comes back as
// the form rather than as an error page.
func refusal(err error) (string, bool) {
	var (
		invalid   *core.ValidationError
		cycle     *core.CycleError
		unknown   *core.UnknownRefError
		ambiguous *core.AmbiguousRefError
	)
	if errors.As(err, &invalid) || errors.As(err, &cycle) || errors.As(err, &unknown) || errors.As(err, &ambiguous) {
		return err.Error(), true
	}
	return "", false
}

// keeping pairs what the issue carries with what the form kept.
func keeping(current, kept []string) []choice {
	choices := make([]choice, len(current))
	for i, v := range current {
		choices[i] = choice{Value: v, Keep: slices.Contains(kept, v)}
	}
	return choices
}

// dropped is the values a checkbox group left unchecked — the removals.
func dropped(choices []choice) []string {
	var out []string
	for _, c := range choices {
		if !c.Keep {
			out = append(out, c.Value)
		}
	}
	return out
}

// values splits a text field holding several entries — labels, references — on
// commas or whitespace, which are the two ways anyone writes a short list.
func values(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
}

// text normalizes the line endings a browser sends from a textarea, so what
// lands in the file is the same Markdown a terminal would have written.
func text(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// priorityOptions marks the select's entries against the current value, taking
// the empty value as the unprioritized one the core spells "none".
func priorityOptions(current string) []option {
	sel := strings.TrimSpace(current)
	if sel == "" {
		sel = "none"
	}
	opts := make([]option, len(priorityLevels))
	for i, level := range priorityLevels {
		label := level
		if level == "none" {
			label = "–"
		}
		opts[i] = option{Value: level, Label: label, Selected: level == sel}
	}
	return opts
}
