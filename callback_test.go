package outline

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildCallbackOwnership(t *testing.T) {
	cases := []struct{ language, filename, source string }{
		{"go", "app.go", "package app\nfunc sink(){}\nfunc side(){}\nfunc register(callback func()){}\nfunc entry(){register(func(){sink(); register(func(){sink()})}); side()}\n"},
		{"python", "app.py", "def sink():\n    pass\ndef side():\n    pass\ndef register(callback):\n    pass\ndef entry():\n    register(lambda: (sink(), register(lambda: sink())))\n    side()\n"},
		{"ruby", "app.rb", "def sink\nend\ndef side\nend\ndef register\nend\ndef entry\n  register { sink(); register do\n    sink()\n  end }\n  side()\nend\n"},
	}
	for _, c := range cases {
		t.Run(c.language, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{c.filename: c.source})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			sink := nodeByName(g, KindFunc, "sink")
			side := nodeByName(g, KindFunc, "side")
			if path := g.Path(entry.ID, sink.ID, TraverseOptions{IncludeInferred: true}); len(path) != 0 {
				t.Fatalf("registration produced an execution path: %v", path)
			}
			if path := g.Path(entry.ID, side.ID, TraverseOptions{IncludeInferred: true}); len(path) != 1 {
				t.Fatalf("ordinary call lost: %v", path)
			}
			owners := assertCallbackOwners(t, g, entry.ID, sink.ID)
			assertCallbackContainment(t, g, entry.ID, sink.ID, owners)
			assertCallbackRepeat(t, root, g)
		})
	}
}

func assertCallbackOwners(t *testing.T, g *Graph, entry, sink string) map[string]bool {
	t.Helper()
	owners := make(map[string]bool)
	for _, edge := range g.Callers(sink) {
		owner := g.Node(edge.From)
		if owner == nil || owner.Kind != KindFunc || owner.Exported || owner.ID == entry || owner.Sig == "" {
			t.Fatalf("callback call has wrong owner: %v", edge)
		}
		if !strings.HasPrefix(owner.Name, "<") || strings.Contains(owner.Sig, "sink()") {
			t.Fatalf("callback lacks a compact identity/signature: %+v", owner)
		}
		if edge.Call == nil || edge.Call.Start < uint32(owner.Start) || edge.Call.End > uint32(owner.End) {
			t.Fatalf("call span outside callback: %v", edge)
		}
		owners[owner.ID] = true
	}
	if len(owners) != 2 {
		t.Fatalf("want two separate callback owners, got %v", owners)
	}
	return owners
}

func assertCallbackContainment(t *testing.T, g *Graph, entry, sink string, owners map[string]bool) {
	t.Helper()
	contains := 0
	for _, edge := range g.Edges {
		if edge.Rel == RelContains && owners[edge.To] && (edge.From == entry || owners[edge.From]) {
			contains++
		}
	}
	if contains != 2 {
		t.Fatalf("callback containment lost: %v", g.Edges)
	}
	for _, path := range g.Affected([]string{sink}, TraverseOptions{IncludeInferred: true}) {
		if len(path) > 0 && path[0].From == entry {
			t.Fatalf("affected query attributed callback execution to entry: %v", path)
		}
	}
}

func assertCallbackRepeat(t *testing.T, root string, g *Graph) {
	t.Helper()
	repeated, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var first, second bytes.Buffer
	if err := g.JSON(&first); err != nil {
		t.Fatal(err)
	}
	if err := repeated.JSON(&second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("callback graph changed on repeated build")
	}
}

func TestBuildDirectAnonymousInvocation(t *testing.T) {
	cases := []struct{ filename, source string }{
		{"app.go", "package app\nfunc sink(){}\nfunc entry(){ (func(){sink()})() }\n"},
		{"app.py", "def sink():\n    pass\ndef entry():\n    (lambda: sink())()\n"},
	}
	for _, c := range cases {
		t.Run(c.filename, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{c.filename: c.source})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			sink := nodeByName(g, KindFunc, "sink")
			path := g.Path(entry.ID, sink.ID, TraverseOptions{})
			if len(path) != 2 || path[0].To != path[1].From || path[0].To == entry.ID {
				t.Fatalf("direct invocation should pass through the anonymous callable: %v", path)
			}
		})
	}
}

