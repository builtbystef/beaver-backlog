package web_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The doctor page is the health report made readable: every finding with the
// facts behind it, and unusable files among them rather than banished to a
// banner.
func TestDoctorPageRendersTheReport(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	drifted := create(t, svc, core.Draft{Title: "Wandered off"})
	canonical := filepath.Base(issueFile(t, dir, drifted.ID))
	rename(t, dir, drifted.ID, "wandered.md")
	typo := create(t, svc, core.Draft{Title: "Custom fields"})
	addCustomField(t, dir, typo.ID, "stat", "todo")
	gone := create(t, svc, core.Draft{Title: "Deleted later"})
	orphan := create(t, svc, core.Draft{Title: "Left pointing at nothing", DependsOn: []string{gone.ID}})
	if _, err := svc.Delete(gone.ID); err != nil {
		t.Fatalf("delete %s: %v", gone.ID, err)
	}
	abandoned := create(t, svc, core.Draft{Title: "Never happening"})
	stuck := create(t, svc, core.Draft{Title: "Waiting forever", DependsOn: []string{abandoned.ID}})
	transition(t, svc, abandoned.ID, issue.StateCancelled)
	writeFile(t, dir, "broken.md", "not an issue file")

	res := get(newHandler(t, dir), "/doctor")

	if res.Code != http.StatusOK {
		t.Fatalf("GET /doctor = %d, want 200", res.Code)
	}
	body := res.Body.String()
	found := findings(t, body)
	for _, category := range []string{"invalid", "filename_drift", "unknown_key", "dangling_reference", "stuck"} {
		if len(found[category]) == 0 {
			t.Errorf("no %s finding on the page:\n%s", category, body)
		}
	}
	// Every finding names the files and issues it concerns, and states the fact
	// its category turns on.
	for _, want := range []string{
		"broken.md",              // invalid: the file
		"wandered.md", canonical, // drift: where it sits and where it belongs
		typo.ID, "stat", "state", // likely typo: the key and what it resembles
		orphan.ID, gone.ID, // dangling: the issue and the target no issue holds
		stuck.ID, abandoned.ID, // stuck: the issue and its cancelled blocker
	} {
		if !strings.Contains(body, want) {
			t.Errorf("report does not mention %q:\n%s", want, body)
		}
	}
}

// The core decides which findings are only advisory; the page shows the
// difference.
func TestDoctorSeparatesAdvisoriesFromProblems(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	drifted := create(t, svc, core.Draft{Title: "Wandered off"})
	rename(t, dir, drifted.ID, "wandered.md")
	typo := create(t, svc, core.Draft{Title: "Custom fields"})
	addCustomField(t, dir, typo.ID, "stat", "todo")

	found := findings(t, get(newHandler(t, dir), "/doctor").Body.String())

	if got := found["unknown_key"]; len(got) != 1 || got[0] != "advisory" {
		t.Errorf("unknown_key renders as %v, want one advisory", got)
	}
	if got := found["filename_drift"]; len(got) != 1 || got[0] != "problem" {
		t.Errorf("filename_drift renders as %v, want one problem", got)
	}
}

func TestDoctorAllClearOnAHealthyStore(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Nothing wrong here"})

	res := get(newHandler(t, dir), "/doctor")

	if res.Code != http.StatusOK {
		t.Fatalf("GET /doctor = %d, want 200", res.Code)
	}
	body := res.Body.String()
	if len(findings(t, body)) != 0 {
		t.Errorf("a healthy store renders findings:\n%s", body)
	}
	if !strings.Contains(body, "all-clear") {
		t.Errorf("a healthy store does not render an all-clear:\n%s", body)
	}
	if strings.Contains(body, "/doctor/fix") {
		t.Error("a healthy store offers a repair button")
	}
}

// The worked example: a file hand-renamed away from its canonical name is
// repaired in place, and nothing is removed.
func TestDoctorFixRenamesTheDriftedFile(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	drifted := create(t, svc, core.Draft{Title: "Wandered off"})
	canonical := filepath.Base(issueFile(t, dir, drifted.ID))
	rename(t, dir, drifted.ID, "wandered.md")
	h := newHandler(t, dir)
	if body := get(h, "/doctor").Body.String(); !strings.Contains(body, "/doctor/fix") {
		t.Fatalf("no repair button while a fixable finding stands:\n%s", body)
	}

	res := post(h, "/doctor/fix", url.Values{})

	if res.Code != http.StatusOK {
		t.Fatalf("POST /doctor/fix = %d, want 200", res.Code)
	}
	if got := filepath.Base(issueFile(t, dir, drifted.ID)); got != canonical {
		t.Errorf("file sits at %s, want the canonical %s", got, canonical)
	}
	if fetch(t, svc, drifted.ID).Title != "Wandered off" {
		t.Error("the repaired issue lost its content")
	}
	if body := res.Body.String(); !strings.Contains(body, canonical) {
		t.Errorf("the repair does not report what it renamed:\n%s", body)
	}
	if found := findings(t, get(h, "/doctor").Body.String()); len(found["filename_drift"]) != 0 {
		t.Errorf("the drift finding survives the repair: %v", found)
	}
}

// Nothing mechanically safe to do means no button to press: every finding here
// needs a human.
func TestDoctorOffersNoRepairWhenNothingIsFixable(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	gone := create(t, svc, core.Draft{Title: "Deleted later"})
	create(t, svc, core.Draft{Title: "Left pointing at nothing", DependsOn: []string{gone.ID}})
	if _, err := svc.Delete(gone.ID); err != nil {
		t.Fatalf("delete %s: %v", gone.ID, err)
	}

	body := get(newHandler(t, dir), "/doctor").Body.String()

	if len(findings(t, body)["dangling_reference"]) == 0 {
		t.Fatalf("the dangling reference is missing from the report:\n%s", body)
	}
	if strings.Contains(body, "/doctor/fix") {
		t.Errorf("a repair button appears with nothing fixable:\n%s", body)
	}
}

var findingMark = regexp.MustCompile(`<li class="finding ([a-z]+)" data-category="([a-z_]+)"`)

// findings reads the rendered report back as what it depicts: each category
// present, and how each of its findings was classed.
func findings(t *testing.T, body string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, m := range findingMark.FindAllStringSubmatch(body, -1) {
		out[m[2]] = append(out[m[2]], m[1])
	}
	return out
}

// issueFile is the one file in the store holding an issue's id.
func issueFile(t *testing.T, dir, id string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".beaver", "issues", id+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("find file for %s: %v (matched %v)", id, err, matches)
	}
	return matches[0]
}

// rename moves an issue's file the way a hand-edit would, drifting it from the
// canonical name its frontmatter implies.
func rename(t *testing.T, dir, id, name string) {
	t.Helper()
	if err := os.Rename(issueFile(t, dir, id), filepath.Join(dir, ".beaver", "issues", name)); err != nil {
		t.Fatalf("rename %s: %v", id, err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".beaver", "issues", name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
