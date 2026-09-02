package outline

import (
	"fmt"
	"io"
	"slices"
	"sort"
)

// TraverseOptions filters graph traversal.
type TraverseOptions struct {
	// Depth caps hop distance. Zero means unbounded.
	Depth int
	// Relations restricts which edge relations are followed. Nil means
	// RelCalls only.
	Relations []string
	// IncludeInferred follows inferred-confidence edges as well as
	// extracted ones. Extracted edges are always followed.
	IncludeInferred bool
}

// Path is an ordered chain of edges from a seed to a reached node.
type Path []Edge

func (o TraverseOptions) allow(e Edge) bool {
	if !o.IncludeInferred && e.Conf != ConfExtracted {
		return false
	}
	if o.Relations == nil {
		return e.Rel == RelCalls
	}
	return slices.Contains(o.Relations, e.Rel)
}

// Def returns nodes whose Name or Qualified exactly matches name, or whose
// ID is name.
func (g *Graph) Def(name string) []Node {
	g.index()
	if i, ok := g.byID[name]; ok {
		return []Node{g.Nodes[i]}
	}
	var out []Node
	for i := range g.Nodes {
		if g.Nodes[i].Name == name || g.Nodes[i].Qualified == name {
			out = append(out, g.Nodes[i])
		}
	}
	return out
}

// Callers returns inbound calls edges to id.
func (g *Graph) Callers(id string) []Edge {
	return filterRel(g.In(id), RelCalls)
}

// Callees returns outbound calls edges from id.
func (g *Graph) Callees(id string) []Edge {
	return filterRel(g.Out(id), RelCalls)
}

func filterRel(edges []Edge, rel string) []Edge {
	var out []Edge
	for _, e := range edges {
		if e.Rel == rel {
			out = append(out, e)
		}
	}
	return out
}

// Affected walks the graph in reverse from each seed and returns one
// evidence path per reached node. The default relation set is RelCalls,
// so with default options this is the transitive caller set.
func (g *Graph) Affected(seeds []string, opts TraverseOptions) []Path {
	g.index()
	prev := make(map[string]int)
	dist := make(map[string]int)
	var frontier []string
	for _, s := range seeds {
		if _, ok := g.byID[s]; ok {
			prev[s] = -1
			dist[s] = 0
			frontier = append(frontier, s)
		}
	}
	for len(frontier) > 0 {
		id := frontier[0]
		frontier = frontier[1:]
		if opts.Depth > 0 && dist[id] >= opts.Depth {
			continue
		}
		for _, ei := range g.rev[id] {
			e := g.Edges[ei]
			if !opts.allow(e) {
				continue
			}
			if _, seen := prev[e.From]; seen {
				continue
			}
			prev[e.From] = ei
			dist[e.From] = dist[id] + 1
			frontier = append(frontier, e.From)
		}
	}
	reached := make([]string, 0, len(prev))
	for id, ei := range prev {
		if ei >= 0 {
			reached = append(reached, id)
		}
	}
	sort.Slice(reached, func(i, j int) bool {
		if dist[reached[i]] != dist[reached[j]] {
			return dist[reached[i]] < dist[reached[j]]
		}
		return reached[i] < reached[j]
	})
	paths := make([]Path, len(reached))
	for i, id := range reached {
		for cur := id; prev[cur] >= 0; cur = g.Edges[prev[cur]].To {
			paths[i] = append(paths[i], g.Edges[prev[cur]])
		}
	}
	return paths
}

// Path returns the shortest forward edge chain from from to to under opts,
// or nil if none exists.
func (g *Graph) Path(from, to string, opts TraverseOptions) Path {
	g.index()
	if _, ok := g.byID[from]; !ok {
		return nil
	}
	prev := map[string]int{from: -1}
	dist := map[string]int{from: 0}
	frontier := []string{from}
	for len(frontier) > 0 {
		id := frontier[0]
		frontier = frontier[1:]
		if id == to {
			var p Path
			for cur := to; prev[cur] >= 0; cur = g.Edges[prev[cur]].From {
				p = append(Path{g.Edges[prev[cur]]}, p...)
			}
			return p
		}
		if opts.Depth > 0 && dist[id] >= opts.Depth {
			continue
		}
		for _, ei := range g.fwd[id] {
			e := g.Edges[ei]
			if !opts.allow(e) {
				continue
			}
			if _, seen := prev[e.To]; seen {
				continue
			}
			prev[e.To] = ei
			dist[e.To] = dist[id] + 1
			frontier = append(frontier, e.To)
		}
	}
	return nil
}

const bytesPerToken = 3

type textWriter struct {
	w       io.Writer
	limit   int
	written int
	err     error
	trunc   bool
}

func (t *textWriter) line(s string) {
	if t.err != nil || t.trunc {
		return
	}
	if t.limit > 0 && t.written+len(s)+1 > t.limit {
		_, t.err = fmt.Fprintln(t.w, "... truncated at budget")
		t.trunc = true
		return
	}
	_, t.err = fmt.Fprintln(t.w, s)
	t.written += len(s) + 1
}

func (t *textWriter) done() bool { return t.err != nil || t.trunc }

func (g *Graph) display(id string) string {
	if n := g.Node(id); n != nil && n.Qualified != "" {
		return n.Qualified
	}
	return id
}

func nodeLine(n *Node) string {
	loc := n.File
	if n.Line > 0 {
		loc = fmt.Sprintf("%s:%d", n.File, n.Line)
	}
	return fmt.Sprintf("NODE %s %s %s exported=%t sig=%s",
		sanitise(n.Qualified), n.Kind, loc, n.Exported, sanitise(n.Sig))
}

func (g *Graph) edgeLine(e Edge) string {
	return fmt.Sprintf("EDGE %s --%s[%s]--> %s at %s:%d",
		sanitise(g.display(e.From)), e.Rel, e.Conf, sanitise(g.display(e.To)), e.File, e.Line)
}

// Text writes a budgeted line-oriented rendering of the seeds and their
// BFS neighbourhood. Every line is sanitised so untrusted source cannot
// inject terminal escapes or break the line protocol.
func (g *Graph) Text(w io.Writer, ids []string, budget int) error {
	g.index()
	tw := &textWriter{w: w, limit: budget * bytesPerToken}
	seen := make(map[string]bool)
	edgeSeen := make(map[int]bool)

	frontier := make([]string, 0, len(ids))
	for _, id := range ids {
		n := g.Node(id)
		if n == nil || seen[id] {
			continue
		}
		seen[id] = true
		frontier = append(frontier, id)
		tw.line(nodeLine(n))
	}
	for len(frontier) > 0 && !tw.done() {
		id := frontier[0]
		frontier = frontier[1:]
		frontier = g.textEdges(tw, id, seen, edgeSeen, frontier)
	}
	return tw.err
}

func (g *Graph) textEdges(tw *textWriter, id string, seen map[string]bool, edgeSeen map[int]bool, frontier []string) []string {
	for _, adj := range [][]int{g.fwd[id], g.rev[id]} {
		for _, ei := range adj {
			if edgeSeen[ei] || tw.done() {
				continue
			}
			edgeSeen[ei] = true
			e := g.Edges[ei]
			tw.line(g.edgeLine(e))
			for _, next := range []string{e.From, e.To} {
				if seen[next] {
					continue
				}
				seen[next] = true
				if n := g.Node(next); n != nil {
					tw.line(nodeLine(n))
				}
				frontier = append(frontier, next)
			}
		}
	}
	return frontier
}
