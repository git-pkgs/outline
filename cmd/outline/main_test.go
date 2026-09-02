package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
