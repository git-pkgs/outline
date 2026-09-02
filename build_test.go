package outline

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func edge(g *Graph, from, to, rel string) *Edge {
	for i := range g.Edges {
		e := &g.Edges[i]
		if e.From == from && e.To == to && e.Rel == rel {
			return e
		}
	}
	return nil
}

func nodeByName(g *Graph, kind, name string) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].Kind == kind && g.Nodes[i].Name == name {
			return &g.Nodes[i]
		}
	}
	return nil
}

func TestBuildGoCrossPackage(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.26\n",
		"main.go": `package main

import (
	"os/exec"
	"example.com/app/util"
)

func main() {
	util.Run("ls")
	exec.Command("ls")
}
`,
		"util/util.go": `package util

func Run(name string) error {
	return nil
}
`,
	})

	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	mainFn := nodeByName(g, KindFunc, "main")
	runFn := nodeByName(g, KindFunc, "Run")
	execMod := nodeByName(g, KindModule, "os/exec")
	if mainFn == nil || runFn == nil || execMod == nil {
		t.Fatalf("missing nodes: main=%v Run=%v exec=%v", mainFn, runFn, execMod)
	}
	if !runFn.Exported {
		t.Errorf("Run should be exported")
	}
	if runFn.Sig == "" || !bytes.Contains([]byte(runFn.Sig), []byte("name string")) {
		t.Errorf("Run sig = %q, want to contain parameter name", runFn.Sig)
	}

	utilMod := ModID("go", "example.com/app/util")
	if g.Node(utilMod) == nil {
		t.Fatalf("missing module node %s", utilMod)
	}
	if e := edge(g, FileID("main.go"), utilMod, RelImports); e == nil {
		t.Error("missing main.go -> util imports edge")
	}
	if e := edge(g, utilMod, FileID("util/util.go"), RelContains); e == nil {
		t.Error("util module did not resolve to util/util.go")
	}

	call := edge(g, mainFn.ID, runFn.ID, RelCalls)
	if call == nil {
		t.Fatalf("missing main -> util.Run calls edge; edges from main: %v", g.Out(mainFn.ID))
	}
	if call.Conf != ConfInferred {
		t.Errorf("cross-package call conf = %q, want inferred", call.Conf)
	}

	extCmd := ExtID("go", "os/exec", "Command")
	if g.Node(extCmd) == nil {
		t.Fatalf("missing external node %s", extCmd)
	}
	if e := edge(g, mainFn.ID, extCmd, RelCalls); e == nil {
		t.Error("missing main -> exec.Command external edge")
	}
}

func TestBuildPythonCrossModule(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py": `import subprocess
from util import run

def handler(name):
    run(name)
    subprocess.call(name)
`,
		"util.py": `def run(name):
    pass
`,
	})

	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	handler := nodeByName(g, KindFunc, "handler")
	runFn := nodeByName(g, KindFunc, "run")
	if handler == nil || runFn == nil {
		t.Fatalf("missing nodes: handler=%v run=%v", handler, runFn)
	}

	utilMod := ModID("python", "util")
	if e := edge(g, utilMod, FileID("util.py"), RelContains); e == nil {
		t.Error("util module did not resolve to util.py")
	}

	call := edge(g, handler.ID, runFn.ID, RelCalls)
	if call == nil {
		t.Fatalf("missing handler -> run calls edge; edges: %v", g.Out(handler.ID))
	}
	if call.Conf != ConfInferred {
		t.Errorf("cross-module call conf = %q, want inferred", call.Conf)
	}

	extCall := ExtID("python", "subprocess", "call")
	if g.Node(extCall) == nil {
		t.Fatalf("missing external node %s", extCall)
	}
	if e := edge(g, handler.ID, extCall, RelCalls); e == nil {
		t.Error("missing handler -> subprocess.call external edge")
	}
}

func TestBuildSameFileExtracted(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.py": "def a():\n    b()\n\ndef b():\n    pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	aFn := nodeByName(g, KindFunc, "a")
	bFn := nodeByName(g, KindFunc, "b")
	e := edge(g, aFn.ID, bFn.ID, RelCalls)
	if e == nil {
		t.Fatal("missing a -> b calls edge")
	}
	if e.Conf != ConfExtracted {
		t.Errorf("same-file call conf = %q, want extracted", e.Conf)
	}
}

func TestBuildGoSamePackage(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module m\n",
		"a.go":   "package m\nfunc A() { helper() }\n",
		"b.go":   "package m\nfunc helper() {}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	aFn := nodeByName(g, KindFunc, "A")
	hFn := nodeByName(g, KindFunc, "helper")
	e := edge(g, aFn.ID, hFn.ID, RelCalls)
	if e == nil {
		t.Fatalf("missing A -> helper same-package edge; edges from A: %v", g.Out(aFn.ID))
	}
	if e.Conf != ConfInferred {
		t.Errorf("same-package cross-file conf = %q, want inferred", e.Conf)
	}
	if n := nodeByName(g, KindExternal, "helper"); n != nil {
		t.Errorf("spurious external node for resolved same-package call: %v", n)
	}
}

func TestBuildDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module m\n",
		"a.go":   "package m\nfunc A() { B() }\n",
		"b.go":   "package m\nfunc B() {}\n",
		"c/c.py": "def c(): pass\n",
	})
	var out [2]bytes.Buffer
	for i := range out {
		g, err := Build(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.JSON(&out[i]); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(out[0].Bytes(), out[1].Bytes()) {
		t.Errorf("Build not deterministic:\n%s\n---\n%s", out[0].String(), out[1].String())
	}
}
