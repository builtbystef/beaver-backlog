package web

// This file holds the graph view: the whole backlog as one picture. Everything
// here is presentation, meaning where a node sits and which arrow curves where.
// No rule lives here: the edges are the stored depends_on, the boxes are the
// stored parent, and ready, blocked and stuck are the core's own answers,
// carried onto a node as markers rather than recomputed.
//
// The layout is the classic layered one, cut to what a backlog needs: layers by
// dependency depth so work reads left to right, a family in a band of its own so
// a containment box can never straddle another, and a barycentre sweep within
// each band so independent chains stop crossing. A dependency cycle, which a
// hand-edit or a merge can always write, is laid out as the DAG left when the
// edges closing the loops are set aside. Those edges are then drawn back the
// other way, distinctly.

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/issue"
)

// The picture's geometry, in SVG user units. A node is a fixed box because its
// text is truncated rather than wrapped: SVG lays out no paragraphs, and a
// uniform card is what makes the layers legible.
const (
	nodeWidth    = 248
	nodeHeight   = 88
	nodePad      = 14 // the node's own margin around everything it says
	layerGap     = 88 // horizontal space between layers, where the arrows run
	rowGap       = 34
	bandGap      = 52  // vertical space between one family's band and the next
	canvasPad    = 56  // room at the edges, including for an arrow looping above a node
	clusterPad   = 26  // room between a containment box and the nodes inside it
	clusterLabel = 34  // the strip at the top of a box holding its title
	titleLimit   = 28  // characters of an issue's title a node shows
	badgeChar    = 6.2 // the width one character of a label badge takes at its size
	badgePad     = 14  // the badge's own padding around that text
	badgeGap     = 6
	badgeHeight  = 18
	nodeRadius   = 10
	gridPitch    = 32 // the dotted canvas the picture is drawn on, in user units
)

// nodeMetrics is where everything inside a node sits, in the node's own
// coordinates. It travels to the template so the box and its contents cannot
// drift apart: the layout here already owns the geometry, and a template
// repeating the numbers is the same measurement written twice. Every value is
// final, because html/template does no arithmetic and should not have to.
var nodeMetrics = metrics{
	W: nodeWidth, H: nodeHeight, Radius: nodeRadius,
	TitleX:     nodePad,
	TitleY:     nodePad + 16,
	MetaY:      nodePad + 35,
	BadgeY:     nodeHeight - nodePad - badgeHeight,
	BadgeH:     badgeHeight,
	BadgeR:     badgeHeight / 2,
	BadgeTextX: 8,
	BadgeTextY: nodeHeight - nodePad - badgeHeight + 13,
	MarkX:      nodeWidth - nodePad - 5,
	MarkY:      nodePad + 2,
	MarkR:      5,
	Grid:       gridPitch,
}

// metrics is the picture's fixed geometry, the node's inner layout and the pitch
// of the dotted ground, handed to the template as one value.
type metrics struct {
	W, H         float64
	Radius       float64
	TitleX       float64
	TitleY       float64
	MetaY        float64
	BadgeY       float64
	BadgeH       float64
	BadgeR       float64
	BadgeTextX   float64 // the badge text's inset from the badge's own left edge
	BadgeTextY   float64
	MarkX, MarkY float64 // the ready dot
	MarkR        float64
	Grid         float64
}

// graph is a laid-out picture: the boxes, the nodes and the arrows, with the
// canvas they all fit inside.
type graph struct {
	Width, Height float64
	Metrics       metrics
	Clusters      []cluster
	Nodes         []node
	Edges         []edge
}

// cluster is one containment box: a parent's title and the rectangle drawn
// around it and its direct children.
type cluster struct {
	ID             string
	Label          string
	X, Y, W, H     float64
	LabelX, LabelY float64
}

// node is one issue as the picture draws it: where it sits, what it says, and
// the derived conditions a reader reads off it.
type node struct {
	Issue      issue.Issue
	Layer, Row int
	X, Y       float64
	Ready      bool
	Blocked    bool
	Stuck      bool
	Title      string  // the issue's title, truncated to the width of the box
	Badges     []badge // its labels, as many as the box has room for
	Class      string  // the state fill and the condition markers, as CSS classes
}

// badge is one label drawn along the bottom of a node. Its width is estimated
// from the text rather than measured, Go having no font metrics here. A badge a
// little wide costs nothing but a little space.
type badge struct {
	X, W float64
	Text string
}

// badges lays out as many of an issue's labels as fit across one node, in stored
// order, dropping the rest rather than spilling over the box.
func badges(labels []string) []badge {
	var out []badge
	x := float64(nodePad)
	for _, label := range labels {
		w := badgePad + badgeChar*float64(len([]rune(label)))
		if x+w > nodeWidth-nodePad {
			break
		}
		out = append(out, badge{X: x, W: w, Text: label})
		x += w + badgeGap
	}
	return out
}

