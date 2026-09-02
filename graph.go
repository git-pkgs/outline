package outline

import (
	"encoding/json"
	"io"
	"sort"
	"strconv"
	"strings"
)

// SchemaVersion is bumped whenever Node, Edge, or Graph change in a way a
// reader must understand.
const SchemaVersion = 1

// Node kinds.
const (
	KindFile     = "file"
	KindModule   = "module"
	KindExternal = "external"
	KindFunc     = "func"
	KindType     = "type"
	KindClass    = "class"
	KindConst    = "const"
	KindVar      = "var"
)

// Edge relations.
const (
	RelContains   = "contains"
	RelImports    = "imports"
	RelCalls      = "calls"
	RelReferences = "references"
	RelInherits   = "inherits"
	RelEmbeds     = "embeds"
	RelReexports  = "reexports"
)

// Edge confidence.
const (
	ConfExtracted = "extracted"
	ConfInferred  = "inferred"
)

// Node is one graph vertex. IDs are opaque and produced only by the id
// encoder helpers below.
type Node struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Qualified string `json:"qualified,omitempty"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Start     int    `json:"start,omitempty"`
	End       int    `json:"end,omitempty"`
	Exported  bool   `json:"exported,omitempty"`
	Sig       string `json:"sig,omitempty"`
}

// Edge is one directed relationship between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Rel  string `json:"rel"`
	Conf string `json:"conf"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
}

// Graph is a whole-repository code graph.
type Graph struct {
	SchemaVersion int      `json:"schema_version"`
	ToolVersion   string   `json:"tool_version"`
	SourceDigest  string   `json:"source_digest"`
	Complete      bool     `json:"complete"`
	Warnings      []string `json:"warnings,omitempty"`
	Nodes         []Node   `json:"nodes"`
	Edges         []Edge   `json:"edges"`

	byID  map[string]int
	fwd   map[string][]int
	rev   map[string][]int
	built bool
}

// idEscape encodes one path or name segment so that ':' remains a safe
// separator in composite IDs. Only ':' and '%' are escaped.
func idEscape(s string) string {
	if strings.IndexByte(s, ':') < 0 && strings.IndexByte(s, '%') < 0 {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ':':
			b.WriteString("%3A")
		case '%':
			b.WriteString("%25")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// FileID returns the node ID for a repository-relative file path.
func FileID(path string) string {
	return "file:" + idEscape(path)
}

// SymID returns the node ID for a declaration at a byte offset in a file.
func SymID(path string, start uint32) string {
	return "sym:" + idEscape(path) + ":" + strconv.FormatUint(uint64(start), 10)
}

// ModID returns the node ID for a module in a language.
func ModID(lang, module string) string {
	return "mod:" + lang + ":" + idEscape(module)
}

// ExtID returns the node ID for an unresolved external target.
func ExtID(lang, module, name string) string {
	return "ext:" + lang + ":" + idEscape(module) + ":" + idEscape(name)
}

// index rebuilds the adjacency maps. It is called lazily on first query.
func (g *Graph) index() {
	if g.built {
		return
	}
	g.byID = make(map[string]int, len(g.Nodes))
	for i := range g.Nodes {
		g.byID[g.Nodes[i].ID] = i
	}
	g.fwd = make(map[string][]int, len(g.Nodes))
	g.rev = make(map[string][]int, len(g.Nodes))
	for i := range g.Edges {
		g.fwd[g.Edges[i].From] = append(g.fwd[g.Edges[i].From], i)
		g.rev[g.Edges[i].To] = append(g.rev[g.Edges[i].To], i)
	}
	g.built = true
}

// Node returns the node with the given ID, or nil.
func (g *Graph) Node(id string) *Node {
	g.index()
	if i, ok := g.byID[id]; ok {
		return &g.Nodes[i]
	}
	return nil
}

// Out returns edges leaving id.
func (g *Graph) Out(id string) []Edge {
	g.index()
	return g.edgeSlice(g.fwd[id])
}

// In returns edges arriving at id.
func (g *Graph) In(id string) []Edge {
	g.index()
	return g.edgeSlice(g.rev[id])
}

func (g *Graph) edgeSlice(idx []int) []Edge {
	if len(idx) == 0 {
		return nil
	}
	out := make([]Edge, len(idx))
	for i, e := range idx {
		out[i] = g.Edges[e]
	}
	return out
}

// sortStable orders Nodes by ID and Edges by (From, To, Rel, File, Line) so
// two builds of the same input produce byte-identical JSON.
func (g *Graph) sortStable() {
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Rel != b.Rel {
			return a.Rel < b.Rel
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	g.built = false
}

// JSON writes the graph as sorted, indented JSON.
func (g *Graph) JSON(w io.Writer) error {
	g.sortStable()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(g)
}
