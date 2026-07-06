package output_test

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/busy-beaver/internal/issue"
	"github.com/builtbystef/busy-beaver/internal/output"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		name     string
		override string
		tty      bool
		agent    bool
		want     output.Format
		wantErr  bool
	}{
		{"override human wins over pipe", "human", false, false, output.Human, false},
		{"override json wins over tty", "json", true, false, output.JSON, false},
		{"interactive tty -> human", "", true, false, output.Human, false},
		{"non-interactive pipe -> json", "", false, false, output.JSON, false},
		{"agent at tty -> json", "", true, true, output.JSON, false},
		{"invalid override errors", "xml", true, false, "", true},
	}
	for _, c := range cases {
		got, err := output.Resolve(c.override, c.tty, c.agent)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: expected error", c.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// The JSON contract: unset single-valued fields are null, unset lists are [],
// timestamps are RFC3339.
func TestWriteIssueJSONNormalizesUnset(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	iss := issue.Issue{ID: "m3k8", Title: "Title", State: issue.StateTodo, Created: now, Updated: now}

	var buf bytes.Buffer
	if err := output.WriteIssue(&buf, iss, output.JSON); err != nil {
		t.Fatalf("WriteIssue: %v", err)
	}
	s := buf.String()

	for _, want := range []string{
		`"assignee": null`, `"priority": null`, `"parent": null`,
		`"labels": []`, `"depends_on": []`, `"body": ""`,
		`"custom": {}`,
		`"created": "2026-06-27T18:30:00Z"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %s in:\n%s", want, s)
		}
	}
}

func TestWriteIssueHuman(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	iss := issue.Issue{ID: "m3k8", Title: "My title", State: issue.StateTodo, Created: now, Updated: now, Body: "Hello body\n"}

	var buf bytes.Buffer
	if err := output.WriteIssue(&buf, iss, output.Human); err != nil {
		t.Fatalf("WriteIssue: %v", err)
	}
	s := buf.String()

	for _, want := range []string{"m3k8", "My title", "todo", "Hello body"} {
		if !strings.Contains(s, want) {
			t.Errorf("human output missing %q in:\n%s", want, s)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(s), "{") {
		t.Errorf("human output should not be JSON:\n%s", s)
	}
}

// Preserved user-defined fields are visible in both renderings: scalars plain
// and sequences as compact JSON in the human view, verbatim under "custom" in JSON.
func TestWriteIssueRendersCustomFields(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	iss := issue.Issue{
		ID: "m3k8", Title: "Title", State: issue.StateTodo, Created: now, Updated: now,
		Custom: map[string]any{"sprint": 7, "reviewers": []any{"stefan", "claude"}},
	}

	var human bytes.Buffer
	if err := output.WriteIssue(&human, iss, output.Human); err != nil {
		t.Fatalf("WriteIssue human: %v", err)
	}
	for _, want := range []string{"sprint", "7", "reviewers", `["stefan","claude"]`} {
		if !strings.Contains(human.String(), want) {
			t.Errorf("human output missing custom %q in:\n%s", want, human.String())
		}
	}

	var jsn bytes.Buffer
	if err := output.WriteIssue(&jsn, iss, output.JSON); err != nil {
		t.Fatalf("WriteIssue json: %v", err)
	}
	if !strings.Contains(jsn.String(), `"sprint": 7`) {
		t.Errorf("JSON output missing custom sprint in:\n%s", jsn.String())
	}
}

// YAML admits non-finite floats (.nan, ±.inf) that encoding/json refuses; a
// custom value carrying one must not fail the write, so they render as their
// conventional names, wherever they nest.
func TestWriteIssueJSONSurvivesNonFiniteCustomValues(t *testing.T) {
	iss := issue.Issue{
		ID: "m3k8", Title: "Title", State: issue.StateTodo,
		Custom: map[string]any{
			"weight": math.NaN(),
			"bounds": []any{math.Inf(1), 1.5},
			"nested": map[string]any{"low": math.Inf(-1)},
		},
	}

	var buf bytes.Buffer
	if err := output.WriteIssue(&buf, iss, output.JSON); err != nil {
		t.Fatalf("WriteIssue with non-finite custom values: %v", err)
	}
	s := buf.String()
	for _, want := range []string{`"weight": "NaN"`, `"Infinity"`, `"low": "-Infinity"`, `1.5`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %s in:\n%s", want, s)
		}
	}
}
