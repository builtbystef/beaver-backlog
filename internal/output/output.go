// Package output renders issues for two audiences: a human reading a terminal,
// and a machine (or agent) consuming JSON. The format is auto-detected and
// overridable (ADR 0013).
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"beaver/internal/issue"
)

// Format is a rendering style.
type Format string

const (
	Human Format = "human"
	JSON  Format = "json"
)

// Resolve picks the output format. An explicit override ("human"/"json") always
// wins; otherwise output is human for an interactive terminal and JSON for a
// non-interactive pipe or a detected agent. Whether an agent is present is decided
// by the caller — via the one agent registry that also drives identity resolution
// (ADR 0010) — and passed in, rather than sniffed from the environment here.
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
// relationship view — what it is waiting on, whether it is ready/blocked/stuck,
// and the inverse edges Busy Beaver never stores (what it blocks, its children).
// show uses it to answer "can I start this, and if not, why", and start emits it
// (JSON only) so an agent sees whether the work it just began was blocked and on
// what. The plain WriteIssue that create, the transitions, and the reserving
// ownership verbs emit stays a pure projection of the stored fields; only the two
// dependency-aware commands carry the derived section, and in JSON it is an
// additive "relationships" object beside the same issue fields, so a consumer
// reading the base fields sees an unchanged shape.
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

// WriteList renders a collection of issues in the given format, preserving the
// order the caller supplies. JSON is an array of the same per-issue objects
// WriteIssue emits — every field present, unset ones normalized to null/empty —
// so an agent listing issues gets complete records. Human output is an aligned
// ID/STATE/TITLE table.
func WriteList(w io.Writer, issues []issue.Issue, f Format) error {
	if f == JSON {
		views := make([]jsonView, len(issues)) // non-nil, so an empty list is [] not null
		for i, iss := range issues {
			views[i] = toJSONView(iss)
		}
		return WriteJSON(w, views)
	}
	return writeTable(w, issues)
}

// writeTable renders issues as an aligned, columnar human table. The header and
// column set are human output, not a contract (ADR 0013); later slices widen them
// as priority, labels, and assignee become settable (p1k765).
func writeTable(w io.Writer, issues []issue.Issue) error {
	if len(issues) == 0 {
		_, err := io.WriteString(w, "No issues.\n")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATE\tTITLE")
	for _, iss := range issues {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", iss.ID, iss.State, oneLine(iss.Title))
	}
	return tw.Flush()
}

// oneLine flattens a value to a single line for a table cell, so a newline or tab
// spliced into a title by a hand-edit or merge cannot break the column grid (a tab
// is what tabwriter uses to delimit columns).
func oneLine(s string) string {
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

// jsonView is the stable machine shape of an issue. Unlike the on-disk file,
// every field is always present: unset single-valued options serialize to null
// and unset lists to [], so consumers need not special-case missing keys.
type jsonView struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Assignee  *string  `json:"assignee"`
	Priority  *string  `json:"priority"`
	Labels    []string `json:"labels"`
	DependsOn []string `json:"depends_on"`
	Parent    *string  `json:"parent"`
	Created   string   `json:"created"`
	Updated   string   `json:"updated"`
	Body      string   `json:"body"`
	// Notes is the append-only log parsed out of the body's Notes section (ADR 0012)
	// into structured entries, so an agent reads attributed, timestamped handoffs
	// without re-parsing the Markdown. The raw entries also remain in Body verbatim.
	// Always present as an array, empty when the issue has no notes, per the
	// no-missing-keys contract (ADR 0013).
	Notes []noteView `json:"notes"`
	// Custom carries user-defined frontmatter keys Busy Beaver preserves but does not
	// interpret (ADR 0014). Always present as an object, empty when the issue has
	// none, so consumers never special-case a missing key.
	Custom map[string]any `json:"custom"`
}

// noteView is one note in JSON: who wrote it, when (RFC3339 UTC, like the issue's own
// timestamps), and its text.
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
		Created:   formatTime(iss.Created),
		Updated:   formatTime(iss.Updated),
		Body:      iss.Body,
		Notes:     toNoteViews(issue.ParseNotes(iss.Body)),
		Custom:    orEmptyMap(iss.Custom),
	}
}

// toNoteViews projects parsed notes into their JSON shape. The result is always
// non-nil (an empty [] rather than null) so an issue with no notes still carries the
// key, matching the no-missing-keys contract (ADR 0013).
func toNoteViews(notes []issue.Note) []noteView {
	views := make([]noteView, len(notes))
	for i, n := range notes {
		views[i] = noteView{Author: n.Author, Time: formatTime(n.Time), Text: n.Text}
	}
	return views
}

// issueWithRel is show's JSON shape: every field of the base jsonView, plus an
// additive "relationships" object carrying the derived view. Embedding jsonView
// inlines its fields, so a consumer reading the base fields sees the same shape
// create and list emit, with the derived object beside them.
type issueWithRel struct {
	jsonView
	Relationships relView `json:"relationships"`
}

// relView is the machine shape of a Relationship: the three readiness booleans,
// the unmet dependencies (each with its target's state, or null when the target is
// missing), and the derived inverse edges. Lists are always present — empty, never
// null — so consumers never special-case a missing key.
type relView struct {
	Ready     bool          `json:"ready"`
	Blocked   bool          `json:"blocked"`
	Stuck     bool          `json:"stuck"`
	BlockedOn []blockerView `json:"blocked_on"`
	Blocks    []string      `json:"blocks"`
	Children  []string      `json:"children"`
}

// blockerView is one unmet dependency in JSON: the target id, its state (null when
// the target is a dangling reference), and a missing flag that says which case it
// is without the consumer inferring it from a null state.
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

// writeHumanHead renders the title line and the aligned "key  value" block of an
// issue's own fields — everything that precedes the body. show layers the derived
// relationship section between this and the body (writeHumanRelationship).
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
	field(b, "created", formatTime(iss.Created))
	field(b, "updated", formatTime(iss.Updated))
	for _, key := range sortedKeys(iss.Custom) {
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
// word, the dependencies the issue is waiting on (each with the state that keeps
// it unmet), and the inverse edges. It writes nothing when there is nothing to say
// — an unblocked issue with no dependents or children — so a simple issue reads
// exactly as it did before this section existed.
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

// statusWord names iss's readiness for the human view, or "" when neither ready
// nor blocked applies — an in-progress issue whose dependencies are all done, or a
// closed one — and the state line already tells the whole story.
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

// blockerLine renders one unmet dependency as "<id>  <state>", or "<id>  (missing)"
// when the target is a dangling reference.
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

func orEmptyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

// sortedKeys returns m's keys in a stable order so the human rendering of custom
// fields is deterministic, matching the sorted order the YAML encoder writes.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// formatCustomValue renders a preserved custom value for the human view: scalars
// print plainly; sequences and maps, which have no single-line "key  value" form,
// print as compact JSON so the glance stays one line per field.
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
