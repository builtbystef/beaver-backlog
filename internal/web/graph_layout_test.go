package web

// The graph's layout arithmetic, asserted where it is decided rather than
// through the rendered picture: which layer a node lands in, which row keeps two
// chains from crossing, and that a dependency cycle still terminates. The SVG the
// numbers turn into is a surface concern and is tested at the web seam.

import (
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// node builds one todo issue for a layout fixture.
func fixture(id string, dependsOn ...string) issue.Issue {
	return issue.Issue{ID: id, Title: id, State: issue.StateTodo, DependsOn: dependsOn}
}

// rows and layers read a laid-out graph back by issue id.
func placement(t *testing.T, g graph) (layers, rows map[string]int) {
	t.Helper()
	layers, rows = map[string]int{}, map[string]int{}
	for _, n := range g.Nodes {
		layers[n.Issue.ID] = n.Layer
		rows[n.Issue.ID] = n.Row
	}
	return layers, rows
}

// The spec's worked example: C depends on B depends on A, so A sits in layer 0,
// B in 1, C in 2 — left to right, prerequisite first.
func TestLayersFollowDependencyDepth(t *testing.T) {
	g := layout([]issue.Issue{fixture("c", "b"), fixture("a"), fixture("b", "a")})

	layers, _ := placement(t, g)
	for id, want := range map[string]int{"a": 0, "b": 1, "c": 2} {
		if layers[id] != want {
			t.Errorf("%s in layer %d, want %d", id, layers[id], want)
		}
	}
	if len(g.Nodes) != 3 {
		t.Errorf("laid out %d nodes, want 3", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Errorf("laid out %d edges, want 2", len(g.Edges))
	}
	// A node's x follows its layer, so work reads left to right.
	byID := map[string]node{}
	for _, n := range g.Nodes {
		byID[n.Issue.ID] = n
	}
	if !(byID["a"].X < byID["b"].X && byID["b"].X < byID["c"].X) {
		t.Errorf("x coordinates %v/%v/%v do not run left to right", byID["a"].X, byID["b"].X, byID["c"].X)
	}
}

// A dependency on an issue outside the set is a dangling reference, not a layer:
// it constrains nothing and draws no arrow.
func TestDanglingDependencyIsNotAnEdge(t *testing.T) {
	g := layout([]issue.Issue{fixture("a", "gone")})

	if len(g.Edges) != 0 {
		t.Errorf("laid out %d edges, want none for a dangling reference", len(g.Edges))
	}
	layers, _ := placement(t, g)
	if layers["a"] != 0 {
		t.Errorf("a in layer %d, want 0", layers["a"])
	}
}

// Two independent chains, fed in an order that would cross: the crossing
// reduction pass settles each chain onto a row of its own.
func TestCrossingReductionKeepsIndependentChainsApart(t *testing.T) {
	g := layout([]issue.Issue{
		fixture("d"), fixture("a"),
		fixture("b", "a"), fixture("e", "d"),
		fixture("f", "e"), fixture("c", "b"),
	})

	_, rows := placement(t, g)
	if rows["a"] != rows["b"] || rows["b"] != rows["c"] {
		t.Errorf("chain a→b→c straddles rows %d/%d/%d", rows["a"], rows["b"], rows["c"])
	}
	if rows["d"] != rows["e"] || rows["e"] != rows["f"] {
		t.Errorf("chain d→e→f straddles rows %d/%d/%d", rows["d"], rows["e"], rows["f"])
	}
	if rows["a"] == rows["d"] {
		t.Errorf("both chains sit on row %d; they must not overlap", rows["a"])
	}
}

// A cycle is a store people really have (doctor reports them), so layout must
// finish and say which edge closed the loop rather than hang or crash.
func TestCycleTerminatesWithABackEdge(t *testing.T) {
	g := layout([]issue.Issue{fixture("a", "c"), fixture("b", "a"), fixture("c", "b")})

	if len(g.Nodes) != 3 {
		t.Fatalf("laid out %d nodes, want 3", len(g.Nodes))
	}
	back := 0
	for _, e := range g.Edges {
		if e.Back {
			back++
		}
	}
	if len(g.Edges) != 3 || back != 1 {
		t.Errorf("laid out %d edges with %d back, want 3 with exactly 1", len(g.Edges), back)
	}
}

// A self-dependency is the smallest cycle there is.
func TestSelfDependencyIsABackEdge(t *testing.T) {
	g := layout([]issue.Issue{fixture("a", "a")})

	if len(g.Edges) != 1 || !g.Edges[0].Back {
		t.Errorf("edges = %+v, want one back edge", g.Edges)
	}
}

// Every node sits inside the picture the page is sized to.
func TestCanvasHoldsEveryNode(t *testing.T) {
	g := layout([]issue.Issue{
		{ID: "p", Title: "Parent", State: issue.StateTodo},
		{ID: "k", Title: "Kid", State: issue.StateTodo, Parent: "p"},
		fixture("free"),
	})

	if len(g.Nodes) != 3 {
		t.Fatalf("laid out %d nodes, want 3", len(g.Nodes))
	}
	for _, n := range g.Nodes {
		if n.X < 0 || n.Y < 0 || n.X+nodeWidth > g.Width || n.Y+nodeHeight > g.Height {
			t.Errorf("node %s at (%v,%v) falls outside the %vx%v canvas", n.Issue.ID, n.X, n.Y, g.Width, g.Height)
		}
	}
	for _, c := range g.Clusters {
		if c.X < 0 || c.Y < 0 || c.X+c.W > g.Width || c.Y+c.H > g.Height {
			t.Errorf("cluster %q at (%v,%v) falls outside the %vx%v canvas", c.Label, c.X, c.Y, g.Width, g.Height)
		}
	}
}

// A parent's box holds the parent and its direct children and nothing else, and
// a free-standing issue is laid out clear of it.
func TestClusterBoxHoldsTheFamilyAndNothingElse(t *testing.T) {
	g := layout([]issue.Issue{
		{ID: "p", Title: "The spec", State: issue.StateTodo},
		{ID: "k1", Title: "Slice one", State: issue.StateTodo, Parent: "p"},
		{ID: "k2", Title: "Slice two", State: issue.StateTodo, Parent: "p"},
		fixture("free"),
	})

	if len(g.Clusters) != 1 {
		t.Fatalf("clusters = %+v, want exactly one", g.Clusters)
	}
	box := g.Clusters[0]
	if box.Label != "The spec" {
		t.Errorf("box labelled %q, want the parent's title", box.Label)
	}
	for _, n := range g.Nodes {
		inside := n.X >= box.X && n.X+nodeWidth <= box.X+box.W && n.Y >= box.Y && n.Y+nodeHeight <= box.Y+box.H
		if want := n.Issue.ID != "free"; inside != want {
			t.Errorf("node %s inside the box = %v, want %v", n.Issue.ID, inside, want)
		}
	}
}
