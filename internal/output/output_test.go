package output_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"beaver/internal/issue"
	"beaver/internal/output"
)

func TestResolve(t *testing.T) {
	none := func(string) string { return "" }
	agent := func(k string) string {
		if k == "AGENT" {
			return "claude"
		}
		return ""
	}

	cases := []struct {
		name     string
		override string
		tty      bool
		getenv   func(string) string
		want     output.Format
		wantErr  bool
	}{
		{"override human wins over pipe", "human", false, none, output.Human, false},
		{"override json wins over tty", "json", true, none, output.JSON, false},
		{"interactive tty -> human", "", true, none, output.Human, false},
		{"non-interactive pipe -> json", "", false, none, output.JSON, false},
		{"agent at tty -> json", "", true, agent, output.JSON, false},
		{"invalid override errors", "xml", true, none, "", true},
	}
	for _, c := range cases {
		got, err := output.Resolve(c.override, c.tty, c.getenv)
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

// TestWriteIssueJSONNormalizesUnset checks the JSON contract: unset single-valued
// fields are null, unset lists are [], timestamps are RFC3339.
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
