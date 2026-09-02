package outline

import (
	"bytes"
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
	if err := g.Text(&buf, []string{"sym:x.go:b"}, 0); err != nil {
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
	if err := g.Text(&buf, []string{"sym:x.go:a"}, 10); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "truncated") {
		t.Errorf("small budget should truncate: %q", buf.String())
	}
}

func TestSanitise(t *testing.T) {
	if got := sanitise("a\n b\t\x1b[31mc"); got != "a b [31mc" {
		t.Errorf("sanitise = %q", got)
	}
}
