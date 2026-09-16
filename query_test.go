package outline

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// chain builds a linear call graph a -> b -> c -> d.
func chain() *Graph {
	g := &Graph{SchemaVersion: SchemaVersion}
	names := []string{"a", "b", "c", "d"}
	for i, n := range names {
		g.Nodes = append(g.Nodes, Node{
			ID: "sym:x.go:" + n, Kind: KindFunc, Name: n, Qualified: n,
			File: "x.go", Line: i + 1,
		})
	}
	for i := 0; i+1 < len(names); i++ {
		conf := ConfExtracted
		if i == 1 {
			conf = ConfInferred
		}
		g.Edges = append(g.Edges, Edge{
			From: "sym:x.go:" + names[i], To: "sym:x.go:" + names[i+1],
			Rel: RelCalls, Conf: conf, File: "x.go", Line: i + 1,
		})
	}
	return g
}

func TestDef(t *testing.T) {
	g := chain()
	if got := g.Def("b"); len(got) != 1 || got[0].ID != "sym:x.go:b" {
		t.Errorf("Def(b) = %v", got)
	}
	if got := g.Def("sym:x.go:c"); len(got) != 1 || got[0].Name != "c" {
		t.Errorf("Def by ID = %v", got)
	}
	if got := g.Def("nope"); got != nil {
		t.Errorf("Def(nope) = %v", got)
	}
}

func TestCallersCallees(t *testing.T) {
	g := chain()
	if got := g.Callers("sym:x.go:b"); len(got) != 1 || got[0].From != "sym:x.go:a" {
		t.Errorf("Callers(b) = %v", got)
	}
	if got := g.Callees("sym:x.go:b"); len(got) != 1 || got[0].To != "sym:x.go:c" {
		t.Errorf("Callees(b) = %v", got)
	}
}

func TestAffected(t *testing.T) {
	g := chain()
	paths := g.Affected([]string{"sym:x.go:d"}, TraverseOptions{IncludeInferred: true})
	got := make(map[string]int)
	for _, p := range paths {
		got[p[0].From] = len(p)
	}
	if got["sym:x.go:c"] != 1 || got["sym:x.go:b"] != 2 || got["sym:x.go:a"] != 3 {
		t.Errorf("Affected paths = %v", got)
	}

	paths = g.Affected([]string{"sym:x.go:d"}, TraverseOptions{IncludeInferred: false})
	if len(paths) != 1 || paths[0][0].From != "sym:x.go:c" {
		t.Errorf("Affected without inferred should stop at c: %v", paths)
	}

	paths = g.Affected([]string{"sym:x.go:d"}, TraverseOptions{Depth: 1, IncludeInferred: true})
	if len(paths) != 1 {
		t.Errorf("Affected depth=1 should return 1 path: %v", paths)
	}

	a := chain().Affected([]string{"sym:x.go:d"}, TraverseOptions{IncludeInferred: true})
	b := chain().Affected([]string{"sym:x.go:d"}, TraverseOptions{IncludeInferred: true})
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Affected order not deterministic:\n%v\n%v", a, b)
	}
	if len(a[0]) > len(a[len(a)-1]) {
		t.Errorf("Affected not sorted by distance: %v", a)
	}
}

func TestPath(t *testing.T) {
	g := chain()
	p := g.Path("sym:x.go:a", "sym:x.go:d", TraverseOptions{IncludeInferred: true})
	if len(p) != 3 || p[0].From != "sym:x.go:a" || p[2].To != "sym:x.go:d" {
		t.Errorf("Path a->d = %v", p)
	}
	if p := g.Path("sym:x.go:a", "sym:x.go:d", TraverseOptions{}); p != nil {
		t.Errorf("Path without inferred should not reach d: %v", p)
	}
	if p := g.Path("sym:x.go:d", "sym:x.go:a", TraverseOptions{IncludeInferred: true}); p != nil {
		t.Errorf("Path is directed; d->a should be nil: %v", p)
	}
}

func TestText(t *testing.T) {
	g := chain()
	var buf bytes.Buffer
	if err := g.Text(&buf, []string{"sym:x.go:b"}, 0, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NODE b func x.go:2") {
		t.Errorf("missing seed node line: %q", out)
	}
	if !strings.Contains(out, "EDGE a --calls[extracted]--> b") {
		t.Errorf("missing inbound edge: %q", out)
	}
	if !strings.Contains(out, "EDGE b --calls[inferred]--> c") {
		t.Errorf("missing outbound edge: %q", out)
	}

	buf.Reset()
	if err := g.Text(&buf, []string{"sym:x.go:a"}, 10, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "truncated") {
		t.Errorf("small budget should truncate: %q", buf.String())
	}

	buf.Reset()
	allow := TraverseOptions{}.Allow
	if err := g.Text(&buf, []string{"sym:x.go:b"}, 0, allow); err != nil {
		t.Fatal(err)
	}
	out = buf.String()
	if strings.Contains(out, "[inferred]") {
		t.Errorf("filtered Text should not print inferred edges: %q", out)
	}
	if !strings.Contains(out, "EDGE a --calls[extracted]--> b") {
		t.Errorf("filtered Text should still print extracted edges: %q", out)
	}
}

func TestTextRendering(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "file:x.go", Kind: KindFile, Name: "x.go", File: "x.go"},
			{ID: "mod:go:m", Kind: KindModule, Name: "m"},
		},
		Edges: []Edge{
			{From: "mod:go:m", To: "file:x.go", Rel: RelContains, Conf: ConfInferred},
		},
	}
	var buf bytes.Buffer
	if err := g.Text(&buf, []string{"file:x.go"}, 0, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NODE x.go file x.go") {
		t.Errorf("file node should fall back to Name label: %q", out)
	}
	if !strings.Contains(out, "NODE m module exported=") {
		t.Errorf("module node should omit empty location: %q", out)
	}
	if strings.Contains(out, "at :0") || strings.Contains(out, " at \n") {
		t.Errorf("edge without location should omit at-clause: %q", out)
	}
	if strings.Contains(out, "NODE  ") {
		t.Errorf("blank node label: %q", out)
	}

	evil := &Node{ID: "x", Kind: "func\ninjected", Name: "n", File: "a\x1b[31m.go", Line: 1}
	line := nodeLine(evil)
	if strings.Contains(line, "\n") || strings.Contains(line, "\x1b") {
		t.Errorf("nodeLine leaked control bytes: %q", line)
	}
	e := Edge{From: "a", To: "b", Rel: "calls\n", Conf: "x\x1b", File: "f\n.go", Line: 2}
	if el := g.edgeLine(e); strings.Contains(el, "\n") || strings.Contains(el, "\x1b") {
		t.Errorf("edgeLine leaked control bytes: %q", el)
	}
}

func TestSanitise(t *testing.T) {
	if got := sanitise("a\n b\t\x1b[31mc"); got != "a b [31mc" {
		t.Errorf("sanitise = %q", got)
	}
}
