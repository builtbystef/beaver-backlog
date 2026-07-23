// Package output renders issues for two audiences: a human reading a terminal,
// and a machine (or agent) consuming JSON. The format is auto-detected and
// overridable.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// Format is a rendering style.
type Format string

// The two rendering styles: aligned text for a terminal, JSON for a machine.
const (
	Human Format = "human"
	JSON  Format = "json"
)

// Resolve picks the output format. An explicit override ("human"/"json")
// always wins; otherwise output is human for an interactive terminal and JSON
// for a non-interactive pipe or a detected agent. Agent detection is the
// caller's, passed in rather than sniffed from the environment here.
func Resolve(override string, stdoutIsTTY bool, agentDetected bool) (Format, error) {
	switch override {
	case string(Human):
		return Human, nil
	case string(JSON):
		return JSON, nil
	case "":
		if !stdoutIsTTY || agentDetected {
			return JSON, nil
		}
		return Human, nil
	default:
		return "", fmt.Errorf("invalid format %q (want human or json)", override)
	}
}

// WriteIssue renders a single issue in the given format.
func WriteIssue(w io.Writer, iss issue.Issue, f Format) error {
	if f == JSON {
		return WriteJSON(w, toJSONView(iss))
	}
	return writeHuman(w, iss)
}

// WriteIssueWithRelationship renders one issue together with its derived
// relationship view: what it is waiting on, whether it is ready/blocked/stuck,
// and the inverse edges (what it blocks, its children). In JSON the derived
// view is an additive "relationships" object beside the same issue fields, so
// a consumer reading the base fields sees an unchanged shape.
func WriteIssueWithRelationship(w io.Writer, iss issue.Issue, rel issue.Relationship, f Format) error {
	if f == JSON {
		return WriteJSON(w, issueWithRel{jsonView: toJSONView(iss), Relationships: toRelView(rel)})
	}
	var b strings.Builder
	writeHumanHead(&b, iss)
	writeHumanRelationship(&b, rel)
	writeHumanBody(&b, iss)
	_, err := io.WriteString(w, b.String())
	return err
}

// WriteIssueWithCommit renders one issue together with the completion commit
// the command may have recorded. In JSON the "commit" object is always present
// beside the same issue fields — null when no commit was made — so consumers
// see one constant shape. Human output stays the plain issue.
func WriteIssueWithCommit(w io.Writer, iss issue.Issue, revision string, f Format) error {
	if f == JSON {
		return WriteJSON(w, issueWithCommit{jsonView: toJSONView(iss), Commit: toCommitView(revision)})
	}
	return writeHuman(w, iss)
}

// WriteList renders a collection of issues in the given format, preserving the
// caller's order. JSON is an array of the same per-issue objects WriteIssue
// emits; human output is an aligned table.
func WriteList(w io.Writer, issues []issue.Issue, f Format) error {
	if f == JSON {
		views := make([]jsonView, len(issues)) // non-nil: an empty list is [] not null
		for i, iss := range issues {
			views[i] = toJSONView(iss)
		}
		return WriteJSON(w, views)
	}
	return writeTable(w, issues)
}

// writeTable renders issues as an aligned human table. The column set is not a
// contract — the machine shape lives in JSON. The free-form title goes last so
// its variable width cannot misalign the rest.
func writeTable(w io.Writer, issues []issue.Issue) error {
	if len(issues) == 0 {
		_, err := io.WriteString(w, "No issues.\n")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPRIORITY\tSTATE\tASSIGNEE\tLABELS\tTITLE")
	for _, iss := range issues {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			iss.ID,
			orDash(string(iss.Priority)),
			iss.State,
			orDash(iss.Assignee),
			orDash(OneLine(strings.Join(iss.Labels, ", "))),
			OneLine(iss.Title))
	}
	return tw.Flush()
}

// orDash renders an absent optional cell as "-", keeping empty table columns legible.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// OneLine flattens a value to a single line, so a newline or tab spliced into
// a title by a hand-edit cannot break a one-line-per-item layout (tabs would
// also confuse tabwriter's column grid).
func OneLine(s string) string {
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ").Replace(s)
}

