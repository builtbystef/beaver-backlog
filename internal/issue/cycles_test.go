package issue

import (
	"reflect"
	"testing"
)

// dep builds a todo issue with the given id and depends_on edges.
func dep(id string, deps ...string) Issue {
	return Issue{ID: id, Title: id, State: StateTodo, DependsOn: deps}
}

func cyclesOf(issues ...Issue) [][]string {
	return NewRelations(issues).Cycles()
}

func TestCyclesNoneWhenAcyclic(t *testing.T) {
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
	got := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesSelfDependency(t *testing.T) {
	// A self-dependency is a cycle of one, the case Tarjan alone would miss.
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
	// An edge to an absent id is a dangling reference, not part of a cycle.
	got := cyclesOf(dep("aaaaaa", "bbbbbb", "ghost0"), dep("bbbbbb", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCyclesMultipleDisjoint(t *testing.T) {
	// Independent cycles are each reported, ordered by their smallest id.
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
	a := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "cccccc"), dep("cccccc", "aaaaaa"))
	b := cyclesOf(dep("cccccc", "aaaaaa"), dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "cccccc"))
	if !reflect.DeepEqual(a, b) {
		t.Errorf("order-dependent result: %v vs %v", a, b)
	}
}

func TestCyclesTailOffCycleNotIncluded(t *testing.T) {
	// c is blocked forever by the a<->b cycle but is not itself part of it.
	got := cyclesOf(dep("aaaaaa", "bbbbbb"), dep("bbbbbb", "aaaaaa"), dep("cccccc", "aaaaaa"))
	want := [][]string{{"aaaaaa", "bbbbbb"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// --- parent cycles ---

// child builds an issue with a parent edge.
func child(id, parent string) Issue {
	return Issue{ID: id, Title: id, State: StateTodo, Parent: parent}
}

func parentCyclesOf(issues ...Issue) [][]string {
	return NewRelations(issues).ParentCycles()
}

func TestParentCyclesNoneInProperTrees(t *testing.T) {
	// A rooted chain, a branching tree, a dangling parent, and no parents: none loops.
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
	// An issue naming itself as its own parent is a cycle of one.
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
	// Chains draining into the loop are not part of it, and the loop is reported once.
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
	// Independent loops are each reported, ordered by their smallest id,
	// regardless of indexing order.
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
