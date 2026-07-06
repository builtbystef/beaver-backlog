package issue

import "sort"

// Cycles returns the dependency cycles in the indexed set: each is the sorted
// ids of a group of issues mutually reachable through depends_on edges, or a
// single issue that depends on itself. Only edges among issues in the set
// count — an edge to an absent id is a dangling reference, not part of any
// cycle — and cycles are ordered by their smallest id, so the result is
// deterministic regardless of indexing order.
func (r *Relations) Cycles() [][]string {
	// Visit nodes and neighbours in sorted order so the output does not depend
	// on map iteration order. Self-edges are tracked apart from the adjacency
	// because Tarjan alone treats a lone self-dependent node as a trivial SCC.
	ids := make([]string, 0, len(r.byID))
	for id := range r.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	selfLoop := make(map[string]bool)
	adj := make(map[string][]string, len(r.byID))
	for _, id := range ids {
		seen := make(map[string]bool)
		var out []string
		for _, dep := range r.byID[id].DependsOn {
			switch {
			case dep == id:
				selfLoop[id] = true
			case seen[dep]:
				// already recorded this edge
			default:
				if _, present := r.byID[dep]; present {
					seen[dep] = true
					out = append(out, dep)
				}
			}
		}
		sort.Strings(out)
		adj[id] = out
	}

	// Tarjan's strongly-connected-components. A component of more than one node
	// is a cycle; a lone node is a cycle only when it depends on itself.
	var (
		index   = make(map[string]int, len(ids))
		lowlink = make(map[string]int, len(ids))
		onStack = make(map[string]bool, len(ids))
		stack   []string
		counter int
		cycles  [][]string
	)
	var connect func(v string)
	connect = func(v string) {
		index[v], lowlink[v] = counter, counter
		counter++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, visited := index[w]; !visited {
				connect(w)
				lowlink[v] = min(lowlink[v], lowlink[w])
			} else if onStack[w] {
				lowlink[v] = min(lowlink[v], index[w])
			}
		}
		if lowlink[v] != index[v] {
			return // not the root of an SCC
		}
		var comp []string
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			comp = append(comp, w)
			if w == v {
				break
			}
		}
		if len(comp) > 1 || selfLoop[comp[0]] {
			sort.Strings(comp)
			cycles = append(cycles, comp)
		}
	}
	for _, v := range ids {
		if _, visited := index[v]; !visited {
			connect(v)
		}
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i][0] < cycles[j][0] })
	return cycles
}

// ParentCycles returns the parent-edge cycles in the indexed set: each is the
// sorted ids of a group of issues whose parent chain loops back on itself,
// including an issue that names itself as its own parent. Like Cycles, only
// edges to issues in the set count, and cycles are ordered by their smallest id,
// so the result is deterministic.
//
// Each issue has at most one parent, so every walk up a parent chain either
// dead-ends or closes exactly one loop; detection is a plain chain-walk rather
// than a full SCC pass.
func (r *Relations) ParentCycles() [][]string {
	ids := make([]string, 0, len(r.byID))
	for id := range r.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	const (
		unvisited = iota
		walking   // on the chain currently being followed
		settled   // fully processed: cycle-free, or its cycle already recorded
	)
	state := make(map[string]int, len(ids))
	var cycles [][]string
	for _, start := range ids {
		if state[start] != unvisited {
			continue
		}
		// Follow the parent chain from start. Meeting the current chain again
		// closes a new cycle; meeting settled ground means the chain drains
		// into territory already accounted for.
		var path []string
		for cur := start; ; {
			if state[cur] == walking {
				for i, p := range path {
					if p == cur {
						cycle := append([]string(nil), path[i:]...)
						sort.Strings(cycle)
						cycles = append(cycles, cycle)
						break
					}
				}
				break
			}
			if state[cur] == settled {
				break
			}
			state[cur] = walking
			path = append(path, cur)
			parent := r.byID[cur].Parent
			if _, present := r.byID[parent]; !present {
				break
			}
			cur = parent
		}
		for _, p := range path {
			state[p] = settled
		}
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i][0] < cycles[j][0] })
	return cycles
}
