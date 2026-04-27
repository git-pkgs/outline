package outline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "main.go", goFixture)
	writeFile(t, root, "lib/user.rb", rubyFixture)
	writeFile(t, root, "README.md", "# hello\n")
	writeFile(t, root, ".gitignore", "secret.txt\n")
	writeFile(t, root, "secret.txt", "password")
	writeFile(t, root, "node_modules/left-pad/index.js", "module.exports = pad")
	writeFile(t, root, "vendor/thing.go", "package thing")
	writeFile(t, root, "logo.png", "\x89PNG\r\n\x1a\n\x00\x00")
	writeFile(t, root, "huge.txt", strings.Repeat("x", 2000))
	return root
}

func fileMap(r *Result) map[string]File {
	m := make(map[string]File)
	for _, f := range r.Files {
		m[f.Path] = f
	}
	return m
}

func TestPack(t *testing.T) {
	root := setupRepo(t)

	r, err := Pack(root, Options{Compress: true, MaxFileSize: 1000})
	if err != nil {
		t.Fatal(err)
	}
	files := fileMap(r)

	for _, p := range []string{"secret.txt", "node_modules/left-pad/index.js", "vendor/thing.go"} {
		if _, ok := files[p]; ok {
			t.Errorf("should be ignored: %s", p)
		}
	}

	if f, ok := files["main.go"]; !ok {
		t.Error("main.go missing")
	} else {
		if !f.Outlined {
			t.Error("main.go should be outlined")
		}
		if !strings.Contains(f.Content, "func SayHello") {
			t.Error("main.go outline missing signature")
		}
		if strings.Contains(f.Content, "fmt.Printf") {
			t.Error("main.go outline leaked body")
		}
	}

	if f, ok := files["lib/user.rb"]; !ok || !f.Outlined {
		t.Error("lib/user.rb should be outlined")
	}

	if f, ok := files["README.md"]; !ok {
		t.Error("README.md missing")
	} else if f.Outlined {
		t.Error("README.md should not be outlined")
	} else if f.Content != "# hello\n" {
		t.Errorf("README.md content = %q", f.Content)
	}

	if f := files["logo.png"]; f.Skipped != "binary" {
		t.Errorf("logo.png skip = %q, want binary", f.Skipped)
	}
	if f := files["huge.txt"]; f.Skipped != "too-large" {
		t.Errorf("huge.txt skip = %q, want too-large", f.Skipped)
	}

	if !strings.Contains(r.Tree, "lib/") || !strings.Contains(r.Tree, "main.go") {
		t.Errorf("tree missing entries:\n%s", r.Tree)
	}
	if strings.Contains(r.Tree, "node_modules") {
		t.Error("tree should not contain ignored dirs")
	}
}

func TestPackNoCompress(t *testing.T) {
	root := setupRepo(t)

	r, err := Pack(root, Options{Compress: false})
	if err != nil {
		t.Fatal(err)
	}
	files := fileMap(r)

	f := files["main.go"]
	if f.Outlined {
		t.Error("should not be outlined when Compress=false")
	}
	if !strings.Contains(f.Content, "fmt.Printf") {
		t.Error("full content should include body")
	}
}

func TestPackMaxFiles(t *testing.T) {
	root := t.TempDir()
	for i := range 5 {
		writeFile(t, root, string(rune('a'+i))+".txt", "x")
	}
	r, err := Pack(root, Options{MaxFiles: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Files) != 3 {
		t.Errorf("got %d files, want 3", len(r.Files))
	}
	if !r.Truncated {
		t.Error("should be truncated")
	}
}

func TestPackUserIgnore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "keep.go", "package main")
	writeFile(t, root, "drop.go", "package main")

	r, err := Pack(root, Options{Ignore: []string{"drop.go"}})
	if err != nil {
		t.Fatal(err)
	}
	files := fileMap(r)
	if _, ok := files["drop.go"]; ok {
		t.Error("drop.go should be ignored")
	}
	if _, ok := files["keep.go"]; !ok {
		t.Error("keep.go missing")
	}
}

func TestIsBinary(t *testing.T) {
	cases := []struct {
		data []byte
		want bool
	}{
		{[]byte("hello world"), false},
		{[]byte("hello\x00world"), true},
		{[]byte{0xff, 0xfe, 0x00}, true},
		{[]byte("日本語"), false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isBinary(c.data); got != c.want {
			t.Errorf("isBinary(%q) = %v, want %v", c.data, got, c.want)
		}
	}
}