// edge is one dependency arrow, already curved. Back marks an edge that closes a
// cycle: it is drawn the other way and styled apart, because layering had to set
// it aside to finish at all.
type edge struct {
	From, To string
	Path     string
	Back     bool
	Class    string
}

// graphPage is the graph view's data.
type graphPage struct {
	page
	Filters filterBar
	Graph   graph
}

// graph renders the backlog the address selects as one picture. It reads like
// the board: the same bar, the same query, one listing from the core, laid out
// for the browser and nothing more. Filtering to a parent is therefore a cluster
// on its own, because the core returns that parent's children and an arrow to
// an issue off the page is no arrow at all.
func (s *server) graph(w http.ResponseWriter, r *http.Request) {
	svc, err := s.open()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	f := parseFilters(r.URL.Query())
	listing, refused, err := s.filtered(svc, f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p := s.page("Graph", listing.Warnings)
	p.Live = true
	p.Section = "graph"
	p.Search = f.Search
	s.render(w, r, "graph.html", http.StatusOK, graphPage{
		page:    p,
		Filters: f.bar("/graph", r.URL.Query(), refused),
		Graph:   layout(listing.Issues),
	})
}

// layout turns a listing into a picture. The given order is the core's, so every
// tie below (which node comes first in a layer, which family gets the top band)
// breaks the same way on the same store.
func layout(issues []issue.Issue) graph {
	g := newLayout(issues)
	g.findBackEdges()
	g.assignLayers()
	g.buildBands()
	return g.place()
}

// layoutState is one run of the layout, from the issues in to the picture out.
type layoutState struct {
	issues []issue.Issue
	pos    map[string]int // an issue's place in the given order; every tiebreak
	byID   map[string]issue.Issue
	rel    *issue.Relations
	out    map[string][]string // prerequisite → the issues waiting on it
	in     map[string][]string // the reverse, over forward edges only
	edges  []edge
	back   map[[2]string]bool
	layers map[string]int
	rows   map[string]int
	bands  []band
}

// band is one horizontal strip of the picture: a family with its containment
// box, or the free-standing issues, which have none.
type band struct {
	parent  string // the issue whose box this is; empty for the free-standing band
	members []string
	rows    int
}

func newLayout(issues []issue.Issue) *layoutState {
	l := &layoutState{
		pos:    make(map[string]int, len(issues)),
		byID:   make(map[string]issue.Issue, len(issues)),
		out:    map[string][]string{},
		in:     map[string][]string{},
		back:   map[[2]string]bool{},
		layers: map[string]int{},
		rows:   map[string]int{},
	}
	// An id claimed twice is one issue on the page: the first wins, exactly as
	// the relationship index resolves it.
	for _, iss := range issues {
		if _, seen := l.byID[iss.ID]; seen {
			continue
		}
		l.pos[iss.ID] = len(l.issues)
		l.byID[iss.ID] = iss
		l.issues = append(l.issues, iss)
	}
	l.rel = issue.NewRelations(l.issues)
	for _, iss := range l.issues {
		seen := map[string]bool{}
		for _, dep := range iss.DependsOn {
			// A dependency on an issue that is not here is dangling, not an
			// arrow: there is nothing on the page to point at.
			if _, present := l.byID[dep]; !present || seen[dep] {
				continue
			}
			seen[dep] = true
			l.edges = append(l.edges, edge{From: dep, To: iss.ID})
			l.out[dep] = append(l.out[dep], iss.ID)
		}
	}
	return l
}

// findBackEdges marks the edges that close a cycle: the ones a depth-first walk
// finds pointing back at an issue it is still inside. Setting them aside is what
// leaves a DAG to layer, so a cycle costs a differently drawn arrow instead of a
// layout that never finishes.
func (l *layoutState) findBackEdges() {
	const (
		open = 1
		shut = 2
	)
	seen := make(map[string]int, len(l.issues))
	var walk func(id string)
	walk = func(id string) {
		seen[id] = open
		for _, next := range l.out[id] {
			switch {
			case next == id || seen[next] == open:
				l.back[[2]string{id, next}] = true
			case seen[next] == 0:
				walk(next)
			}
		}
		seen[id] = shut
	}
	for _, iss := range l.issues {
		if seen[iss.ID] == 0 {
			walk(iss.ID)
		}
	}
	for i := range l.edges {
		e := &l.edges[i]
		e.Back = l.back[[2]string{e.From, e.To}]
		e.Class = "edge"
		if e.Back {
			e.Class = "edge edge-back"
		}
		if !e.Back {
			l.in[e.To] = append(l.in[e.To], e.From)
		}
	}
}

// assignLayers puts every issue as far right as its longest chain of
// prerequisites demands: an issue nothing precedes is layer 0, and every other
// sits one layer past the last of its prerequisites.
func (l *layoutState) assignLayers() {
	remaining := make(map[string]int, len(l.issues))
	var ready []string
	for _, iss := range l.issues {
		remaining[iss.ID] = len(l.in[iss.ID])
		if remaining[iss.ID] == 0 {
			ready = append(ready, iss.ID)
		}
	}
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		for _, next := range l.out[id] {
			if l.back[[2]string{id, next}] {
				continue
			}
			if l.layers[id]+1 > l.layers[next] {
				l.layers[next] = l.layers[id] + 1
			}
			remaining[next]--
			if remaining[next] == 0 {
				ready = append(ready, next)
			}
		}
	}
}

