package outline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRubyEquivalentLoadClosures(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprintf("conflict=%t", conflict), func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"barrel.rb": "require_relative 'a'\nrequire_relative 'b'\nrequire_relative 'helper'\n",
				"a.rb":      "require_relative 'barrel'\ndef entry_a\n  Helper.ready\nend\n",
				"b.rb":      "require_relative 'barrel'\ndef entry_b\n  Helper.ready\nend\n",
				"helper.rb": "require_relative 'barrel'\nmodule Helper\n  def self.ready\n    File.read('config')\n  end\nend\n",
				"alone.rb":  "def entry_alone\n  Helper.ready\nend\n",
			}
			if conflict {
				files["barrel.rb"] += "require_relative 'conflict'\n"
				files["conflict.rb"] = "require_relative 'barrel'\nmodule Helper\n  def self.ready\n  end\nend\n"
			}
			writeFiles(t, root, files)
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"entry_a", "entry_b", "entry_alone"} {
				entry := nodeByQualified(g, name)
				calls := g.Callees(entry.ID)
				if len(calls) != 1 {
					t.Fatalf("%s calls=%v", name, calls)
				}
				target := g.Node(calls[0].To)
				if conflict || name == "entry_alone" {
					if target.Kind != KindExternal {
						t.Fatalf("%s selected conflicting or unloaded source: %+v", name, target)
					}
				} else if target.File != "helper.rb" || target.Qualified != "Helper.ready" {
					t.Fatalf("equivalent load closure lost %s target: %+v", name, target)
				}
			}
		})
	}
}

func BenchmarkBuildRubySharedLoadClosure(b *testing.B) {
	const files, methods = 64, 20
	root := b.TempDir()
	var barrel strings.Builder
	for i := range files {
		name := fmt.Sprintf("worker_%03d.rb", i)
		fmt.Fprintf(&barrel, "require_relative '%s'\n", strings.TrimSuffix(name, ".rb"))
		var source strings.Builder
		fmt.Fprintf(&source, "require_relative 'barrel'\nmodule Worker%d\n", i)
		for method := range methods {
			fmt.Fprintf(&source, "  def self.method%d\n    File.read('config')\n  end\n", method)
		}
		source.WriteString("end\n")
		if err := os.WriteFile(filepath.Join(root, name), []byte(source.String()), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "barrel.rb"), []byte(barrel.String()), 0o600); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Build(root, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}
