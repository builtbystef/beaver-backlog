package issue

import "sort"

// Cycles returns the dependency cycles in the indexed set. Each cycle is the
// sorted ids of a group of issues that are mutually reachable through depends_on
// edges — a group that can never all reach done, because every member is waiting
// on another (ADR 0011). depends_on is a directed edge ("a waits on b"), so a
// cycle is a strongly connected component of more than one issue, or a single
// issue that depends on itself (a cycle of one).
//
// Only edges among issues in the set count: an edge to an id that is not present
// is a dangling reference — doctor's separate concern (n9b4a7) — not part of any
// cycle. The member ids within each cycle are sorted, and the cycles are ordered
// by their smallest id, so the result is deterministic regardless of the order the
// issues were indexed in. Busy Beaver never breaks a cycle itself: doctor reports it
// for a human to re-scope or drop an edge.
func (r *Relations) Cycles() [][]string {
	// Visit nodes, and each node's neighbours, in sorted order, so Tarjan's walk —
	// and therefore the output — does not depend on map iteration order. Self-edges
	// are tracked apart from the adjacency so a lone node that depends on itself
	// still reads as a one-issue cycle (Tarjan alone treats it as a trivial SCC).
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

	// Tarjan's strongly-connected-components. A component of more than one node is a
	// cycle; a lone node is a cycle only when it depends on itself.
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
