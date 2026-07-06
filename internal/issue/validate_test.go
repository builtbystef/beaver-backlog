package issue_test

import (
	"strings"
	"testing"
	"time"

	"github.com/builtbystef/busy-beaver/internal/issue"
)

// A well-formed id and a legal state make an issue valid. The id check turns on
// the character set, not the length, so a shorter or longer id stays valid.
func TestValidateAccepts(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	base := issue.Issue{ID: "a1b2c3", Title: "Fine", State: issue.StateTodo, Created: now, Updated: now}

	for _, state := range []issue.State{issue.StateTodo, issue.StateInProgress, issue.StateDone, issue.StateCancelled} {
		iss := base
		iss.State = state
		if err := issue.Validate(iss); err != nil {
			t.Errorf("state %q rejected: %v", state, err)
		}
	}
	for _, id := range []string{"m3k8" /* shorter */, "abcdef", "012345", "a1b2c3d4e5" /* longer */} {
		iss := base
		iss.ID = id
		if err := issue.Validate(iss); err != nil {
			t.Errorf("id %q rejected: %v", id, err)
		}
	}
}

// A missing or malformed id or state is a hard validation error, and the
// message names the specific defect.
func TestValidateRejects(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	ok := issue.Issue{ID: "a1b2c3", Title: "T", State: issue.StateTodo, Created: now, Updated: now}

	cases := []struct {
		name    string
		mutate  func(*issue.Issue)
		wantSub string
	}{
		{"empty id", func(i *issue.Issue) { i.ID = "" }, "id"},
		{"uppercase id", func(i *issue.Issue) { i.ID = "ABC123" }, "malformed id"},
		{"hyphen in id", func(i *issue.Issue) { i.ID = "a1-b2c" }, "malformed id"},
		{"space in id", func(i *issue.Issue) { i.ID = "a1 b2c" }, "malformed id"},
		{"empty state", func(i *issue.Issue) { i.State = "" }, "state"},
		{"illegal state", func(i *issue.Issue) { i.State = "archived" }, "invalid state"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			iss := ok
			c.mutate(&iss)
			err := issue.Validate(iss)
			if err == nil {
				t.Fatalf("Validate accepted %s, want an error", c.name)
			}
			if !strings.Contains(err.Error(), c.wantSub) {
				t.Errorf("error %q does not mention %q", err, c.wantSub)
			}
		})
	}
}
