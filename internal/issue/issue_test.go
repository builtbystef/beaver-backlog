package issue_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"beaver/internal/issue"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Login form rejects valid passwords":       "login-form-rejects-valid-passwords",
		"Walking skeleton: init, create, show":     "walking-skeleton-init-create-show",
		"Ownership: claim, assign, release, start": "ownership-claim-assign-release-start",
		"  Leading and trailing punctuation!!  ":   "leading-and-trailing-punctuation",
		"Multiple   internal   spaces":             "multiple-internal-spaces",
		"CamelCase Title":                          "camelcase-title",
		"Has 123 numbers":                          "has-123-numbers",
		"":                                         "",
		"!!!":                                      "",
		"日本語のタイトル":                                 "", // non-ASCII collapses to empty
	}
	for title, want := range cases {
		if got := issue.Slug(title); got != want {
			t.Errorf("Slug(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestSlugTruncatesOnWordBoundary(t *testing.T) {
	s := issue.Slug(strings.Repeat("word ", 40)) // far over the length cap
	if len(s) > 60 {
		t.Errorf("slug length %d exceeds cap: %q", len(s), s)
	}
	if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") || strings.Contains(s, "--") {
		t.Errorf("slug has edge or doubled hyphen: %q", s)
	}
}

func TestFileName(t *testing.T) {
	if got := issue.FileName("m3k8", "some-slug"); got != "m3k8-some-slug.md" {
		t.Errorf("FileName with slug = %q", got)
	}
	if got := issue.FileName("m3k8", ""); got != "m3k8.md" {
		t.Errorf("FileName with empty slug = %q", got)
	}
}

func TestIDFromFileName(t *testing.T) {
	cases := map[string]string{
		"m3k8-walking-skeleton.md": "m3k8",
		"m3k8.md":                  "m3k8",
		"m3k8":                     "m3k8",
	}
	for name, want := range cases {
		if got := issue.IDFromFileName(name); got != want {
			t.Errorf("IDFromFileName(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestNewIDFormat asserts IDs are short, lowercase-alphanumeric, and actually
// random (ADR 0002).
func TestNewIDFormat(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	seen := make(map[string]bool)
	for range 1000 {
		id := issue.NewID()
		if len(id) != 4 {
			t.Fatalf("id %q has length %d, want 4", id, len(id))
		}
		for _, r := range id {
			if !strings.ContainsRune(alphabet, r) {
				t.Fatalf("id %q contains %q, which is not lowercase-alphanumeric", id, r)
			}
		}
		seen[id] = true
	}
	if len(seen) < 200 {
		t.Errorf("only %d distinct IDs in 1000 draws — not random enough", len(seen))
	}
}

// TestFrontmatterRoundTrip checks that an issue with every field set survives a
// Marshal/Unmarshal cycle, including a body that itself contains "---" fences.
func TestFrontmatterRoundTrip(t *testing.T) {
	created := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	in := issue.Issue{
		ID:        "m3k8",
		Title:     "Walking skeleton: init, create, show",
		State:     issue.StateInProgress,
		Assignee:  "stefan",
		Priority:  issue.PriorityHigh,
		Labels:    []string{"feature", "cli"},
		DependsOn: []string{"r7p2", "h5t1"},
		Parent:    "z4m9",
		Created:   created,
		Updated:   created.Add(time.Hour),
		Body:      "Body text.\n\n---\nThis is a horizontal rule, not frontmatter.\n---\n\nDone.\n",
	}

	data, err := issue.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out, err := issue.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !out.Created.Equal(in.Created) || !out.Updated.Equal(in.Updated) {
		t.Errorf("timestamps not preserved: created %v/%v updated %v/%v",
			out.Created, in.Created, out.Updated, in.Updated)
	}
	out.Created, out.Updated = in.Created, in.Updated // normalize for whole-struct compare
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip mismatch:\n in = %+v\nout = %+v\nfile:\n%s", in, out, data)
	}
}

// TestMarshalOmitsUnsetOptionals verifies that a minimal issue writes only the
// required fields, in canonical order, with plain RFC3339 timestamps.
func TestMarshalOmitsUnsetOptionals(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	data, err := issue.Marshal(issue.Issue{
		ID: "m3k8", Title: "Title", State: issue.StateTodo, Created: now, Updated: now,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)

	for _, omitted := range []string{"assignee", "priority", "labels", "depends_on", "parent"} {
		if strings.Contains(s, omitted) {
			t.Errorf("expected %q to be omitted, file:\n%s", omitted, s)
		}
	}
	if !strings.Contains(s, "created: "+now.Format(time.RFC3339)+"\n") {
		t.Errorf("timestamp not plain RFC3339, file:\n%s", s)
	}
	if !canonicalOrder(s, "id:", "title:", "state:", "created:", "updated:") {
		t.Errorf("fields not in canonical order, file:\n%s", s)
	}
}

// TestCustomFieldsSurviveRoundTrip is the core guarantee of ADR 0014: a
// hand-added frontmatter key Busy Beaver knows nothing about is preserved through a
// read-modify-write, not silently dropped. It lands in Custom on read, a command
// mutates a known field, and the custom key is still there on write.
func TestCustomFieldsSurviveRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	src := "---\n" +
		"id: m3k8\n" +
		"title: Title\n" +
		"state: todo\n" +
		"sprint: 7\n" +
		"estimate: 3d\n" +
		"reviewers:\n" +
		"    - stefan\n" +
		"    - claude\n" +
		"created: " + now.Format(time.RFC3339) + "\n" +
		"updated: " + now.Format(time.RFC3339) + "\n" +
		"---\n\nBody.\n"

	iss, err := issue.Unmarshal([]byte(src))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := iss.Custom["sprint"]; got != 7 {
		t.Errorf("Custom[sprint] = %v (%T), want 7", got, got)
	}
	if _, ok := iss.Custom["estimate"]; !ok {
		t.Errorf("Custom missing estimate: %v", iss.Custom)
	}
	if _, ok := iss.Custom["reviewers"]; !ok {
		t.Errorf("Custom missing reviewers: %v", iss.Custom)
	}

	// A command mutates a known field; the custom keys must ride along.
	iss.State = issue.StateDone
	out, err := issue.Marshal(iss)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(out)
	for _, want := range []string{"sprint: 7", "estimate: 3d", "reviewers:", "- stefan", "- claude"} {
		if !strings.Contains(s, want) {
			t.Errorf("custom field %q dropped on save, file:\n%s", want, s)
		}
	}
	// Custom keys follow the machine-owned fields, never before them.
	if !canonicalOrder(s, "id:", "title:", "state:", "created:", "updated:", "sprint:") {
		t.Errorf("custom keys not placed after machine-owned fields, file:\n%s", s)
	}
}

// TestNoCustomFieldsLeavesCustomNil guards the DeepEqual round-trip contract: an
// issue with no user keys unmarshals to a nil Custom map, not an empty one.
func TestNoCustomFieldsLeavesCustomNil(t *testing.T) {
	now := time.Date(2026, 6, 27, 18, 30, 0, 0, time.UTC)
	data, err := issue.Marshal(issue.Issue{
		ID: "m3k8", Title: "Title", State: issue.StateTodo, Created: now, Updated: now,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	iss, err := issue.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if iss.Custom != nil {
		t.Errorf("Custom = %v, want nil for a file with no custom keys", iss.Custom)
	}
}

func TestUnmarshalReportsMalformed(t *testing.T) {
	if _, err := issue.Unmarshal([]byte("no frontmatter at all")); !errors.Is(err, issue.ErrMissingFrontmatter) {
		t.Errorf("missing frontmatter: got %v, want ErrMissingFrontmatter", err)
	}
	if _, err := issue.Unmarshal([]byte("---\nid: m3k8\n")); !errors.Is(err, issue.ErrUnterminatedFrontmatter) {
		t.Errorf("unterminated frontmatter: got %v, want ErrUnterminatedFrontmatter", err)
	}
	if _, err := issue.Unmarshal([]byte("---\nid: [unclosed\n---\n")); err == nil {
		t.Errorf("invalid YAML should error")
	}
}

// canonicalOrder reports whether the substrings appear in s in the given order.
func canonicalOrder(s string, subs ...string) bool {
	last := -1
	for _, sub := range subs {
		i := strings.Index(s, sub)
		if i <= last {
			return false
		}
		last = i
	}
	return true
}
