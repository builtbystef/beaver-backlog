// Package ci guards the GitHub Actions workflows. Releasing is one push of a
// tag, so the release workflow's contract (what triggers it, what it may write,
// what it runs) is asserted here rather than discovered by pushing a tag. The
// site workflow is the same idea: a broken docs build must fail before merge,
// so the trigger, the lockfile install, and the build step are asserted here.
package ci

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	workflowDir     = "../../.github/workflows"
	releaseWorkflow = "release.yml"
	ciWorkflow      = "ci.yml"
	siteWorkflow    = "site.yml"
)

type workflow struct {
	On          map[string]yaml.Node `yaml:"on"`
	Permissions map[string]string    `yaml:"permissions"`
	Jobs        map[string]job       `yaml:"jobs"`
}

type job struct {
	Permissions map[string]string `yaml:"permissions"`
	Steps       []step            `yaml:"steps"`
}

type step struct {
	Name             string               `yaml:"name"`
	Uses             string               `yaml:"uses"`
	Run              string               `yaml:"run"`
	WorkingDirectory string               `yaml:"working-directory"`
	With             map[string]yaml.Node `yaml:"with"`
	Env              map[string]string    `yaml:"env"`
}

func load(t *testing.T, name string) workflow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(workflowDir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s is not a valid workflow: %v", name, err)
	}
	return w
}

func (w workflow) steps() []step {
	var all []step
	for _, j := range w.Jobs {
		all = append(all, j.Steps...)
	}
	return all
}

func TestReleaseTriggersOnlyOnVersionTags(t *testing.T) {
	w := load(t, releaseWorkflow)

	if len(w.On) != 1 {
		t.Fatalf("triggers %v, want push alone: a tag is the only thing that publishes a release", keys(w.On))
	}
	node, ok := w.On["push"]
	if !ok {
		t.Fatalf("triggers %v, want push", keys(w.On))
	}

	var push struct {
		Tags     []string `yaml:"tags"`
		Branches []string `yaml:"branches"`
	}
	if err := node.Decode(&push); err != nil {
		t.Fatalf("decode push trigger: %v", err)
	}
	if got, want := strings.Join(push.Tags, ","), "v*"; got != want {
		t.Errorf("push tags %q, want %q", got, want)
	}
	if len(push.Branches) != 0 {
		t.Errorf("push branches %v, want none: a branch push must not publish", push.Branches)
	}
}

func TestReleaseGrantsContentsWriteAndNothingMore(t *testing.T) {
	w := load(t, releaseWorkflow)

	perms := w.Permissions
	for name, j := range w.Jobs {
		if j.Permissions == nil {
			continue
		}
		if perms != nil {
			t.Fatalf("job %q sets permissions and so does the workflow: keep one place", name)
		}
		perms = j.Permissions
	}
	if len(perms) != 1 || perms["contents"] != "write" {
		t.Errorf("permissions %v, want contents: write alone", perms)
	}
}

func TestReleaseChecksOutFullHistoryAndTags(t *testing.T) {
	w := load(t, releaseWorkflow)

	for _, s := range w.steps() {
		if !strings.HasPrefix(s.Uses, "actions/checkout@") {
			continue
		}
		depth, ok := s.With["fetch-depth"]
		if !ok {
			t.Fatal("checkout sets no fetch-depth: a shallow clone has no tags to derive the version and notes from")
		}
		if depth.Value != "0" {
			t.Errorf("checkout fetch-depth %q, want 0", depth.Value)
		}
		return
	}
	t.Fatal("no checkout step")
}

func TestReleaseRunsGoReleaserInReleaseMode(t *testing.T) {
	w := load(t, releaseWorkflow)

	var found bool
	for _, s := range w.steps() {
		if !strings.Contains(s.Run, "goreleaser release") {
			continue
		}
		found = true
		for _, forbidden := range []string{"--snapshot", "--skip"} {
			if strings.Contains(s.Run, forbidden) {
				t.Errorf("release step passes %s, so it would publish nothing: %q", forbidden, s.Run)
			}
		}
		if strings.Contains(s.Run, "-f") || strings.Contains(s.Run, "--config") {
			t.Errorf("release step names another configuration: %q, want the .goreleaser.yaml CI checks", s.Run)
		}
		if token := s.Env["GITHUB_TOKEN"]; !strings.Contains(token, "github.token") && !strings.Contains(token, "secrets.GITHUB_TOKEN") {
			t.Errorf("release step's GITHUB_TOKEN is %q, want the token Actions already provides", token)
		}
	}
	if !found {
		t.Fatal("no step runs goreleaser release")
	}
	if _, err := os.Stat("../../.goreleaser.yaml"); err != nil {
		t.Errorf("the configuration the release runs against is missing: %v", err)
	}
}

func TestReleaseTakesItsGoVersionFromGoMod(t *testing.T) {
	w := load(t, releaseWorkflow)

	for _, s := range w.steps() {
		if !strings.HasPrefix(s.Uses, "actions/setup-go@") {
			continue
		}
		if file := s.With["go-version-file"]; file.Value != "go.mod" {
			t.Errorf("setup-go go-version-file %q, want go.mod", file.Value)
		}
		return
	}
	t.Fatal("no setup-go step")
}

