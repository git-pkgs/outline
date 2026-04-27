package outline

import "testing"

func TestOutlineHCL(t *testing.T) {
	langCase{
		filename: "main.tf",
		src: `terraform {
  required_version = ">= 1.0"
}

variable "name" {
  type = string
}

resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}

module "vpc" {
  source = "./vpc"
}
`,
		want: []string{
			"terraform {",
			`variable "name" {`,
			`resource "aws_instance" "web" {`,
			`module "vpc" {`,
		},
		drop: []string{"required_version", "ami-123", "t2.micro", "./vpc"},
	}.run(t)
}

func TestOutlineStarlark(t *testing.T) {
	langCase{
		filename: "BUILD.bazel",
		src: `load("@rules_go//go:def.bzl", "go_library")

MAX = 100

def _impl(ctx):
    return [DefaultInfo()]

go_library(
    name = "lib",
    srcs = ["a.go"],
)
`,
		want: []string{
			`load("@rules_go//go:def.bzl", "go_library")`,
			"MAX = 100",
			"def _impl(ctx):",
			"go_library(",
		},
		drop: []string{"return [DefaultInfo()]", `srcs = ["a.go"]`},
	}.run(t)
}

func TestOutlineCMake(t *testing.T) {
	langCase{
		filename: "CMakeLists.txt",
		src: `cmake_minimum_required(VERSION 3.10)
project(app)

function(greet name)
    message("hi ${name}")
endfunction()

add_executable(app main.c)
`,
		want: []string{
			"cmake_minimum_required(VERSION 3.10)",
			"project(app)",
			"function(greet name)",
			"add_executable(app main.c)",
		},
		drop: []string{`message("hi`},
	}.run(t)
}

func TestOutlineBash(t *testing.T) {
	langCase{
		filename: "build.sh",
		src: `#!/bin/bash
set -e

VERSION=1.0

greet() {
    echo "hi $1"
}

main() {
    greet "world"
}
`,
		want: []string{
			"#!/bin/bash",
			"set -e",
			"VERSION=1.0",
			"greet() {",
			"main() {",
		},
		drop: []string{`echo "hi`, `greet "world"`},
	}.run(t)
}

func TestOutlineMake(t *testing.T) {
	langCase{
		filename: "Makefile",
		src: `CC = gcc

all: build test

build:
	$(CC) -o app main.c

test:
	./app --test
`,
		want: []string{
			"CC = gcc",
			"all: build test",
			"build:",
			"test:",
		},
		drop: []string{"$(CC) -o app", "./app --test"},
	}.run(t)
}

func TestDetectByName(t *testing.T) {
	cases := map[string]string{
		"Makefile":           "make",
		"path/to/Makefile":   "make",
		"CMakeLists.txt":     "cmake",
		"BUILD":              "starlark",
		"src/BUILD.bazel":    "starlark",
		"main.go":            "go",
		"README":             "",
	}
	for in, want := range cases {
		l, ok := detect(in)
		if want == "" {
			if ok {
				t.Errorf("detect(%q) = %s, want unsupported", in, l.name)
			}
			continue
		}
		if !ok || l.name != want {
			got := ""
			if l != nil {
				got = l.name
			}
			t.Errorf("detect(%q) = %q, want %q", in, got, want)
		}
	}
}
