package outline

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
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

func nodeByQualified(g *Graph, qualified string) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].Qualified == qualified {
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

func TestBuildRubyCalls(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"worker.rb": `class Worker
  def run(client)
    helper()
    self.helper
    File.read("config.yml")
    client.get("/")
  end

  def helper
  end

  def self.run
    build()
  end

  def self.build
  end

  build()
end

def Worker.configure
end

def system(command)
end

def invoke
  system("echo")
  Worker.run
  Worker.configure
end
`,
	})

	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	worker := nodeByQualified(g, "Worker")
	instanceRun := nodeByQualified(g, "Worker#run")
	singletonRun := nodeByQualified(g, "Worker.run")
	helper := nodeByQualified(g, "Worker#helper")
	build := nodeByQualified(g, "Worker.build")
	configure := nodeByQualified(g, "Worker.configure")
	invoke := nodeByQualified(g, "invoke")
	localSystem := nodeByQualified(g, "system")
	if worker == nil || instanceRun == nil || singletonRun == nil || helper == nil || build == nil || configure == nil || invoke == nil || localSystem == nil {
		t.Fatalf("missing Ruby nodes: worker=%v instance=%v singleton=%v helper=%v build=%v configure=%v invoke=%v system=%v",
			worker, instanceRun, singletonRun, helper, build, configure, invoke, localSystem)
	}

	if e := edge(g, instanceRun.ID, helper.ID, RelCalls); e == nil || e.Conf != ConfExtracted || e.Call == nil {
		t.Errorf("missing instance helper call with facts: %v", e)
	}
	if e := edge(g, singletonRun.ID, build.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("missing singleton build call: %v", e)
	}
	if e := edge(g, worker.ID, build.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("missing class-body build call: %v", e)
	}
	if e := edge(g, invoke.ID, localSystem.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("local system should shadow Kernel.system: %v", e)
	}
	if e := edge(g, invoke.ID, singletonRun.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("missing Worker.run call: %v", e)
	}
	if e := edge(g, invoke.ID, configure.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("missing Worker.configure call: %v", e)
	}

	read := ExtID("ruby", "File", "read")
	if e := edge(g, instanceRun.ID, read, RelCalls); e == nil || e.Call == nil || len(e.Call.Arguments) != 1 {
		t.Errorf("missing File.read external call facts: %v", e)
	}
	get := ExtID("ruby", "client", "get")
	if e := edge(g, instanceRun.ID, get, RelCalls); e == nil || e.Call == nil || e.Call.ReceiverKind != ReceiverLocal {
		t.Errorf("missing unresolved client.get call facts: %v", e)
	}
}