// buildBands groups the issues into the strips they are drawn in. An issue
// belongs to its parent's band when that parent is on the page; a parent that is
// nobody's child heads a band of its own; everything left over shares the
// free-standing band at the bottom. Nesting deeper than that is flattened, a
// middle issue sitting in its own parent's box and labelling a second box for
// its children, because a box inside a box buys nothing the layers do not
// already say.
func (l *layoutState) buildBands() {
	children := map[string]bool{}
	for _, iss := range l.issues {
		if _, present := l.byID[iss.Parent]; iss.Parent != "" && present {
			children[iss.Parent] = true
		}
	}
	index := map[string]int{}
	var free band
	for _, iss := range l.issues {
		key := ""
		switch {
		case iss.Parent != "" && l.byID[iss.Parent].ID != "":
			key = iss.Parent
		case children[iss.ID]:
			key = iss.ID
		}
		if key == "" {
			free.members = append(free.members, iss.ID)
			continue
		}
		at, seen := index[key]
		if !seen {
			at = len(l.bands)
			index[key] = at
			l.bands = append(l.bands, band{parent: key})
		}
		l.bands[at].members = append(l.bands[at].members, iss.ID)
	}
	if len(free.members) > 0 {
		l.bands = append(l.bands, free)
	}
	for i := range l.bands {
		l.orderBand(&l.bands[i])
	}
}

// orderBand settles the rows within one band. Each layer starts in the core's
// order and is then swept by barycentre, each node pulled towards the average
// row of what it connects to, which is what stops two independent chains from
// crossing. Only edges inside the band count: a node cannot be pulled by
// something drawn in another strip.
func (l *layoutState) orderBand(b *band) {
	inBand := make(map[string]bool, len(b.members))
	for _, id := range b.members {
		inBand[id] = true
	}
	byLayer := map[int][]string{}
	var order []int
	for _, id := range b.members {
		layer := l.layers[id]
		if _, seen := byLayer[layer]; !seen {
			order = append(order, layer)
		}
		byLayer[layer] = append(byLayer[layer], id)
	}
	sort.Ints(order)
	for _, layer := range order {
		for i, id := range byLayer[layer] {
			l.rows[id] = i
		}
		if n := len(byLayer[layer]); n > b.rows {
			b.rows = n
		}
	}
	// Three sweeps: rightwards pulling each node towards its prerequisites,
	// leftwards towards its dependents, then rightwards again to settle.
	for sweep := range 3 {
		layers := slicesReversedIf(order, sweep%2 == 1)
		for _, layer := range layers {
			neighbours := l.in
			if sweep%2 == 1 {
				neighbours = l.out
			}
			l.sortLayer(byLayer[layer], neighbours, inBand)
		}
	}
}

// sortLayer reorders one layer by the average row of each node's neighbours,
// leaving a node with none where it already was, and breaking every tie by the
// core's order.
func (l *layoutState) sortLayer(ids []string, neighbours map[string][]string, inBand map[string]bool) {
	weight := make(map[string]float64, len(ids))
	for i, id := range ids {
		sum, n := 0.0, 0
		for _, other := range neighbours[id] {
			if inBand[other] {
				sum += float64(l.rows[other])
				n++
			}
		}
		if n == 0 {
			weight[id] = float64(i)
			continue
		}
		weight[id] = sum / float64(n)
	}
	sort.SliceStable(ids, func(i, j int) bool {
		if weight[ids[i]] != weight[ids[j]] {
			return weight[ids[i]] < weight[ids[j]]
		}
		return l.pos[ids[i]] < l.pos[ids[j]]
	})
	for i, id := range ids {
		l.rows[id] = i
	}
}

// slicesReversedIf returns layers in the order a sweep visits them.
func slicesReversedIf(layers []int, reverse bool) []int {
	if !reverse {
		return layers
	}
	out := make([]int, len(layers))
	for i, layer := range layers {
		out[len(layers)-1-i] = layer
	}
	return out
}

