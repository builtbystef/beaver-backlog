package issue

import (
	"reflect"
	"testing"
)

// dep builds a todo issue with the given id and depends_on edges — enough to shape a
// dependency graph for the cycle tests.
func dep(id string, deps ...string) Issue {
	return Issue{ID: id, Title: id, State: StateTodo, DependsOn: deps}
}

func cyclesOf(issues ...Issue) [][]string {
	return NewRelations(issues).Cycles()
}

func TestCyclesNoneWhenAcyclic(t *testing.T) {
	// A plain chain a -> b -> c and a diamond both have no cycle: every path
	// eventually dead-ends, so nothing is mutually reachable.
	cases := map[string][]Issue{
		"empty":  nil,
		"single": {dep("aaaaaa")},
		"chain":  {dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "cccccc"), dep("cccccc")},
		"diamond": {
			dep("aaaaaa", "bbbbbb", "cccccc"),
			dep("bbbbbb", "dddddd"),
			dep("cccccc", "dddddd"),
			dep("dddddd"),
		},
	}
	for name, issues := range cases {
		if got := cyclesOf(issues...); len(got) != 0 {
			t.Errorf("%s: got cycles %v, want none", name, got)
		}
	}
}

func TestCyclesTwoNodeCycle(t *testing.T) {
	// a waits on b and b waits on a: neither can ever be done, so the pair is a cycle,
	// reported with its members sorted.
	got := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesSelfDependency(t *testing.T) {
	// An issue that depends on itself is a cycle of one — the degenerate case Tarjan
	// alone would miss (a lone node is a trivial SCC), so Cycles tracks the self-edge.
	got := cyclesOf(dep("aaaaaa", "aaaaaa"))
	want := [][]string{{"aaaaaa"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesThreeNodeCycle(t *testing.T) {
	got := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "cccccc"), dep("cccccc", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb", "cccccc"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesIgnoreDanglingEdges(t *testing.T) {
	// An edge to an id no issue holds is a dangling reference, not part of a cycle:
	// the a<->b cycle is reported, and a's edge to the absent "ghost" is ignored.
	got := cyclesOf(dep("aaaaaa", "bbbbbb", "ghost0"), dep("bbbbbb", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesMultipleDisjoint(t *testing.T) {
	// Two independent cycles are each reported, ordered by their smallest id, so the
	// result is deterministic regardless of input order.
	got := cyclesOf(
		dep("xxxxxx", "yyyyyy"),
		dep("yyyyyy", "xxxxxx"),
		dep("aaaaaa", "bbbbbb"),
		dep("bbbbbb", "aaaaaa"),
	)
	want := [][]string{{"aaaaaa", "bbbbbb"}, {"xxxxxx", "yyyyyy"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesDeterministicAcrossInputOrder(t *testing.T) {
	// The same graph indexed in a different order yields the identical result — the
	// property doctor relies on for byte-stable output.
	a := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "cccccc"), dep("cccccc", "aaaaaa"))
	b := cyclesOf(dep("cccccc", "aaaaaa"), dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "cccccc"))
	if !reflect.DeepEqual(a, b) {
		t.Errorf("order-dependent result: %v vs %v", a, b)
	}
}

func TestCyclesTailOffCycleNotIncluded(t *testing.T) {
	// c depends on the a<->b cycle but nothing depends back on c, so c is blocked
	// forever yet is not itself part of the cycle: only a and b are reported.
	got := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "aaaaaa"), dep("cccccc", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// --- parent cycles ---

// child builds an issue with a parent edge — enough to shape a hierarchy for the
// parent-cycle tests.
func child(id, parent string) Issue {
	return Issue{ID: id, Title: id, State: StateTodo, Parent: parent}
}

func parentCyclesOf(issues ...Issue) [][]string {
	return NewRelations(issues).ParentCycles()
}

func TestParentCyclesNoneInProperTrees(t *testing.T) {
	// A rooted chain, a branching tree, a dangling parent (doctor's separate
	// dangling-reference concern), and no parents at all: none loops.
	cases := map[string][]Issue{
		"empty":    nil,
		"rootless": {child("aaaaaa", "")},
		"chain":    {child("aaaaaa", "bbbbbb"), child("bbbbbb", "cccccc"), child("cccccc", "")},
		"tree":     {child("aaaaaa", "cccccc"), child("bbbbbb", "cccccc"), child("cccccc", "")},
		"dangling": {child("aaaaaa", "ghost0")},
	}
	for name, issues := range cases {
		if got := parentCyclesOf(issues...); len(got) != 0 {
			t.Errorf("%s: got parent cycles %v, want none", name, got)
		}
	}
}

func TestParentCyclesSelfParent(t *testing.T) {
	// An issue naming itself as its own parent is a cycle of one — the degenerate
	// hierarchy show would otherwise render as an issue among its own children.
	got := parentCyclesOf(child("aaaaaa", "aaaaaa"))
	want := [][]string{{"aaaaaa"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParentCyclesMutualPair(t *testing.T) {
	got := parentCyclesOf(child("aaaaaa", "bbbbbb"), child("bbbbbb", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParentCyclesTailIntoLoopNotIncluded(t *testing.T) {
	// c hangs off the a<->b loop: its chain drains into the cycle, but c itself is
	// not part of it, and the cycle is reported exactly once however many chains
	// lead in.
	got := parentCyclesOf(
		child("aaaaaa", "bbbbbb"),
		child("bbbbbb", "aaaaaa"),
		child("cccccc", "aaaaaa"),
		child("dddddd", "cccccc"),
	)
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParentCyclesMultipleDisjointDeterministic(t *testing.T) {
	// Two independent loops are each reported, ordered by their smallest id, and the
	// result is identical however the issues were indexed — the property doctor
	// relies on for byte-stable output.
	a := parentCyclesOf(
		child("xxxxxx", "yyyyyy"), child("yyyyyy", "xxxxxx"),
		child("aaaaaa", "bbbbbb"), child("bbbbbb", "aaaaaa"),
	)
	b := parentCyclesOf(
		child("bbbbbb", "aaaaaa"), child("aaaaaa", "bbbbbb"),
		child("yyyyyy", "xxxxxx"), child("xxxxxx", "yyyyyy"),
	)
	want := [][]string{{"aaaaaa", "bbbbbb"}, {"xxxxxx", "yyyyyy"}}
	if !reflect.DeepEqual(a, want) || !reflect.DeepEqual(b, want) {
		t.Errorf("got %v and %v, want both %v", a, b, want)
	}
}
