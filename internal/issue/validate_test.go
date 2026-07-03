package issue_test

import (
	"strings"
	"testing"
	"time"

	"beaver/internal/issue"
)

// A well-formed id and a legal state make an issue valid. The id check turns on
// the character set, not the length, so a legacy short id minted before the
// length was widened (ADR 0002) stays valid rather than being read as broken.
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
	for _, id := range []string{"m3k8" /* legacy 4-char */, "abcdef", "012345", "a1b2c3d4e5" /* longer */} {
		iss := base
		iss.ID = id
		if err := issue.Validate(iss); err != nil {
			t.Errorf("id %q rejected: %v", id, err)
		}
	}
}

// A file that parses but is not a usable issue — no id, a malformed id, or an
// illegal/absent state — is a hard validation error. The message names the
// specific defect so the store can pair it with the file name (ADR 0005).
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
