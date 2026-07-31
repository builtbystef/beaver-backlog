package web_test

// The graph as a reader meets it: the whole backlog as one picture. What is
// asserted here is the surface — a node per issue, an arrow per edge, the
// containment box, and the markers that make execution status readable — never
// what blocked or ready mean, which is the core's to say.

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// backlog is the fixture the graph tests read: a cluster, a free-standing
// issue, a blocked chain, and an issue stuck behind a cancelled dependency.
type backlog struct {
	parent, child, started, free, base, blocked, cancelled, stuck issue.Issue
}

func seedBacklog(t *testing.T, dir string) backlog {
	t.Helper()
	svc := open(t, dir)
	var b backlog
	b.parent = create(t, svc, core.Draft{Title: "The spec"})
	b.child = create(t, svc, core.Draft{Title: "Slice one", Parent: b.parent.ID, Labels: []string{"ui"}})
	b.started = create(t, svc, core.Draft{Title: "Slice two", Parent: b.parent.ID})
	start(t, svc, b.started.ID)
	b.free = create(t, svc, core.Draft{Title: "Standalone chore"})
	b.base = create(t, svc, core.Draft{Title: "Groundwork"})
	b.blocked = create(t, svc, core.Draft{Title: "Waits on groundwork", DependsOn: []string{b.base.ID}})
	b.cancelled = create(t, svc, core.Draft{Title: "Never mind"})
	transition(t, svc, b.cancelled.ID, issue.StateCancelled)
	b.stuck = create(t, svc, core.Draft{Title: "Waits forever", DependsOn: []string{b.cancelled.ID}})
	return b
}

func (b backlog) all() []issue.Issue {
	return []issue.Issue{b.parent, b.child, b.started, b.free, b.base, b.blocked, b.cancelled, b.stuck}
}

func TestGraphDrawsANodePerIssueAndAnArrowPerEdge(t *testing.T) {
	dir := newStore(t)
	b := seedBacklog(t, dir)

	res := get(newHandler(t, dir), "/graph")

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	nodes := graphNodes(t, body)
	if len(nodes) != len(b.all()) {
		t.Errorf("graph holds %d nodes, want %d", len(nodes), len(b.all()))
	}
	for _, iss := range b.all() {
		markup, ok := nodes[iss.ID]
		if !ok {
			t.Errorf("issue %s (%q) has no node", iss.ID, iss.Title)
			continue
		}
		if !strings.Contains(markup, iss.Title) {
			t.Errorf("node for %s does not name it: %s", iss.ID, markup)
		}
		// Every node is the way into its issue.
		if link := `href="/issues/` + iss.ID + `"`; !strings.Contains(markup, link) {
			t.Errorf("node for %s is not a link to its detail page (%s)", iss.ID, link)
		}
	}
	// One arrow per stored dependency, pointing prerequisite → dependent.
	edges := graphEdges(body)
	want := map[string]bool{
		b.base.ID + "→" + b.blocked.ID:    true,
		b.cancelled.ID + "→" + b.stuck.ID: true,
	}
	if len(edges) != len(want) {
		t.Errorf("graph holds %d arrows %v, want %d", len(edges), edges, len(want))
	}
	for edge := range want {
		if !edges[edge] {
			t.Errorf("no arrow for %s; drawn: %v", edge, edges)
		}
	}
}

func TestGraphEncodesEveryDerivedCondition(t *testing.T) {
	dir := newStore(t)
	b := seedBacklog(t, dir)

	nodes := graphNodes(t, get(newHandler(t, dir), "/graph").Body.String())

	for _, c := range []struct {
		what  string
		id    string
		want  []string
		avoid []string
	}{
		{"a todo issue nobody waits on", b.base.ID, []string{"state-todo", "ready"}, []string{"blocked", "stuck"}},
		{"an issue waiting on unfinished work", b.blocked.ID, []string{"state-todo", "blocked"}, []string{"stuck", "ready"}},
		{"an issue waiting on a cancelled dependency", b.stuck.ID, []string{"blocked", "stuck"}, nil},
		{"a claimed issue", b.started.ID, []string{"state-in-progress", "tester"}, []string{"ready"}},
		{"a cancelled issue", b.cancelled.ID, []string{"state-cancelled"}, []string{"ready"}},
		{"a labelled issue", b.child.ID, []string{"ui"}, nil},
	} {
		markup := nodes[c.id]
		for _, want := range c.want {
			if !strings.Contains(markup, want) {
				t.Errorf("node for %s (%s) missing %q: %s", c.what, c.id, want, markup)
			}
		}
		for _, avoid := range c.avoid {
			if strings.Contains(markup, avoid) {
				t.Errorf("node for %s (%s) should not carry %q: %s", c.what, c.id, avoid, markup)
			}
		}
	}
}

