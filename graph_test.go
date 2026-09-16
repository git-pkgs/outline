package outline

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestIDEscape(t *testing.T) {
	cases := map[string]string{
		"plain":    "plain",
		"a:b":      "a%3Ab",
		"100%":     "100%25",
		"a:b%c":    "a%3Ab%25c",
		"src/x.go": "src/x.go",
	}
	for in, want := range cases {
		if got := idEscape(in); got != want {
			t.Errorf("idEscape(%q) = %q, want %q", in, got, want)
		}
		if got := idUnescape(idEscape(in)); got != in {
			t.Errorf("idUnescape(idEscape(%q)) = %q", in, got)
		}
	}
	if got := modName(ModID("go", "a:b/c")); got != "a:b/c" {
		t.Errorf("modName round-trip = %q", got)
	}
	if a, b := SymID("a:b.go", 10), SymID("a", 10); a == b {
		t.Errorf("distinct paths collided: %q == %q", a, b)
	}
}

func TestGraphAdjacency(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "file:a.go", Kind: KindFile},
			{ID: "sym:a.go:0", Kind: KindFunc, Name: "A"},
			{ID: "sym:b.go:0", Kind: KindFunc, Name: "B"},
		},
		Edges: []Edge{
			{From: "file:a.go", To: "sym:a.go:0", Rel: RelContains, Conf: ConfExtracted},
			{From: "sym:a.go:0", To: "sym:b.go:0", Rel: RelCalls, Conf: ConfInferred},
		},
	}
	if n := g.Node("sym:a.go:0"); n == nil || n.Name != "A" {
		t.Fatalf("Node lookup failed: %v", n)
	}
	out := g.Out("sym:a.go:0")
	if len(out) != 1 || out[0].To != "sym:b.go:0" || out[0].Rel != RelCalls {
		t.Errorf("Out = %v", out)
	}
	in := g.In("sym:a.go:0")
	if len(in) != 1 || in[0].From != "file:a.go" || in[0].Rel != RelContains {
		t.Errorf("In = %v", in)
	}
	if g.Node("missing") != nil {
		t.Error("Node(missing) should be nil")
	}
}

func TestGraphJSONDeterministic(t *testing.T) {
	mk := func() *Graph {
		return &Graph{
			SchemaVersion: SchemaVersion,
			Complete:      true,
			Nodes: []Node{
				{ID: "sym:b.go:0", Kind: KindFunc, Name: "B"},
				{ID: "file:a.go", Kind: KindFile},
			},
			Edges: []Edge{
				{From: "sym:b.go:0", To: "file:a.go", Rel: RelReferences},
				{From: "file:a.go", To: "sym:b.go:0", Rel: RelContains},
			},
		}
	}
	var a, b bytes.Buffer
	if err := mk().JSON(&a); err != nil {
		t.Fatal(err)
	}
	if err := mk().JSON(&b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Errorf("non-deterministic JSON:\n%s\n---\n%s", a.String(), b.String())
	}
	var round Graph
	if err := json.Unmarshal(a.Bytes(), &round); err != nil {
		t.Fatal(err)
	}
	if round.Nodes[0].ID != "file:a.go" {
		t.Errorf("nodes not sorted: first = %q", round.Nodes[0].ID)
	}
	if round.Edges[0].From != "file:a.go" {
		t.Errorf("edges not sorted: first from = %q", round.Edges[0].From)
	}
}
