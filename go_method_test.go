package outline

import (
	"strings"
	"testing"
)

func TestBuildGoMethodOwners(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"types.go": "package app\ntype A struct{}\ntype B struct{}\ntype Box[T any] struct{}\n",
		"methods.go": `package app
func (a A) Run() { Probe() }
func (b *B) Run() {}
func (*A) Probe() {}
func (box *Box[T]) Get() {}
func (Box[T]) Put() {}
func entry() { Probe(); Run() }
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A.Run", "B.Run", "A.Probe", "Box.Get", "Box.Put"} {
		methods := g.Def(name)
		if len(methods) != 1 {
			t.Errorf("method %s definitions=%v", name, methods)
			continue
		}
		owner, _, _ := strings.Cut(name, ".")
		typeNode := nodeByName(g, KindType, owner)
		if edge(g, typeNode.ID, methods[0].ID, RelContains) == nil {
			t.Errorf("owner %s does not contain method %s", owner, name)
		}
	}
	entry := nodeByName(g, KindFunc, "entry")
	for _, name := range []string{"Probe", "Run"} {
		if edge(g, entry.ID, ExtID("go", "", name), RelCalls) == nil {
			t.Errorf("bare %s incorrectly resolved to a method: %v", name, g.Callees(entry.ID))
		}
	}
	if methods := g.Def("A.Run"); len(methods) == 1 {
		if edge(g, methods[0].ID, ExtID("go", "", "Probe"), RelCalls) == nil {
			t.Error("bare call inside a method resolved to another method")
		}
	}
}

func TestBuildGoMethodsAreNotPackageFunctions(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod": "module example.com/app\n",
		"main.go": `package app
import "example.com/app/util"
type Local struct{}
func (Local) Ready() {}
func entry() { Ready(); OnlyMethod(); util.Ready(); util.OnlyMethod() }
`,
		"functions.go": "package app\nfunc Ready(){}\n",
		"methods.go":   "package app\nfunc (Local) OnlyMethod(){}\n",
		"util/util.go": `package util
type Worker struct{}
func (Worker) Ready() {}
func (Worker) OnlyMethod() {}
func Ready() {}
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	if calls := g.Callees(entry.ID); len(calls) != 4 {
		t.Fatalf("calls=%v, want all four source calls", calls)
	}
	for _, call := range g.Callees(entry.ID) {
		target := g.Node(call.To)
		switch call.Call.Name {
		case "Ready":
			if target.Kind != KindFunc || target.Qualified != "Ready" {
				t.Errorf("package function bound to method: %+v", target)
			}
		case "OnlyMethod":
			if target.Kind != KindExternal {
				t.Errorf("method entered package lookup: %+v", target)
			}
		}
	}
}

func TestBuildGoMethodOwnerScopes(t *testing.T) {
	for _, c := range []struct {
		name, ownerFile, source string
		resolved                bool
	}{
		{"production", "types.go", "package app\ntype Worker struct{}\n", true},
		{"test-only", "types_test.go", "package app\ntype Worker struct{}\n", false},
		{"other-package", "types_test.go", "package app_test\ntype Worker struct{}\n", false},
		{"wrong-kind", "types.go", "package app\nvar Worker = 1\n", false},
		{"ambiguous", "types.go", "package app\ntype Worker struct{}\n", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"methods.go": "package app\nfunc (Worker) Run(){}\n",
				c.ownerFile:  c.source,
			}
			if c.name == "ambiguous" {
				files["variant.go"] = "package app\ntype Worker struct{}\n"
			}
			writeFiles(t, root, files)
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			methods := g.Def("Worker.Run")
			if len(methods) != 1 {
				t.Fatalf("method definitions=%v", methods)
			}
			for _, e := range g.In(methods[0].ID) {
				if e.Rel != RelContains {
					continue
				}
				owner := g.Node(e.From)
				if (owner.Kind == KindType) != c.resolved {
					t.Errorf("owner=%+v, want type owner=%t", owner, c.resolved)
				}
			}
		})
	}
}

func TestBuildMethodBareCallsAcrossLanguages(t *testing.T) {
	for _, c := range []struct{ lang, filename, source string }{
		{"go", "app.go", "package app\ntype A struct{}\nfunc (A) Probe(){}\nfunc entry(){Probe()}\n"},
		{"python", "app.py", "class A:\n    def Probe(self):\n        pass\ndef entry():\n    Probe()\n"},
		{"ruby", "app.rb", "class A\n  def Probe\n  end\nend\ndef entry\n  Probe()\nend\n"},
	} {
		t.Run(c.lang, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{c.filename: c.source})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			if edge(g, entry.ID, ExtID(c.lang, "", "Probe"), RelCalls) == nil {
				t.Fatalf("bare call bound to a class/type method: %v", g.Callees(entry.ID))
			}
		})
	}
}