func TestBuildAnonymousParameterScope(t *testing.T) {
	cases := []struct{ language, filename, source string }{
		{"go", "app.go", "package app\nfunc helper(){}\nfunc entry(){ register(func(helper func()){helper()}); helper() }\n"},
		{"python", "app.py", "def helper():\n    pass\ndef entry():\n    register(lambda helper: helper())\n    helper()\n"},
	}
	for _, c := range cases {
		t.Run(c.language, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{c.filename: c.source})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			helper := nodeByName(g, KindFunc, "helper")
			resolved := g.Callers(helper.ID)
			if len(resolved) != 1 || resolved[0].From != entry.ID {
				t.Fatalf("anonymous parameter leaked or failed to shadow: %v", resolved)
			}
			unresolved := g.Callers(ExtID(c.language, "", "helper"))
			if len(unresolved) != 1 || unresolved[0].From == entry.ID {
				t.Fatalf("parameter call should belong to the callback: %v", unresolved)
			}
		})
	}
}

func TestBuildLambdaDefaultOwnership(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"app.py": "def default():\n    pass\ndef body():\n    pass\ndef entry():\n    register(lambda helper=default(): body())\n"})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	defaults := g.Callers(nodeByName(g, KindFunc, "default").ID)
	body := g.Callers(nodeByName(g, KindFunc, "body").ID)
	if len(defaults) != 1 || defaults[0].From != entry.ID || len(body) != 1 || body[0].From == entry.ID {
		t.Fatalf("lambda creation and body execution were merged: defaults=%v body=%v", defaults, body)
	}
}

func TestBuildRubyBlockSingletonContext(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"app.rb": "module Worker\n  def self.sink\n  end\nend\ndef Worker.entry\n  register { self.sink(); sink() }\nend\n"})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := g.Def("Worker.entry")
	sink := g.Def("Worker.sink")
	if len(entry) != 1 || len(sink) != 1 {
		t.Fatalf("missing methods: entry=%v sink=%v", entry, sink)
	}
	for _, edge := range g.Callers(sink[0].ID) {
		if edge.From == entry[0].ID || !strings.HasPrefix(g.Node(edge.From).Name, "<block@") {
			t.Fatalf("block lost its owner or singleton context: %v", edge)
		}
	}
	if len(g.Callers(sink[0].ID)) != 2 {
		t.Fatalf("singleton targets lost inside block: %v", g.Edges)
	}
}

func TestBuildRubyStabbyLambdaOwnership(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"app.rb": "def sink\nend\ndef entry\n  register(->(item) { sink() })\nend\n"})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	edges := g.Callers(nodeByName(g, KindFunc, "sink").ID)
	if len(edges) != 1 || !strings.HasPrefix(g.Node(edges[0].From).Name, "<lambda@") {
		t.Fatalf("Ruby lambda lost anonymous ownership: %v", edges)
	}
	count := 0
	for _, node := range g.Nodes {
		if strings.HasPrefix(node.Name, "<lambda@") || strings.HasPrefix(node.Name, "<block@") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("lambda wrapper produced duplicate callable nodes: %v", g.Nodes)
	}
}

func TestBuildRubyBlockDefaultOwnership(t *testing.T) {
	cases := []string{"register { |item=default()| body() }", "register(->(item=default()) { body() })"}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{"app.rb": "def default\nend\ndef body\nend\ndef entry\n  " + source + "\nend\n"})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			defaults := g.Callers(nodeByName(g, KindFunc, "default").ID)
			body := g.Callers(nodeByName(g, KindFunc, "body").ID)
			if len(defaults) != 1 || len(body) != 1 || defaults[0].From == entry.ID || defaults[0].From != body[0].From {
				t.Fatalf("Ruby callable defaults should be deferred with the body: defaults=%v body=%v", defaults, body)
			}
		})
	}
}