func TestGraphBoxesAParentWithItsChildren(t *testing.T) {
	dir := newStore(t)
	b := seedBacklog(t, dir)

	body := get(newHandler(t, dir), "/graph").Body.String()

	box := regexp.MustCompile(`data-cluster="([a-z0-9]+)"`).FindAllStringSubmatch(body, -1)
	if len(box) != 1 || box[0][1] != b.parent.ID {
		t.Fatalf("containment boxes = %v, want one for the parent %s", box, b.parent.ID)
	}
	if !strings.Contains(body, "The spec") {
		t.Error("the containment box is unlabelled")
	}
}

// The picture is sized in the markup, so a backlog wider than the window makes
// the page scroll rather than shrinking the nodes to nothing.
func TestGraphCanvasIsSized(t *testing.T) {
	dir := newStore(t)
	seedBacklog(t, dir)

	body := get(newHandler(t, dir), "/graph").Body.String()

	svg := regexp.MustCompile(`<svg[^>]*>`).FindString(body)
	for _, attr := range []string{"width=", "height=", "viewBox="} {
		if !strings.Contains(svg, attr) {
			t.Errorf("svg tag %q has no %s", svg, attr)
		}
	}
}

// A cycle is a store doctor reports, not one the graph may choke on.
func TestGraphRendersACycleWithADistinctBackEdge(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	first := create(t, svc, core.Draft{Title: "First"})
	second := create(t, svc, core.Draft{Title: "Second", DependsOn: []string{first.ID}})
	// The store refuses a cycle through the core, so close it by hand — exactly
	// the way a merge or a hand-edit does.
	addFrontmatterEdge(t, dir, first.ID, second.ID)

	res := get(newHandler(t, dir), "/graph")

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 despite the cycle", res.Code)
	}
	body := res.Body.String()
	if nodes := graphNodes(t, body); len(nodes) != 2 {
		t.Errorf("graph holds %d nodes, want both sides of the cycle", len(nodes))
	}
	edges := graphEdges(body)
	if len(edges) != 2 {
		t.Fatalf("graph holds %d arrows %v, want 2", len(edges), edges)
	}
	if !strings.Contains(body, "edge-back") {
		t.Errorf("no arrow is marked as the edge closing the cycle:\n%s", body)
	}
}

func TestGraphIsReachableFromEveryPage(t *testing.T) {
	dir := newStore(t)
	h := newHandler(t, dir)
	for _, page := range []string{"/", "/issues", "/graph", "/nope"} {
		if body := get(h, page).Body.String(); !strings.Contains(body, `href="/graph"`) {
			t.Errorf("%s has no navigation to the graph", page)
		}
	}
}

// The graph is a view like any other: a broken file costs a banner, not a page
// (ADR 0003), and the store's own change feed may redraw it.
func TestGraphSurvivesAnInvalidFile(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Perfectly fine"})
	writeFile(t, dir, "broken.md", "not an issue file")

	res := get(newHandler(t, dir), "/graph")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 despite the invalid file", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "broken.md") {
		t.Errorf("banner does not name the skipped file:\n%s", body)
	}
	if len(graphNodes(t, body)) != 1 {
		t.Error("the valid issue is missing; one broken file must not empty the graph")
	}
}

// addFrontmatterEdge writes a depends_on edge straight into an issue's
// frontmatter — the way a merge or a hand-edit closes a loop the core refuses.
func addFrontmatterEdge(t *testing.T, dir, id, dep string) {
	t.Helper()
	path := issueFile(t, dir, id)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	edited := strings.Replace(string(raw), "\nstate:", "\ndepends_on:\n    - "+dep+"\nstate:", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

var (
	nodeMark = regexp.MustCompile(`data-issue="([a-z0-9]+)"`)
	edgeMark = regexp.MustCompile(`data-edge="([a-z0-9]+→[a-z0-9]+)"`)
)

// graphNodes reads the rendered picture back as the nodes it draws: each issue's
// id against the markup of its own node.
func graphNodes(t *testing.T, body string) map[string]string {
	t.Helper()
	marks := nodeMark.FindAllStringSubmatchIndex(body, -1)
	nodes := make(map[string]string, len(marks))
	for _, m := range marks {
		// A node is the whole element the mark sits in, opening tag and all.
		open := strings.LastIndex(body[:m[0]], "<a ")
		if open < 0 {
			t.Fatalf("node mark at %d is not inside an element:\n%s", m[0], body)
		}
		rest := body[open:]
		end := strings.Index(rest, "</a>")
		if end < 0 {
			t.Fatalf("node markup at %d is never closed:\n%s", m[0], rest)
		}
		nodes[body[m[2]:m[3]]] = rest[:end]
	}
	return nodes
}

func graphEdges(body string) map[string]bool {
	edges := map[string]bool{}
	for _, m := range edgeMark.FindAllStringSubmatch(body, -1) {
		edges[m[1]] = true
	}
	return edges
}