// The pinning convention of the existing workflows, held to across every one:
// a full commit SHA, with the human-readable version in a trailing comment.
func TestEveryActionIsPinnedToASHAWithAVersionComment(t *testing.T) {
	pinned := regexp.MustCompile(`^uses:\s+\S+@[0-9a-f]{40}\s+# v\S+$`)

	for _, name := range []string{releaseWorkflow, ciWorkflow, siteWorkflow} {
		data, err := os.ReadFile(filepath.Join(workflowDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimLeft(strings.TrimSpace(line), "- ")
			if !strings.HasPrefix(trimmed, "uses:") {
				continue
			}
			if !pinned.MatchString(trimmed) {
				t.Errorf("%s:%d: %q is not pinned to a commit SHA with a version comment", name, i+1, trimmed)
			}
		}
	}
}

// The release runs the actions CI already exercises, at the same revisions, so
// a tag build cannot differ from what every pull request proved.
func TestReleaseUsesTheSameActionRevisionsAsCI(t *testing.T) {
	inCI := map[string]string{}
	for _, s := range load(t, ciWorkflow).steps() {
		if action, ref, ok := strings.Cut(s.Uses, "@"); ok {
			inCI[action] = ref
		}
	}
	for _, s := range load(t, releaseWorkflow).steps() {
		action, ref, ok := strings.Cut(s.Uses, "@")
		if !ok {
			continue
		}
		if want, known := inCI[action]; known && ref != want {
			t.Errorf("%s is pinned to %s here and %s in %s", action, ref, want, ciWorkflow)
		}
	}
}

func TestSiteWorkflowBuildsOnPullRequestsThatTouchTheSite(t *testing.T) {
	w := load(t, siteWorkflow)

	node, ok := w.On["pull_request"]
	if !ok {
		t.Fatalf("triggers %v, want pull_request so a docs change is built before merge", keys(w.On))
	}

	var pr struct {
		Paths []string `yaml:"paths"`
	}
	if err := node.Decode(&pr); err != nil {
		t.Fatalf("decode pull_request trigger: %v", err)
	}
	if !contains(pr.Paths, "site/**") {
		t.Errorf("pull_request paths %v, want site/** so a change under site/ is built", pr.Paths)
	}
}

func TestSiteWorkflowInstallsFromTheLockfileAndBuilds(t *testing.T) {
	w := load(t, siteWorkflow)

	var found bool
	for _, s := range w.steps() {
		if !strings.Contains(s.Run, "npm ci") {
			continue
		}
		found = true
		if strings.Contains(s.Run, "npm install") {
			t.Errorf("site build uses npm install, which ignores the lockfile: %q", s.Run)
		}
		if !strings.Contains(s.Run, "npm run build") {
			t.Errorf("site step runs %q, want npm run build so a broken site fails the job", s.Run)
		}
		if s.WorkingDirectory != "site" && !strings.Contains(s.Run, "cd site") {
			t.Errorf("site build working-directory %q, want site", s.WorkingDirectory)
		}
	}
	if !found {
		t.Fatal("no step runs npm ci")
	}
}

func TestSiteWorkflowUsesTheSameCheckoutRevisionAsCI(t *testing.T) {
	ciRef := actionRef(t, ciWorkflow, "actions/checkout")
	siteRef := actionRef(t, siteWorkflow, "actions/checkout")
	if siteRef != ciRef {
		t.Errorf("site checkout is %s, CI checkout is %s; keep them the same", siteRef, ciRef)
	}
}

func TestSiteWorkflowRebuildsWhenTheLandingPageSourcesChange(t *testing.T) {
	w := load(t, siteWorkflow)

	node, ok := w.On["pull_request"]
	if !ok {
		t.Fatalf("triggers %v, want pull_request", keys(w.On))
	}

	var pr struct {
		Paths []string `yaml:"paths"`
	}
	if err := node.Decode(&pr); err != nil {
		t.Fatalf("decode pull_request trigger: %v", err)
	}
	for _, path := range []string{"internal/web/styles/tailwind.css", "docs/assets/**"} {
		if !contains(pr.Paths, path) {
			t.Errorf("pull_request paths %v, want %s so a token or logo change rebuilds the site", pr.Paths, path)
		}
	}
}

func TestSiteWorkflowSetsUpNode22(t *testing.T) {
	w := load(t, siteWorkflow)

	for _, s := range w.steps() {
		if !strings.HasPrefix(s.Uses, "actions/setup-node@") {
			continue
		}
		version := s.With["node-version"]
		if version.Value != "22" {
			t.Errorf("setup-node node-version %q, want 22 (Astro 7 needs Node 22.12 or later)", version.Value)
		}
		return
	}
	t.Fatal("no setup-node step")
}

func actionRef(t *testing.T, workflowName, action string) string {
	t.Helper()
	for _, s := range load(t, workflowName).steps() {
		name, ref, ok := strings.Cut(s.Uses, "@")
		if ok && name == action {
			return ref
		}
	}
	t.Fatalf("%s has no %s step", workflowName, action)
	return ""
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func keys(m map[string]yaml.Node) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