// WriteJSON encodes v as indented JSON with a trailing newline, without escaping
// HTML metacharacters (so titles and bodies read naturally).
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonView is the stable machine shape of an issue: every field is always
// present, with unset single-valued options as null and unset lists as [], so
// consumers need not special-case missing keys.
type jsonView struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Assignee  *string  `json:"assignee"`
	Priority  *string  `json:"priority"`
	Labels    []string `json:"labels"`
	DependsOn []string `json:"depends_on"`
	Parent    *string  `json:"parent"`
	Created   *string  `json:"created"` // null when the file carries no timestamp
	Updated   *string  `json:"updated"`
	Body      string   `json:"body"`
	// Notes is the body's Notes section parsed into structured entries; the raw
	// entries also remain in Body verbatim. Always an array, empty when none.
	Notes []noteView `json:"notes"`
	// Custom carries user-defined frontmatter keys, preserved but uninterpreted.
	// Always an object, empty when none.
	Custom map[string]any `json:"custom"`
}

// noteView is one note in JSON: who wrote it, when (RFC3339 UTC), and its text.
type noteView struct {
	Author string `json:"author"`
	Time   string `json:"time"`
	Text   string `json:"text"`
}

func toJSONView(iss issue.Issue) jsonView {
	return jsonView{
		ID:        iss.ID,
		Title:     iss.Title,
		State:     string(iss.State),
		Assignee:  optString(iss.Assignee),
		Priority:  optString(string(iss.Priority)),
		Labels:    orEmpty(iss.Labels),
		DependsOn: orEmpty(iss.DependsOn),
		Parent:    optString(iss.Parent),
		Created:   optString(formatTime(iss.Created)),
		Updated:   optString(formatTime(iss.Updated)),
		Body:      iss.Body,
		Notes:     toNoteViews(issue.ParseNotes(iss.Body)),
		Custom:    sanitizeCustom(iss.Custom),
	}
}

// sanitizeCustom projects a preserved custom map into a JSON-encodable copy.
// YAML admits non-finite floats (.nan, ±.inf) that encoding/json refuses, and
// one such value would otherwise fail the whole JSON write; they render as
// "NaN"/"Infinity"/"-Infinity", recursing into sequences and maps. The result
// is always non-nil.
func sanitizeCustom(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = jsonSafe(v)
	}
	return out
}

func jsonSafe(v any) any {
	switch t := v.(type) {
	case float64:
		return finiteOrName(t)
	case float32:
		return finiteOrName(float64(t))
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = jsonSafe(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = jsonSafe(e)
		}
		return out
	default:
		return v
	}
}

// finiteOrName passes a finite float through and names a non-finite one the
// way JavaScript and Python spell them.
func finiteOrName(f float64) any {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	default:
		return f
	}
}

// toNoteViews projects parsed notes into their JSON shape; the result is
// always non-nil so an issue with no notes still carries the key.
func toNoteViews(notes []issue.Note) []noteView {
	views := make([]noteView, len(notes))
	for i, n := range notes {
		views[i] = noteView{Author: n.Author, Time: formatTime(n.Time), Text: n.Text}
	}
	return views
}

// issueWithCommit is the base jsonView plus an always-present "commit" key:
// the completion commit when one was recorded, null otherwise.
type issueWithCommit struct {
	jsonView
	Commit *commitView `json:"commit"`
}

// commitView is the completion commit in JSON: the short revision the VCS
// adapter reported.
type commitView struct {
	Revision string `json:"revision"`
}

func toCommitView(revision string) *commitView {
	if revision == "" {
		return nil
	}
	return &commitView{Revision: revision}
}

// issueWithRel is the base jsonView plus an additive "relationships" object
// carrying the derived view.
type issueWithRel struct {
	jsonView
	Relationships relView `json:"relationships"`
}

// relView is the machine shape of a Relationship: the readiness booleans, the
// unmet dependencies, and the derived inverse edges. Lists are always present,
// empty rather than null.
type relView struct {
	Ready     bool          `json:"ready"`
	Blocked   bool          `json:"blocked"`
	Stuck     bool          `json:"stuck"`
	BlockedOn []blockerView `json:"blocked_on"`
	Blocks    []string      `json:"blocks"`
	Children  []string      `json:"children"`
}

// blockerView is one unmet dependency in JSON: the target id, its state (null
// for a dangling reference), and an explicit missing flag so consumers need
// not infer the case from a null state.
type blockerView struct {
	ID      string  `json:"id"`
	State   *string `json:"state"`
	Missing bool    `json:"missing"`
}