// place turns layers and rows into coordinates: the layer fixes x for the whole
// picture, so an arrow between two bands still runs left to right, and each band
// takes the vertical room its rows need, one below the last.
func (l *layoutState) place() graph {
	g := graph{Width: 2 * canvasPad, Height: 2 * canvasPad, Metrics: nodeMetrics}
	originX := float64(canvasPad + clusterPad)
	at := make(map[string]node, len(l.issues))
	y := float64(canvasPad)
	for _, b := range l.bands {
		top := y
		if b.parent != "" {
			y += clusterLabel + clusterPad
		}
		minX, maxX := 0.0, 0.0
		for i, id := range b.members {
			iss := l.byID[id]
			rel := l.rel.For(iss)
			n := node{
				Issue:   iss,
				Layer:   l.layers[id],
				Row:     l.rows[id],
				X:       originX + float64(l.layers[id])*(nodeWidth+layerGap),
				Y:       y + float64(l.rows[id])*(nodeHeight+rowGap),
				Ready:   rel.Ready,
				Blocked: rel.Blocked,
				Stuck:   rel.Stuck,
				Title:   truncate(iss.Title, titleLimit),
				Badges:  badges(iss.Labels),
			}
			n.Class = nodeClass(n)
			at[id] = n
			if i == 0 || n.X < minX {
				minX = n.X
			}
			if i == 0 || n.X > maxX {
				maxX = n.X
			}
		}
		height := float64(b.rows)*(nodeHeight+rowGap) - rowGap
		y += height
		if b.parent != "" {
			y += clusterPad
			g.Clusters = append(g.Clusters, cluster{
				ID:     b.parent,
				Label:  truncate(l.byID[b.parent].Title, 2*titleLimit),
				X:      minX - clusterPad,
				Y:      top,
				W:      maxX + nodeWidth + clusterPad - (minX - clusterPad),
				H:      y - top,
				LabelX: minX - clusterPad + 14,
				LabelY: top + clusterLabel - 12,
			})
		}
		y += bandGap
		g.Height = y - bandGap + canvasPad
	}
	// The nodes come out band by band, which is also the order they are drawn
	// in: a node never covers another, so any order would do.
	for _, b := range l.bands {
		for _, id := range b.members {
			n := at[id]
			g.Nodes = append(g.Nodes, n)
			if right := n.X + nodeWidth + canvasPad + clusterPad; right > g.Width {
				g.Width = right
			}
		}
	}
	for _, e := range l.edges {
		e.Path = arrow(at[e.From], at[e.To])
		g.Edges = append(g.Edges, e)
	}
	return g
}

// nodeClass is the encoding a reader reads off a node: the state's fill, and
// each derived condition as a marker. Ready is only ever on unblocked todo work,
// because that is what the core means by it.
func nodeClass(n node) string {
	classes := []string{"node", "state-" + string(n.Issue.State)}
	if n.Blocked {
		classes = append(classes, "blocked")
	}
	if n.Stuck {
		classes = append(classes, "stuck")
	}
	if n.Ready {
		classes = append(classes, "ready")
	}
	return strings.Join(classes, " ")
}

// arrow curves one dependency from its prerequisite to the issue waiting on it:
// out of the right side into the left where the layers run that way, and back
// around the outside where they do not, which means a cycle's closing edge or an
// edge within one layer.
func arrow(from, to node) string {
	const loop = 40
	if from.Issue.ID == to.Issue.ID {
		// A self-dependency has nowhere to go but over the top of its own node.
		x, y := from.X, from.Y
		return fmt.Sprintf("M %g %g C %g %g, %g %g, %g %g",
			x+0.3*nodeWidth, y, x+0.05*nodeWidth, y-loop, x+0.95*nodeWidth, y-loop, x+0.7*nodeWidth, y)
	}
	sy, ty := from.Y+nodeHeight/2, to.Y+nodeHeight/2
	if to.X >= from.X+nodeWidth {
		sx, tx := from.X+nodeWidth, to.X
		c := (tx - sx) / 2
		return fmt.Sprintf("M %g %g C %g %g, %g %g, %g %g", sx, sy, sx+c, sy, tx-c, ty, tx, ty)
	}
	sx, tx := from.X, to.X+nodeWidth
	c := max((sx-tx)/2, loop)
	return fmt.Sprintf("M %g %g C %g %g, %g %g, %g %g", sx, sy, sx-c, sy, tx+c, ty, tx, ty)
}

// truncate shortens a title to what fits a node, counting characters rather than
// bytes so a multi-byte title is not cut mid-rune.
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return strings.TrimRight(string(runes[:limit-1]), " ") + "…"
}
