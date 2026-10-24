package outline

import "testing"

func TestBuildGoLocalBindings(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		resolved   int
		unresolved int
	}{
		{"short declaration", "helper := func(){}; helper()", 0, 1},
		{"var declaration", "var helper = func(){}; helper()", 0, 1},
		{"initializer", "helper := helper; helper()", 0, 1},
		{"initializer call", "helper := func(){ helper() }; helper()", 0, 1},
		{"before declaration", "helper(); helper := func(){}; helper()", 1, 1},
		{"sibling block", "{ helper := func(){}; helper() }; helper()", 1, 1},
		{"if initializer", "if helper := func(){}; true { helper() }; helper()", 1, 1},
		{"range variable", "for _, helper := range []func(){helper} { helper() }; helper()", 1, 1},
		{"destructuring", "helper, other := func(){}, 1; helper(); _ = other", 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{"app.go": "package app\nfunc helper(){}\nfunc entry(){" + c.body + "}\n"})
			assertBindingCalls(t, root, "go", c.resolved, c.unresolved)
		})
	}
}

func TestBuildPythonLocalBindings(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		resolved   int
		unresolved int
	}{
		{"assignment", "    helper = lambda: None\n    helper()\n", 0, 1},
		{"before assignment", "    helper()\n    helper = lambda: None\n    helper()\n", 0, 2},
		{"destructuring", "    helper, other = (lambda: None), 1\n    helper()\n", 0, 1},
		{"loop variable", "    for helper in []:\n        helper()\n", 0, 1},
		{"sibling function", "    def nested():\n        helper = lambda: None\n        helper()\n    helper()\n", 1, 0},
		{"nested definition", "    helper = lambda: None\n    def nested():\n        def helper():\n            pass\n        helper()\n    helper()\n", 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{"app.py": "def helper():\n    pass\ndef entry():\n" + c.body})
			assertBindingCalls(t, root, "python", c.resolved, c.unresolved)
		})
	}
}

func assertBindingCalls(t *testing.T, root, language string, wantResolved, wantUnresolved int) {
	t.Helper()
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	helper := nodeByName(g, KindFunc, "helper")
	if entry == nil || helper == nil {
		t.Fatal("missing fixture functions")
	}
	resolved, unresolved := 0, 0
	for _, e := range g.Callees(entry.ID) {
		switch e.To {
		case helper.ID:
			resolved++
		case ExtID(language, "", "helper"):
			unresolved++
		}
	}
	if resolved != wantResolved || unresolved != wantUnresolved {
		t.Fatalf("resolved=%d unresolved=%d, want %d/%d; calls=%v", resolved, unresolved, wantResolved, wantUnresolved, g.Callees(entry.ID))
	}
}

func TestBuildLocalImportShadow(t *testing.T) {
	cases := []struct{ name, language, filename, source string }{
		{"Go variable", "go", "app.go", "package app\nimport \"example.com/app/util\"\nfunc entry(){util := struct{Run func()}{func(){}}; util.Run()}\n"},
		{"Go type", "go", "app.go", "package app\nimport \"example.com/app/util\"\ntype Runner struct{}\nfunc (Runner) Run(){}\nfunc entry(){type util = Runner; util.Run(util{})}\n"},
		{"Python variable", "python", "app.py", "import util\ndef entry():\n    util = object()\n    util.Run()\n"},
		{"Python class", "python", "app.py", "import util\nclass util:\n    def Run():\n        pass\ndef entry():\n    util.Run()\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"go.mod":       "module example.com/app\n",
				c.filename:     c.source,
				"util/util.go": "package util\nfunc Run(){}\n",
				"util.py":      "def Run():\n    pass\n",
			})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			if edge(g, entry.ID, ExtID(c.language, "util", "Run"), RelCalls) == nil {
				t.Fatalf("local receiver resolved through import: %v", g.Callees(entry.ID))
			}
		})
	}
}

func TestBuildGoBlockTypeScope(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":       "module example.com/app\n",
		"app.go":       "package app\nimport \"example.com/app/util\"\ntype Runner struct{}\nfunc entry(){ { type util = Runner; _ = util{} }; util.Run() }\n",
		"util/util.go": "package util\nfunc Run(){}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	run := nodeByName(g, KindFunc, "Run")
	if edge(g, entry.ID, run.ID, RelCalls) == nil {
		t.Fatalf("block type hid the import after leaving scope: %v", g.Callees(entry.ID))
	}
}

func TestBuildCallableLocalBoundaries(t *testing.T) {
	cases := []struct{ language, filename, source, target string }{
		{"python", "app.py", "def _():\n    pass\ndef entry():\n    _ = lambda: None\n    _()\n", ExtID("python", "", "_")},
		{"ruby", "app.rb", "def helper\nend\ndef entry\n  helper = -> { nil }\n  helper.call\nend\n", ExtID("ruby", "helper", "call")},
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
			if edge(g, entry.ID, c.target, RelCalls) == nil {
				t.Fatalf("callable local lost its unresolved target: %v", g.Callees(entry.ID))
			}
		})
	}
}
