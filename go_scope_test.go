package outline

import "testing"

func TestBuildGoPackageScopes(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":           "module example.com/app\n",
		"main.go":          "package app\nfunc entry(){testOnly(); externalOnly(); helper()}\nfunc helper(){}\n",
		"main_test.go":     "package app\nfunc testOnly(){}\nfunc internalTest(){helper();testOnly();externalOnly()}\n",
		"external_test.go": "package app_test\nfunc externalOnly(){}\nfunc externalTest(){externalOnly();helper()}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	internal := nodeByName(g, KindFunc, "internalTest")
	external := nodeByName(g, KindFunc, "externalTest")
	helper := nodeByName(g, KindFunc, "helper")
	internalOnly := nodeByName(g, KindFunc, "testOnly")
	externalOnly := nodeByName(g, KindFunc, "externalOnly")
	for _, c := range []struct {
		from, to string
		resolved bool
	}{
		{entry.ID, helper.ID, true},
		{entry.ID, internalOnly.ID, false},
		{entry.ID, externalOnly.ID, false},
		{internal.ID, helper.ID, true},
		{internal.ID, internalOnly.ID, true},
		{internal.ID, externalOnly.ID, false},
		{external.ID, externalOnly.ID, true},
		{external.ID, helper.ID, false},
	} {
		if got := edge(g, c.from, c.to, RelCalls) != nil; got != c.resolved {
			t.Errorf("call %s -> %s resolved=%t, want %t", c.from, c.to, got, c.resolved)
		}
		if !c.resolved && edge(g, c.from, ExtID("go", "", g.Node(c.to).Name), RelCalls) == nil {
			t.Errorf("unresolved call %s -> %s was lost", c.from, c.to)
		}
	}
}

func TestBuildTestOnlyScope(t *testing.T) {
	cases := []struct{ language, filename, testfile, source, testsource string }{
		{"python", "app.py", "test_app.py", "def entry():\n    testOnly()\n", "def testOnly():\n    pass\n"},
		{"ruby", "app.rb", "test_app.rb", "def entry\n  testOnly()\nend\n", "def testOnly\nend\n"},
	}
	for _, c := range cases {
		t.Run(c.language, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{c.filename: c.source, c.testfile: c.testsource})
			g, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			entry := nodeByName(g, KindFunc, "entry")
			if edge(g, entry.ID, ExtID(c.language, "", "testOnly"), RelCalls) == nil {
				t.Fatalf("separate test-only declaration resolved into the application: %v", g.Callees(entry.ID))
			}
		})
	}
}

func TestBuildGoImportsExcludeTests(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":                "module example.com/app\n",
		"main.go":               "package app\nimport \"example.com/app/util\"\nfunc entry(){util.Ready();util.InternalOnly();util.ExternalOnly()}\n",
		"util/util.go":          "package util\nfunc Ready(){}\n",
		"util/util_test.go":     "package util\nfunc InternalOnly(){}\n",
		"util/external_test.go": "package util_test\nfunc ExternalOnly(){}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	ready := nodeByName(g, KindFunc, "Ready")
	if edge(g, entry.ID, ready.ID, RelCalls) == nil {
		t.Fatal("production export did not resolve")
	}
	for _, name := range []string{"InternalOnly", "ExternalOnly"} {
		if edge(g, entry.ID, ExtID("go", "example.com/app/util", name), RelCalls) == nil {
			t.Errorf("test export %s did not remain external", name)
		}
	}
}

func TestBuildGoAmbiguousVariants(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":          "module example.com/app\n",
		"main.go":         "package app\nimport \"example.com/app/util\"\nfunc entry(){util.Run();variant()}\n",
		"linux.go":        "//go:build linux\n\npackage app\nfunc variant(){}\n",
		"windows.go":      "//go:build windows\n\npackage app\nfunc variant(){}\n",
		"util/linux.go":   "//go:build linux\n\npackage util\nfunc Run(){}\n",
		"util/windows.go": "//go:build windows\n\npackage util\nfunc Run(){}\n",
	})
	g, err := Build(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	entry := nodeByName(g, KindFunc, "entry")
	for _, target := range []string{ExtID("go", "example.com/app/util", "Run"), ExtID("go", "", "variant")} {
		if edge(g, entry.ID, target, RelCalls) == nil {
			t.Errorf("ambiguous target %s resolved arbitrarily: %v", target, g.Callees(entry.ID))
		}
	}
}
