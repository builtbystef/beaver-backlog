// Package output renders issues for two audiences: a human reading a terminal,
// and a machine (or agent) consuming JSON. The format is auto-detected and
// overridable (ADR 0013)..
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
// non-interactive pipe or a detected agent.
func Resolve(override string, stdoutIsTTY bool, getenv func(string) string) (Format, error) {
	switch override {
	case string(Human):
		return Human, nil
	case string(JSON):
		return JSON, nil
	case "":
		if !stdoutIsTTY || agentDetected(getenv) {
			return JSON, nil
		}
		return Human, nil
	default:
		return "", fmt.Errorf("invalid format %q (want human or json)", override)
	}
}

// agentDetected is a minimal check used only to bias output toward JSON for
// agents. Full actor/agent resolution (ADR 0010) arrives with the identity slice
// and will centralize this; output will defer to it then.
func agentDetected(getenv func(string) string) bool {
	if getenv == nil {
		return false
	}
	return getenv("AGENT") != "" || getenv("CLAUDECODE") != ""
}

// WriteIssue renders a single issue in the given format.
func WriteIssue(w io.Writer, iss issue.Issue, f Format) error {
	if f == JSON {
		return WriteJSON(w, toJSONView(iss))
	}
	return writeHuman(w, iss)
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
	}
}

func writeHuman(w io.Writer, iss issue.Issue) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n\n", iss.ID, iss.Title)

	field(&b, "state", string(iss.State))
	if iss.Priority != "" {
		field(&b, "priority", string(iss.Priority))
	}
	if iss.Assignee != "" {
		field(&b, "assignee", iss.Assignee)
	}
	if len(iss.Labels) > 0 {
		field(&b, "labels", strings.Join(iss.Labels, ", "))
	}
	if len(iss.DependsOn) > 0 {
		field(&b, "depends_on", strings.Join(iss.DependsOn, ", "))
	}
	if iss.Parent != "" {
		field(&b, "parent", iss.Parent)
	}
	field(&b, "created", formatTime(iss.Created))
	field(&b, "updated", formatTime(iss.Updated))

	if body := strings.TrimRight(iss.Body, "\n"); body != "" {
		fmt.Fprintf(&b, "\n%s\n", body)
	}

	_, err := io.WriteString(w, b.String())
	return err
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

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
