package outline

import "testing"

func TestBuildRubyLoadedSingletons(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb":        "require_relative 'lib/bridge'\ndef entry\n  Helper.ready\n  Other.ready\nend\n",
		"lib/bridge.rb": "require_relative 'helper'\nrequire_relative '../app'\n",
		"lib/helper.rb": "module Helper\n  def self.ready\n    File.read('config')\n  end\nend\n",
		"unloaded.rb":   "module Other\n  def self.ready\n  end\nend\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByQualified(g, "entry")
	ready := nodeByQualified(g, "Helper.ready")
	if ready == nil || ready.Kind != KindFunc {
		t.Fatalf("missing singleton definition: %+v", ready)
	}
	if e := edge(g, entry.ID, ready.ID, RelCalls); e == nil || e.Conf != ConfInferred {
		t.Fatalf("loaded singleton target missing: %v", g.Callees(entry.ID))
	}
	if edge(g, entry.ID, ExtID("ruby", "Other", "ready"), RelCalls) == nil {
		t.Error("unloaded source was used as a target")
	}
	if path := g.Path(entry.ID, ExtID("ruby", "File", "read"), TraverseOptions{IncludeInferred: true}); len(path) != 2 {
		t.Fatalf("loaded source path=%v", path)
	}
}

func TestBuildRubyModuleFunctions(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb": "require_relative 'utils'\ndef entry\n  Rack::Utils.escape('a b')\n  Named.run\n  Named.other\nend\n",
		"utils.rb": `module Rack
  module Utils
    module_function
    def escape(value)
      encode(value)
    end
    def encode(value)
      URI.encode_www_form_component(value)
    end
  end
end
module Named
  def run
  end
  module_function :run
  def other
  end
end
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByQualified(g, "entry")
	for _, name := range []string{"Rack::Utils.escape", "Named.run"} {
		target := nodeByQualified(g, name)
		if target == nil || target.Kind != KindFunc || edge(g, entry.ID, target.ID, RelCalls) == nil {
			t.Errorf("module function %s did not resolve", name)
		}
	}
	escape := nodeByQualified(g, "Rack::Utils.escape")
	encode := nodeByQualified(g, "Rack::Utils.encode")
	if escape != nil && encode != nil && edge(g, escape.ID, encode.ID, RelCalls) == nil {
		t.Error("module-function body did not call the copied module method")
	}
	if edge(g, entry.ID, ExtID("ruby", "local:Named", "other"), RelCalls) == nil {
		t.Error("named module_function incorrectly applied to subsequent definitions")
	}
}

func TestBuildRubyConflictingLoadedMethods(t *testing.T) {
	for _, c := range []struct{ name, extra string }{
		{"duplicate", "module Helper\n  def self.ready\n  end\nend\n"},
		{"assignment", "Helper = Object.new\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"app.rb": "require_relative 'one'\nrequire_relative 'two'\ndef entry\n  Helper.ready\nend\n",
				"one.rb": "module Helper\n  def self.ready\n  end\nend\n",
				"two.rb": c.extra,
			})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByQualified(g, "entry")
			calls := g.Callees(entry.ID)
			if len(calls) != 1 || g.Node(calls[0].To).Kind != KindExternal {
				t.Fatalf("conflicting definitions selected one target: %v", calls)
			}
		})
	}
}

func TestBuildRubyScopedConstants(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb": `module Other
  module Helper
    def self.ready
    end
  end
end
def entry
  Helper.ready
end
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByQualified(g, "entry")
	if calls := g.Callees(entry.ID); len(calls) != 1 || g.Node(calls[0].To).Kind != KindExternal {
		t.Fatalf("unqualified constant bound to unrelated nested owner: %v", calls)
	}
}

func TestBuildRubyModuleFunctionBoundaries(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb": `module M
  module_function
  def copied
  end
  public
  def instance
  end
  module Nested
    def instance
    end
  end
end
module Named
  def run
    File.read('old')
  end
  module_function :run
  def run
    File.read('new')
  end
end
def entry
  M.copied
  M.instance
  M::Nested.instance
  Named.run
end
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByQualified(g, "entry")
	for _, name := range []string{"M.copied", "Named.run"} {
		fn := nodeByQualified(g, name)
		if fn == nil || fn.Kind != KindFunc || edge(g, entry.ID, fn.ID, RelCalls) == nil {
			t.Errorf("copied method %s missing", name)
		}
	}
	for _, owner := range []string{"M", "M::Nested"} {
		if edge(g, entry.ID, ExtID("ruby", "local:"+owner, "instance"), RelCalls) == nil {
			t.Errorf("module-function mode leaked into %s", owner)
		}
	}
	run := nodeByQualified(g, "Named.run")
	if run != nil {
		calls := g.Callees(run.ID)
		if len(calls) != 1 || calls[0].Call.Arguments[0].Text != "'old'" {
			t.Errorf("named copy changed after instance method redefinition: %v", calls)
		}
	}
}

func TestBuildRubyDeferredLoadsRemainUnknown(t *testing.T) {
	for _, c := range []struct{ name, load string }{
		{"function", "def setup\n  require_relative 'helper'\nend\n"},
		{"conditional", "if enabled\n  require_relative 'helper'\nend\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"app.rb":    c.load + "def entry\n  Helper.ready\nend\n",
				"helper.rb": "module Helper\n  def self.ready\n  end\nend\n",
			})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByQualified(g, "entry")
			if edge(g, entry.ID, ExtID("ruby", "Helper", "ready"), RelCalls) == nil {
				t.Fatalf("deferred source load produced a resolved target: %v", g.Callees(entry.ID))
			}
		})
	}
}

func TestBuildRubyQualifiedConstantShadow(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb": `module Helper
  module Nested
    def self.ready
    end
  end
end
module Outer
  Helper = Object.new
  def self.entry
    Helper::Nested.ready
  end
end
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByQualified(g, "Outer.entry")
	if calls := g.Callees(entry.ID); len(calls) != 1 || g.Node(calls[0].To).Kind != KindExternal {
		t.Fatalf("qualified constant bypassed a local shadow: %v", calls)
	}
}

func TestBuildRubyCustomModuleFunctionRemainsUnknown(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.rb": `module Custom
  def self.module_function(*names)
  end
  module_function
  def run
  end
end
def entry
  Custom.run
end
`,
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByQualified(g, "entry")
	if edge(g, entry.ID, ExtID("ruby", "local:Custom", "run"), RelCalls) == nil {
		t.Fatalf("custom module_function call inferred a native copy: %v", g.Callees(entry.ID))
	}
}
