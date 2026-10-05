package outline

import "testing"

func TestBuildDigestGoModule(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":       "module example.com/app\n",
		"app.go":       "package app\nimport \"example.com/app/util\"\nfunc entry(){util.Ready()}\n",
		"util/util.go": "package util\nfunc Ready(){}\n",
	})
	local, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(local, KindFunc, "entry")
	ready := nodeByName(local, KindFunc, "Ready")
	if edge(local, entry.ID, ready.ID, RelCalls) == nil {
		t.Fatal("fixture did not resolve the local module")
	}
	writeFiles(t, root, map[string]string{"go.mod": "module example.com/other\n"})
	external, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if edge(external, entry.ID, ExtID("go", "example.com/app/util", "Ready"), RelCalls) == nil {
		t.Fatal("changed module identity did not alter resolution")
	}
	if local.SourceDigest == external.SourceDigest {
		t.Fatal("changed Go module resolution retained the same digest")
	}
}

func TestBuildDigestSourceRoots(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"app.py":           "import helper\ndef entry():\n    helper.Ready()\n",
		"first/helper.py":  "def Ready():\n    pass\n",
		"second/helper.py": "def Ready():\n    pass\n",
	})
	cases := []struct {
		roots  []string
		target string
	}{
		{nil, ""},
		{[]string{"first", "second"}, "first/helper.py"},
		{[]string{"second", "first"}, "second/helper.py"},
	}
	digests := make(map[string]bool)
	for _, c := range cases {
		opts := Options{Resolution: ResolutionHints{SourceRoots: c.roots}}
		g, err := Build(root, opts)
		if err != nil {
			t.Fatal(err)
		}
		entry := nodeByName(g, KindFunc, "entry")
		calls := g.Callees(entry.ID)
		if len(calls) != 1 || g.Node(calls[0].To).File != c.target {
			t.Fatalf("roots=%v target=%s calls=%v", c.roots, c.target, calls)
		}
		if digests[g.SourceDigest] {
			t.Fatalf("different resolution inputs shared digest %s", g.SourceDigest)
		}
		digests[g.SourceDigest] = true
		repeated, err := Build(root, opts)
		if err != nil {
			t.Fatal(err)
		}
		if repeated.SourceDigest != g.SourceDigest {
			t.Fatal("same resolution inputs produced a different digest")
		}
	}
}