func toRelView(rel issue.Relationship) relView {
	blockers := make([]blockerView, len(rel.BlockedOn))
	for i, b := range rel.BlockedOn {
		var state *string
		if !b.Missing {
			s := string(b.State)
			state = &s
		}
		blockers[i] = blockerView{ID: b.ID, State: state, Missing: b.Missing}
	}
	return relView{
		Ready:     rel.Ready,
		Blocked:   rel.Blocked,
		Stuck:     rel.Stuck,
		BlockedOn: blockers,
		Blocks:    orEmpty(rel.Blocks),
		Children:  orEmpty(rel.Children),
	}
}

func writeHuman(w io.Writer, iss issue.Issue) error {
	var b strings.Builder
	writeHumanHead(&b, iss)
	writeHumanBody(&b, iss)
	_, err := io.WriteString(w, b.String())
	return err
}

// writeHumanHead renders the title line and the aligned "key  value" block of
// an issue's own fields — everything that precedes the body.
func writeHumanHead(b *strings.Builder, iss issue.Issue) {
	fmt.Fprintf(b, "%s  %s\n\n", iss.ID, iss.Title)

	field(b, "state", string(iss.State))
	if iss.Priority != "" {
		field(b, "priority", string(iss.Priority))
	}
	if iss.Assignee != "" {
		field(b, "assignee", iss.Assignee)
	}
	if len(iss.Labels) > 0 {
		field(b, "labels", strings.Join(iss.Labels, ", "))
	}
	if len(iss.DependsOn) > 0 {
		field(b, "depends_on", strings.Join(iss.DependsOn, ", "))
	}
	if iss.Parent != "" {
		field(b, "parent", iss.Parent)
	}
	// A zero timestamp means the file carries none; skip the line rather than
	// render a blank value.
	if !iss.Created.IsZero() {
		field(b, "created", formatTime(iss.Created))
	}
	if !iss.Updated.IsZero() {
		field(b, "updated", formatTime(iss.Updated))
	}
	for _, key := range issue.CustomKeys(iss.Custom) {
		field(b, key, formatCustomValue(iss.Custom[key]))
	}
}

// writeHumanBody appends the issue's Markdown body, if any, after a blank
// separator line, matching the on-disk layout.
func writeHumanBody(b *strings.Builder, iss issue.Issue) {
	if body := strings.TrimRight(iss.Body, "\n"); body != "" {
		fmt.Fprintf(b, "\n%s\n", body)
	}
}

// writeHumanRelationship appends the derived relationship section: a readiness
// word, the unmet dependencies, and the inverse edges. It writes nothing when
// there is nothing to say, so a simple issue reads as a plain one.
func writeHumanRelationship(b *strings.Builder, rel issue.Relationship) {
	status := statusWord(rel)
	if status == "" && len(rel.BlockedOn) == 0 && len(rel.Blocks) == 0 && len(rel.Children) == 0 {
		return
	}
	b.WriteByte('\n')
	if status != "" {
		field(b, "status", status)
	}
	for i, bl := range rel.BlockedOn {
		key := "" // continuation rows align their value under the first
		if i == 0 {
			key = "waiting on"
		}
		field(b, key, blockerLine(bl))
	}
	if len(rel.Blocks) > 0 {
		field(b, "blocks", strings.Join(rel.Blocks, ", "))
	}
	if len(rel.Children) > 0 {
		field(b, "children", strings.Join(rel.Children, ", "))
	}
}

// statusWord names the readiness for the human view, or "" when neither ready
// nor blocked applies and the state line already tells the whole story.
func statusWord(rel issue.Relationship) string {
	switch {
	case rel.Stuck:
		return "stuck (waiting on a cancelled issue)"
	case rel.Ready:
		return "ready"
	case rel.Blocked:
		return "blocked"
	default:
		return ""
	}
}

// blockerLine renders one unmet dependency as "<id>  <state>", or
// "<id>  (missing)" for a dangling reference.
func blockerLine(b issue.Blocker) string {
	if b.Missing {
		return b.ID + "  (missing)"
	}
	return b.ID + "  " + string(b.State)
}

// field writes one aligned "key  value" line of the human rendering.
func field(b *strings.Builder, key, val string) {
	fmt.Fprintf(b, "  %-11s%s\n", key, val)
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func orEmpty(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// formatCustomValue renders a custom value for the human view: scalars print
// plainly; sequences and maps print as compact JSON to stay one line per field.
func formatCustomValue(v any) string {
	switch v.(type) {
	case nil:
		return ""
	case []any, map[string]any:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
