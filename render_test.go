package outline

import (
	"strings"
	"testing"
)

func TestMarkdown(t *testing.T) {
	r := &Result{
		Tree: "└── main.go\n",
		Files: []File{
			{Path: "main.go", Content: "package main\n", Language: "go"},
			{Path: "logo.png", Skipped: "binary", Size: 1024},
		},
	}
	var b strings.Builder
	if err := r.Markdown(&b); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	t.Log(got)

	for _, want := range []string{
		"## Structure",
		"└── main.go",
		"### main.go",
		"```go\npackage main\n```",
		"### logo.png",
		"_(skipped: binary, 1024 bytes)_",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing: %q", want)
		}
	}
}

func TestMarkdownFenceGrows(t *testing.T) {
	r := &Result{
		Files: []File{{Path: "x.md", Content: "````\ncode\n````"}},
	}
	var b strings.Builder
	if err := r.Markdown(&b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "`````markdown") {
		t.Errorf("fence should grow past content backticks:\n%s", b.String())
	}
}

func TestXML(t *testing.T) {
	r := &Result{
		Tree: "└── a.go\n",
		Files: []File{
			{Path: "a.go", Content: "package a"},
			{Path: "b.bin", Skipped: "binary", Size: 10},
		},
	}
	var b strings.Builder
	if err := r.XML(&b); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	for _, want := range []string{
		"<directory_structure>",
		`<file path="a.go">`,
		"package a",
		"</file>",
		`<file path="b.bin" skipped="binary" size="10"/>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing: %q", want)
		}
	}
}

func TestMarkdownFenceClamped(t *testing.T) {
	r := &Result{
		Files: []File{{Path: "x.md", Content: strings.Repeat("`", 1000) + "\ncode"}},
	}
	var b strings.Builder
	if err := r.Markdown(&b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	fence := strings.Repeat("`", 17)
	// Check fence lines (fence+lang and closing fence), not content lines.
	var foundOpen, foundClose bool
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, fence+"markdown") {
			foundOpen = true
			if strings.HasPrefix(line, fence+"`") {
				t.Error("opening fence is longer than 17 backticks")
			}
		}
		if line == fence {
			foundClose = true
		}
	}
	if !foundOpen {
		t.Error("fence should be exactly 17 backticks (16 max + 1) for opening")
	}
	if !foundClose {
		t.Error("fence should be exactly 17 backticks (16 max + 1) for closing")
	}
}

func TestPackMarkdownEndToEnd(t *testing.T) {
	root := setupRepo(t)
	r, err := Pack(root, Options{Compress: true, MaxFileSize: 1000})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := r.Markdown(&b); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "func SayHello") {
		t.Error("rendered output missing outlined go signature")
	}
	if strings.Contains(got, "fmt.Printf") {
		t.Error("rendered output leaked function body")
	}
}