func TestBuildRubyLoadsAndExecutables(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb": `require "json"
require_relative "lib/helper"
Kernel.require "net/http"
require "#{name}/client"
require File.join("plugins", name)
`,
		"lib/helper.rb":  "module Helper\nend\n",
		"bin/tool":       "#!/usr/bin/env ruby\nFile.delete(\"stale\")\n",
		"sample.gemspec": "Gem::Specification.new do |spec|\n  spec.name = \"sample\"\nend\n",
	})

	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"app.rb", "lib/helper.rb", "bin/tool", "sample.gemspec"} {
		if g.Node(FileID(file)) == nil {
			t.Errorf("missing Ruby file node %q", file)
		}
	}
	if e := edge(g, FileID("bin/tool"), ExtID("ruby", "File", "delete"), RelCalls); e == nil {
		t.Error("Ruby executable was selected but not analysed")
	}

	relative := ModID("ruby", "lib/helper")
	if e := edge(g, FileID("app.rb"), relative, RelLoads); e == nil || e.Operation != "require_relative" {
		t.Errorf("missing require_relative load edge: %v", e)
	}
	if e := edge(g, relative, FileID("lib/helper.rb"), RelContains); e == nil {
		t.Errorf("require_relative did not resolve helper: %v", e)
	}
	if g.Node(ModID("ruby", "#{name}/client")) != nil {
		t.Error("dynamic require became a fixed module")
	}
	if g.Node(ModID("ruby", "plugins")) != nil {
		t.Error("nested File.join string became a fixed module")
	}
	if e := edge(g, FileID("app.rb"), ExtID("ruby", "", "require"), RelCalls); e == nil {
		t.Error("dynamic require was omitted instead of retained as an unresolved call")
	}
}

func TestBuildRubyConstantShadow(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"shadow.rb": `class File
end

def read
  File.read("local")
end
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if g.Node(ExtID("ruby", "File", "read")) != nil {
		t.Error("local File constant resolved as core File")
	}
	if g.Node(ExtID("ruby", "local:File", "read")) == nil {
		t.Error("missing unresolved target for shadowing File constant")
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

func TestBuildGoMultiName(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module m\n",
		"m.go":   "package m\n\nvar a, b int\n\nconst c, d = 1, 2\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]string)
	for _, n := range g.Nodes {
		if other, dup := seen[n.ID]; dup {
			t.Errorf("duplicate node ID %s: %q and %q", n.ID, other, n.Name)
		}
		seen[n.ID] = n.Name
	}
	for _, name := range []string{"a", "b", "c", "d"} {
		if nodeByName(g, KindVar, name) == nil && nodeByName(g, KindConst, name) == nil {
			t.Errorf("missing decl node %q", name)
		}
	}
	for _, e := range g.Edges {
		if e.Rel == RelContains && e.From != FileID("m.go") && g.Node(e.From).Kind != KindFile {
			t.Errorf("multi-name decl got non-file parent: %v", e)
		}
	}
}

func TestBuildPythonRelativeDistinct(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a/__init__.py": "",
		"a/app.py":      "from .util import run\ndef fa(): run()\n",
		"a/util.py":     "def run(): pass\n",
		"b/__init__.py": "",
		"b/app.py":      "from .util import run\ndef fb(): run()\n",
		"b/util.py":     "def run(): pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fa := nodeByName(g, KindFunc, "fa")
	fb := nodeByName(g, KindFunc, "fb")
	if fa == nil || fb == nil {
		t.Fatalf("missing fa=%v fb=%v", fa, fb)
	}
	target := func(from string) string {
		for _, e := range g.Out(from) {
			if e.Rel == RelCalls {
				return e.To
			}
		}
		return ""
	}
	ta, tb := target(fa.ID), target(fb.ID)
	if !strings.Contains(ta, "a/util.py") {
		t.Errorf("fa resolved to %q, want a/util.py", ta)
	}
	if !strings.Contains(tb, "b/util.py") {
		t.Errorf("fb resolved to %q, want b/util.py", tb)
	}
	if ta == tb {
		t.Errorf("both packages resolved to same target %q", ta)
	}
}

func TestBuildExtQualifiedCanonical(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.py": "import subprocess as one\ndef fa(): one.run('x')\n",
		"b.py": "import subprocess as two\ndef fb(): two.run('x')\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ext := g.Node(ExtID("python", "subprocess", "run"))
	if ext == nil {
		t.Fatal("missing ext:python:subprocess:run node")
	}
	if ext.Qualified != "subprocess.run" {
		t.Errorf("ext Qualified = %q, want subprocess.run", ext.Qualified)
	}
	if hits := g.Def("subprocess.run"); len(hits) != 1 || hits[0].ID != ext.ID {
		t.Errorf("Def(subprocess.run) = %v", hits)
	}
}

func TestBuildGoModuleBoundary(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":         "module example.com/svc\n",
		"main.go":        "package main\nimport \"example.com/svc2/util\"\nfunc main() { util.Run() }\n",
		"2/util/util.go": "package util\nfunc Run() {}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	mid := ModID("go", "example.com/svc2/util")
	if e := edge(g, mid, FileID("2/util/util.go"), RelContains); e != nil {
		t.Errorf("svc2 import wrongly resolved into svc module: %v", e)
	}
}

func TestBuildGoVersionedAlias(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":  "module m\n",
		"main.go": "package main\nimport \"github.com/foo/bar/v3\"\nfunc main() { bar.Do() }\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := ExtID("go", "github.com/foo/bar/v3", "Do")
	if g.Node(want) == nil {
		t.Fatalf("bar.Do() did not resolve via versioned alias; want %s, ext nodes: %v", want, g.Def("Do"))
	}
	if defaultAlias("go", "gopkg.in/vault") != "vault" {
		t.Errorf("non-numeric v-prefix segment stripped: %q", defaultAlias("go", "gopkg.in/vault"))
	}
}

func TestBuildPythonRelativeExtDistinct(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a/__init__.py": "",
		"a/app.py":      "from .missing import run\ndef fa(): run()\n",
		"b/__init__.py": "",
		"b/app.py":      "from .missing import run\ndef fb(): run()\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fa := nodeByName(g, KindFunc, "fa")
	fb := nodeByName(g, KindFunc, "fb")
	target := func(from string) string {
		for _, e := range g.Out(from) {
			if e.Rel == RelCalls {
				return e.To
			}
		}
		return ""
	}
	ta, tb := target(fa.ID), target(fb.ID)
	if ta == tb {
		t.Errorf("unresolved relative imports collided: fa->%q fb->%q", ta, tb)
	}
	if !strings.HasPrefix(ta, "ext:") || !strings.HasPrefix(tb, "ext:") {
		t.Errorf("expected ext: targets, got %q %q", ta, tb)
	}
}

func TestBuildLexicalScope(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"n.py": `def outer():
    def inner():
        pass
    inner()

def helper():
    pass

class C:
    def m(self):
        m()
        helper()

def caller(helper):
    helper()
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	outer := nodeByName(g, KindFunc, "outer")
	inner := nodeByName(g, KindFunc, "inner")
	helperFn := nodeByName(g, KindFunc, "helper")
	m := nodeByName(g, KindFunc, "m")
	caller := nodeByName(g, KindFunc, "caller")

	if e := edge(g, outer.ID, inner.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("nested inner() should resolve extracted: %v", e)
	}
	if e := edge(g, m.ID, m.ID, RelCalls); e != nil {
		t.Errorf("m() inside method should not resolve to the method: %v", e)
	}
	if e := edge(g, m.ID, helperFn.ID, RelCalls); e == nil || e.Conf != ConfExtracted {
		t.Errorf("helper() from method should resolve to top-level extracted: %v", e)
	}
	if e := edge(g, caller.ID, helperFn.ID, RelCalls); e != nil {
		t.Errorf("helper() shadowed by param should not resolve to top-level: %v", e)
	}
	extHelper := ExtID("python", "", "helper")
	if e := edge(g, caller.ID, extHelper, RelCalls); e == nil {
		t.Errorf("shadowed helper() should produce ext edge; edges from caller: %v", g.Out(caller.ID))
	}
}

func TestBuildGoParamShadow(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module example.com/app\n",
		"main.go": `package main
import "example.com/app/util"
func Handler() {}
func A(Handler func()) { Handler() }
func B(util T) { util.Run() }
`,
		"util/util.go": "package util\nfunc Run() {}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	a := nodeByName(g, KindFunc, "A")
	handler := nodeByName(g, KindFunc, "Handler")
	if e := edge(g, a.ID, handler.ID, RelCalls); e != nil {
		t.Errorf("Handler() shadowed by param should not resolve to top-level: %v", e)
	}
	b := nodeByName(g, KindFunc, "B")
	runFn := nodeByName(g, KindFunc, "Run")
	if e := edge(g, b.ID, runFn.ID, RelCalls); e != nil {
		t.Errorf("util.Run() with util param should not resolve via import: %v", e)
	}
}

func TestBuildDeterministic(t *testing.T) {
	root := t.TempDir()
	// pkg defines D in two files, as build-tag variants do, so resolution
	// must pick the same declaration on every build.
	writeFiles(t, root, map[string]string{
		"go.mod":    "module m\n",
		"a.go":      "package m\n\nimport \"m/pkg\"\n\nfunc A() { B(); pkg.D() }\n",
		"b.go":      "package m\nfunc B() {}\n",
		"pkg/d1.go": "package pkg\nfunc D() {}\n",
		"pkg/d2.go": "package pkg\nfunc D() {}\n",
		"c/c.py":    "def c(): pass\n",
	})
	var out [4]bytes.Buffer
	for i := range out {
		g, err := Build(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.JSON(&out[i]); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i < len(out); i++ {
		if !bytes.Equal(out[0].Bytes(), out[i].Bytes()) {
			t.Fatalf("Build not deterministic:\n%s\n---\n%s", out[0].String(), out[i].String())
		}
	}
}

func TestSignatureRuneBoundary(t *testing.T) {
	src := []byte("func " + strings.Repeat("a", sigCap-6) + "é()")
	d := decl{Start: 0, End: uint32(len(src)), SigEnd: uint32(len(src))}
	s := signature(src, d)
	if !utf8.ValidString(s) {
		t.Errorf("signature is not valid UTF-8: %q", s)
	}
	if len(s) > sigCap {
		t.Errorf("signature length %d exceeds cap %d", len(s), sigCap)
	}
}
