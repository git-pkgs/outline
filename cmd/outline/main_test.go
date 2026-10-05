package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	dir, err := os.MkdirTemp("", "outline-cli-test")
	if err != nil {
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	binPath = filepath.Join(dir, "outline")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		_, _ = os.Stderr.Write(out)
		return 1
	}
	return m.Run()
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("outline %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func testdata(rel string) string {
	return filepath.Join("..", "..", "testdata", rel)
}

func TestCLIGoAffected(t *testing.T) {
	dir := testdata("cli-go")
	gpath := filepath.Join(t.TempDir(), "g.json")

	run(t, "graph", "-o", gpath, dir)
	if fi, err := os.Stat(gpath); err != nil || fi.Size() == 0 {
		t.Fatalf("graph.json not written: %v", err)
	}

	out := run(t, "affected", "-g", gpath, "-inferred", "ext:go:os/exec:Command")
	if !strings.Contains(out, "Handler") {
		t.Errorf("affected output missing Handler:\n%s", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("affected output missing main (transitive caller):\n%s", out)
	}

	out = run(t, "path", "-g", gpath, "-inferred", "Handler", "Load")
	if !strings.Contains(out, "--calls[inferred]--> Load") {
		t.Errorf("path output missing cross-package call:\n%s", out)
	}
}

func TestCLIPythonAffected(t *testing.T) {
	dir := testdata("cli-py")

	out := run(t, "affected", "-dir", dir, "-inferred", "ext:python:subprocess:run")
	if !strings.Contains(out, "handler") {
		t.Errorf("affected output missing handler:\n%s", out)
	}

	out = run(t, "def", "-dir", dir, "load")
	if !strings.Contains(out, "pkg/loader.py") {
		t.Errorf("def output missing loader location:\n%s", out)
	}
}

func TestCLIRubyAffected(t *testing.T) {
	dir := testdata("cli-ruby")

	out := run(t, "affected", "-dir", dir, "-inferred", "ext:ruby:File:read")
	if !strings.Contains(out, "Worker#run") {
		t.Errorf("affected output missing Ruby method:\n%s", out)
	}

	out = run(t, "affected", "-dir", dir, "-inferred", "ext:ruby:File:delete")
	if !strings.Contains(out, "bin/tool") {
		t.Errorf("affected output missing Ruby executable:\n%s", out)
	}
}

func TestCLICallersInferred(t *testing.T) {
	dir := testdata("cli-go")
	out := run(t, "callers", "-dir", dir, "Load")
	if strings.Contains(out, "Handler") || strings.Contains(out, "[inferred]") {
		t.Errorf("callers without -inferred should not show cross-package caller:\n%s", out)
	}
	if strings.Contains(out, "--contains") || strings.Contains(out, "--imports") {
		t.Errorf("callers output should not include non-call edges:\n%s", out)
	}
	out = run(t, "callers", "-dir", dir, "-inferred", "Load")
	if !strings.Contains(out, "Handler") {
		t.Errorf("callers -inferred should show cross-package caller:\n%s", out)
	}
}

func TestCLIUsage(t *testing.T) {
	cmd := exec.Command(binPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected non-zero exit for missing subcommand")
	}
	if !strings.Contains(string(out), "outline graph") {
		t.Errorf("usage not printed: %s", out)
	}
}

func TestCLIQuerySelection(t *testing.T) {
	cases := []struct {
		language string
		filename string
		source   string
	}{
		{"go", "app.go", "package app\nfunc entry(){middle()}\nfunc middle(){sink();sink();side()}\nfunc sink(){}\nfunc side(){}\n"},
		{"python", "app.py", "def entry():\n    middle()\ndef middle():\n    sink()\n    sink()\n    side()\ndef sink():\n    pass\ndef side():\n    pass\n"},
		{"ruby", "app.rb", "def entry\n  middle()\nend\ndef middle\n  sink()\n  sink()\n  side()\nend\ndef sink\nend\ndef side\nend\n"},
	}
	for _, c := range cases {
		t.Run(c.language, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, c.filename), []byte(c.source), 0o600); err != nil {
				t.Fatal(err)
			}
			graph := filepath.Join(t.TempDir(), "graph.json")
			run(t, "graph", "-o", graph, dir)
			queries := []struct {
				name  string
				args  []string
				nodes string
				edges string
			}{
				{"affected", []string{"affected", "-depth", "1", "sink"}, "middle,sink", "middle>sink"},
				{"path", []string{"path", "middle", "sink"}, "middle,sink", "middle>sink"},
				{"callers", []string{"callers", "sink"}, "middle,sink", "middle>sink,middle>sink"},
				{"callees", []string{"callees", "middle"}, "middle,side,sink", "middle>side,middle>sink,middle>sink"},
			}
			for _, q := range queries {
				t.Run(q.name, func(t *testing.T) {
					args := append([]string{q.args[0], "-g", graph}, q.args[1:]...)
					out := run(t, args...)
					assertQuerySelection(t, out, q.nodes, q.edges)
				})
			}
		})
	}
}

func assertQuerySelection(t *testing.T, out, wantNodes, wantEdges string) {
	t.Helper()
	var nodes, edges []string
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "NODE":
			nodes = append(nodes, fields[1])
		case "EDGE":
			edges = append(edges, fields[1]+">"+fields[3])
		}
	}
	slices.Sort(nodes)
	slices.Sort(edges)
	if strings.Join(nodes, ",") != wantNodes || strings.Join(edges, ",") != wantEdges {
		t.Fatalf("nodes=%v edges=%v, want nodes=%s edges=%s\n%s", nodes, edges, wantNodes, wantEdges, out)
	}
}
