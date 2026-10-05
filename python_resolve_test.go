package outline

import "testing"

func TestBuildPythonPackageJoins(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py": `import pkg
import pkg.loader
import pkg.loader as direct
from pkg import loader as sub
from pkg import encode as public
def entry():
    pkg.encode()
    public()
    pkg.loader.run()
    direct.run()
    sub.run()
`,
		"pkg/__init__.py": "from .api import encode\n",
		"pkg/api.py":      "from .impl import encode\n",
		"pkg/impl.py":     "def encode():\n    pass\n",
		"pkg/loader.py":   "def run():\n    pass\n",
		"pkg.py":          "def other():\n    pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	encode := nodeByName(g, KindFunc, "encode")
	run := nodeByName(g, KindFunc, "run")
	loader := ModID("python", "pkg.loader")
	if edge(g, FileID("app.py"), loader, RelImports) == nil || edge(g, loader, FileID("pkg/loader.py"), RelContains) == nil {
		t.Error("resolved submodule import did not retain its source connection")
	}
	calls := g.Callees(entry.ID)
	if len(calls) != 5 {
		t.Fatalf("calls=%v, want all five calls", calls)
	}
	for _, call := range calls {
		want := run.ID
		if call.Call.Name == "encode" || call.Call.Name == "public" {
			want = encode.ID
		}
		if call.To != want {
			t.Errorf("%s.%s target=%s, want %s", call.Call.Receiver, call.Call.Name, call.To, want)
		}
	}
}

func TestBuildPythonModuleReexports(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py":          "import pkg\nfrom pkg import exposed as local\ndef entry():\n    pkg.exposed.run()\n    local.run()\n",
		"pkg/__init__.py": "from . import loader\nfrom . import loader as exposed\n",
		"pkg/loader.py":   "def run():\n    pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	run := nodeByName(g, KindFunc, "run")
	if calls := g.Callees(entry.ID); len(calls) != 2 {
		t.Fatalf("calls=%v", calls)
	}
	for _, call := range g.Callees(entry.ID) {
		if call.To != run.ID {
			t.Errorf("module re-export target=%s, want %s", call.To, run.ID)
		}
	}
}

func TestBuildPythonUncertainReexports(t *testing.T) {
	for _, c := range []struct{ name, source string }{
		{"cycle", "from .other import run\n"},
		{"conflict", "from .one import run\nfrom .two import run\n"},
		{"missing-conflict", "from .one import run\nfrom .missing import run\n"},
		{"nested-import", "def setup():\n    from .one import run\n"},
		{"class-import", "class Setup:\n    from .one import run\n"},
		{"definition-conflict", "from .one import run\ndef run():\n    pass\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"app.py":          "import pkg\ndef entry():\n    pkg.run()\n",
				"pkg/__init__.py": c.source,
				"pkg/other.py":    "from . import run\n",
				"pkg/one.py":      "def run():\n    pass\n",
				"pkg/two.py":      "def run():\n    pass\n",
			})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			if edge(g, entry.ID, ExtID("python", "pkg", "run"), RelCalls) == nil {
				t.Fatalf("uncertain re-export bound to a source target: %v", g.Callees(entry.ID))
			}
		})
	}
}

func TestBuildPythonReceiverChains(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py": `import pkg
def entry():
    pkg.missing.run()
    factory().run()
def shadow(pkg):
    pkg.run()
def factory():
    pass
`,
		"pkg/__init__.py": "def run():\n    pass\n",
		"pkg/missing.py":  "def run():\n    pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	for _, call := range g.Callees(entry.ID) {
		if call.Call.Name == "run" && g.Node(call.To).Kind != KindExternal {
			t.Errorf("receiver chain collapsed into a package function: %+v", call)
		}
	}
	shadow := nodeByName(g, KindFunc, "shadow")
	if edge(g, shadow.ID, ExtID("python", "pkg", "run"), RelCalls) == nil {
		t.Error("parameter receiver resolved through a module")
	}
}

func TestBuildPythonStandaloneModuleHasNoSubmodules(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py":         "from util import loader\ndef entry():\n    loader.run()\n",
		"util.py":        "def other():\n    pass\n",
		"util/loader.py": "def run():\n    pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	if edge(g, entry.ID, ExtID("python", "loader", "run"), RelCalls) == nil {
		t.Fatalf("standalone util.py incorrectly provided a submodule: %v", g.Callees(entry.ID))
	}
}

func TestBuildPythonWildcardExports(t *testing.T) {
	for _, c := range []struct {
		name, exports string
		resolved      bool
	}{
		{"implicit", "", true},
		{"listed", "__all__ = ['run']\n", true},
		{"excluded", "__all__ = []\n", false},
		{"dynamic", "__all__ = names()\n", false},
		{"duplicate", "__all__ = ['run']\n__all__ = []\n", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"app.py":          "from pkg import *\ndef entry():\n    run()\n",
				"pkg/__init__.py": c.exports + "from .impl import run\n",
				"pkg/impl.py":     "def run():\n    pass\n",
			})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			run := nodeByName(g, KindFunc, "run")
			if got := edge(g, entry.ID, run.ID, RelCalls) != nil; got != c.resolved {
				t.Fatalf("resolved=%t, want %t: %v", got, c.resolved, g.Callees(entry.ID))
			}
		})
	}
}

func TestBuildPythonLocalImports(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py": `from one import run
def local():
    run()
    from two import run
    run()
def entry():
    run()
class Holder:
    from two import run
    def method(self):
        run()
`,
		"one.py": "def run():\n    pass\n",
		"two.py": "def run():\n    pass\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		function string
		files    []string
	}{
		{"local", []string{"", "two.py"}},
		{"entry", []string{"one.py"}},
		{"method", []string{"one.py"}},
	} {
		fn := nodeByName(g, KindFunc, c.function)
		calls := g.Callees(fn.ID)
		if len(calls) != len(c.files) {
			t.Fatalf("%s calls=%v", c.function, calls)
		}
		for i, call := range calls {
			if got := g.Node(call.To).File; got != c.files[i] {
				t.Errorf("%s call %d target=%s, want file %s", c.function, i, call.To, c.files[i])
			}
		}
	}
}
