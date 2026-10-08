package outline

import (
	"strings"
	"testing"
)

func TestBuildRecoveredSyntax(t *testing.T) {
	cases := []struct{ language, filename, valid, broken string }{
		{"go", "app.go", "package app\nfunc valid(){}\n", "func broken(\n"},
		{"python", "app.py", "def valid():\n    pass\n", "def broken(:\n"},
		{"ruby", "app.rb", "def valid\nend\n", "def broken(\n"},
	}
	for _, c := range cases {
		t.Run(c.language, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{c.filename: c.valid})
			good, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !good.Complete || len(good.Warnings) != 0 {
				t.Fatalf("valid input marked incomplete: complete=%t warnings=%v", good.Complete, good.Warnings)
			}
			writeFiles(t, root, map[string]string{c.filename: c.valid + c.broken})
			partial, err := Build(root, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if partial.Complete || !strings.Contains(strings.Join(partial.Warnings, "\n"), c.filename+": syntax-errors") {
				t.Fatalf("syntax errors not reported: complete=%t warnings=%v", partial.Complete, partial.Warnings)
			}
			if nodeByName(partial, KindFunc, "valid") == nil {
				t.Fatal("valid definition lost after syntax recovery")
			}
		})
	}
}
