package outline

import (
	"reflect"
	"testing"
)

func symbolsFor(t *testing.T, filename, src string) []Symbol {
	t.Helper()
	_, symbols, ok := outlineFile([]byte(src), filename)
	if !ok {
		t.Fatalf("%s: not supported", filename)
	}
	return symbols
}

func TestGoSymbols(t *testing.T) {
	src := `package sample

type Public struct{}
type private = int
type (
	Other int
	hidden string
)

const PublicConst = 1
const privateConst = 2
var PublicVar, privateVar = 1, 2

func PublicFunc() {
	var local int
}

func privateFunc() {}
func (Public) Method() {}
`
	want := []Symbol{
		{Name: "Public", Kind: "type", Line: 3, Exported: true},
		{Name: "private", Kind: "type", Line: 4},
		{Name: "Other", Kind: "type", Line: 6, Exported: true},
		{Name: "hidden", Kind: "type", Line: 7},
		{Name: "PublicConst", Kind: "const", Line: 10, Exported: true},
		{Name: "privateConst", Kind: "const", Line: 11},
		{Name: "PublicVar", Kind: "var", Line: 12, Exported: true},
		{Name: "privateVar", Kind: "var", Line: 12},
		{Name: "PublicFunc", Kind: "func", Line: 14, Exported: true},
		{Name: "privateFunc", Kind: "func", Line: 18},
		{Name: "Method", Kind: "func", Line: 19, Exported: true},
	}
	if got := symbolsFor(t, "sample.go", src); !reflect.DeepEqual(got, want) {
		t.Errorf("symbols = %#v, want %#v", got, want)
	}
}

func TestJavascriptSymbols(t *testing.T) {
	src := `export class PublicClass {}
class LocalClass {}
export function publicFunc() {}
function localFunc() {}
export const direct = () => {};
const named = () => {};
export { named };
const common = () => {};
module.exports.common = common;
const objectValue = () => {};
module.exports = { objectValue };
class Container { method() {} }
`
	want := []Symbol{
		{Name: "PublicClass", Kind: "class", Line: 1, Exported: true},
		{Name: "LocalClass", Kind: "class", Line: 2},
		{Name: "publicFunc", Kind: "func", Line: 3, Exported: true},
		{Name: "localFunc", Kind: "func", Line: 4},
		{Name: "direct", Kind: "const", Line: 5, Exported: true},
		{Name: "named", Kind: "const", Line: 6, Exported: true},
		{Name: "common", Kind: "const", Line: 8, Exported: true},
		{Name: "objectValue", Kind: "const", Line: 10, Exported: true},
		{Name: "Container", Kind: "class", Line: 12},
	}
	if got := symbolsFor(t, "sample.js", src); !reflect.DeepEqual(got, want) {
		t.Errorf("symbols = %#v, want %#v", got, want)
	}
}

func TestPythonSymbols(t *testing.T) {
	src := `class Public:
    field = 1

class _Private:
    pass

def public_func():
    def nested():
        pass

def _private_func():
    pass

PUBLIC_VALUE = 1
_private_value = 2
`
	want := []Symbol{
		{Name: "Public", Kind: "class", Line: 1, Exported: true},
		{Name: "_Private", Kind: "class", Line: 4},
		{Name: "public_func", Kind: "func", Line: 7, Exported: true},
		{Name: "_private_func", Kind: "func", Line: 11},
		{Name: "PUBLIC_VALUE", Kind: "var", Line: 14, Exported: true},
		{Name: "_private_value", Kind: "var", Line: 15},
	}
	if got := symbolsFor(t, "sample.py", src); !reflect.DeepEqual(got, want) {
		t.Errorf("symbols = %#v, want %#v", got, want)
	}
}

func TestTypescriptSymbols(t *testing.T) {
	src := `export interface PublicType {}
type LocalType = string;
enum LocalEnum { Value }
declare function declared(): void;
export abstract class PublicClass {}
export const make = () => {};
`
	want := []Symbol{
		{Name: "PublicType", Kind: "type", Line: 1, Exported: true},
		{Name: "LocalType", Kind: "type", Line: 2},
		{Name: "LocalEnum", Kind: "type", Line: 3},
		{Name: "declared", Kind: "func", Line: 4},
		{Name: "PublicClass", Kind: "class", Line: 5, Exported: true},
		{Name: "make", Kind: "const", Line: 6, Exported: true},
	}
	if got := symbolsFor(t, "sample.ts", src); !reflect.DeepEqual(got, want) {
		t.Errorf("symbols = %#v, want %#v", got, want)
	}
}

func TestLanguageSymbolsCommon(t *testing.T) {
	cases := []struct {
		filename string
		src      string
		want     []Symbol
	}{
		{
			filename: "sample.c",
			src:      "#define MAX 1\nstruct User {};\ntypedef int ID;\nint value;\nstatic int hidden;\nint run(void) {}\n",
			want: []Symbol{
				{Name: "MAX", Kind: "const", Line: 1, Exported: true},
				{Name: "User", Kind: "type", Line: 2, Exported: true},
				{Name: "ID", Kind: "type", Line: 3, Exported: true},
				{Name: "value", Kind: "var", Line: 4, Exported: true},
				{Name: "hidden", Kind: "var", Line: 5},
				{Name: "run", Kind: "func", Line: 6, Exported: true},
			},
		},
		{
			filename: "sample.cpp",
			src:      "namespace app {}\nclass Widget {};\nstruct Value {};\nusing ID = int;\nint count;\nint run() {}\n",
			want: []Symbol{
				{Name: "app", Kind: "type", Line: 1, Exported: true},
				{Name: "Widget", Kind: "class", Line: 2, Exported: true},
				{Name: "Value", Kind: "type", Line: 3, Exported: true},
				{Name: "ID", Kind: "type", Line: 4, Exported: true},
				{Name: "count", Kind: "var", Line: 5, Exported: true},
				{Name: "run", Kind: "func", Line: 6, Exported: true},
			},
		},
		{
			filename: "Sample.cs",
			src:      "public class User {}\ninternal interface Hidden {}\npublic struct Value {}\npublic enum Color { Red }\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Hidden", Kind: "type", Line: 2},
				{Name: "Value", Kind: "type", Line: 3, Exported: true},
				{Name: "Color", Kind: "type", Line: 4, Exported: true},
			},
		},
		{
			filename: "Sample.java",
			src:      "public class User {}\ninterface Hidden {}\nenum Color { RED }\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Hidden", Kind: "type", Line: 2},
				{Name: "Color", Kind: "type", Line: 3},
			},
		},
		{
			filename: "sample.php",
			src:      "<?php\nclass User {}\ninterface Greeter {}\nfunction helper() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 2, Exported: true},
				{Name: "Greeter", Kind: "type", Line: 3, Exported: true},
				{Name: "helper", Kind: "func", Line: 4, Exported: true},
			},
		},
		{
			filename: "sample.rs",
			src:      "pub struct User;\nconst HIDDEN: usize = 1;\npub fn run() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "type", Line: 1, Exported: true},
				{Name: "HIDDEN", Kind: "const", Line: 2},
				{Name: "run", Kind: "func", Line: 3, Exported: true},
			},
		},
		{
			filename: "Sample.scala",
			src:      "class User\ntrait Greeter\nobject Main\ntype ID = String\nval count = 1\nvar mutable = 2\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Greeter", Kind: "type", Line: 2, Exported: true},
				{Name: "Main", Kind: "type", Line: 3, Exported: true},
				{Name: "ID", Kind: "type", Line: 4, Exported: true},
				{Name: "count", Kind: "const", Line: 5, Exported: true},
				{Name: "mutable", Kind: "var", Line: 6, Exported: true},
			},
		},
		{
			filename: "sample.dart",
			src:      "class User {}\nmixin Shared {}\nenum Color { red }\ntypedef ID = String;\nvoid main() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Shared", Kind: "type", Line: 2, Exported: true},
				{Name: "Color", Kind: "type", Line: 3, Exported: true},
				{Name: "ID", Kind: "type", Line: 4, Exported: true},
				{Name: "main", Kind: "func", Line: 5, Exported: true},
			},
		},
		{
			filename: "sample.d",
			src:      "struct User {}\nclass Service {}\nalias ID = int;\nvoid run() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "type", Line: 1, Exported: true},
				{Name: "Service", Kind: "class", Line: 2, Exported: true},
				{Name: "ID", Kind: "type", Line: 3, Exported: true},
				{Name: "run", Kind: "func", Line: 4, Exported: true},
			},
		},
		{
			filename: "Sample.groovy",
			src:      "class User {}\ndef main() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "main", Kind: "func", Line: 2, Exported: true},
			},
		},
		{
			filename: "sample.sh",
			src:      "VERSION=1\ngreet() { :; }\n",
			want: []Symbol{
				{Name: "VERSION", Kind: "var", Line: 1, Exported: true},
				{Name: "greet", Kind: "func", Line: 2, Exported: true},
			},
		},
		{
			filename: "CMakeLists.txt",
			src:      "function(greet name)\nendfunction()\nmacro(build target)\nendmacro()\n",
			want: []Symbol{
				{Name: "greet", Kind: "func", Line: 1, Exported: true},
				{Name: "build", Kind: "func", Line: 3, Exported: true},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			if got := symbolsFor(t, tc.filename, tc.src); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("symbols = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestLanguageSymbolsAdditional(t *testing.T) {
	cases := []struct {
		filename string
		src      string
		want     []Symbol
	}{
		{
			filename: "sample.cr",
			src:      "class User\nend\nmodule Tools\nend\nstruct Value\nend\nenum Color\n Red\nend\nMAX = 1\nalias ID = Int32\ndef run\nend\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Tools", Kind: "type", Line: 3, Exported: true},
				{Name: "Value", Kind: "type", Line: 5, Exported: true},
				{Name: "Color", Kind: "type", Line: 7, Exported: true},
				{Name: "MAX", Kind: "const", Line: 10, Exported: true},
				{Name: "ID", Kind: "type", Line: 11, Exported: true},
				{Name: "run", Kind: "func", Line: 12, Exported: true},
			},
		},
		{
			filename: "sample.rb",
			src:      "class User\nend\nmodule Tools\nend\nMAX = 1\ndef run\nend\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Tools", Kind: "type", Line: 3, Exported: true},
				{Name: "MAX", Kind: "const", Line: 5, Exported: true},
				{Name: "run", Kind: "func", Line: 6, Exported: true},
			},
		},
		{
			filename: "sample.kt",
			src:      "class User\ninterface Greeter\ntypealias ID = String\nval count = 1\nvar mutable = 2\nfun run() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Greeter", Kind: "type", Line: 2, Exported: true},
				{Name: "ID", Kind: "type", Line: 3, Exported: true},
				{Name: "count", Kind: "const", Line: 4, Exported: true},
				{Name: "mutable", Kind: "var", Line: 5, Exported: true},
				{Name: "run", Kind: "func", Line: 6, Exported: true},
			},
		},
		{
			filename: "sample.swift",
			src:      "public class User {}\npublic struct Value {}\npublic enum Color { case red }\npublic protocol Greeter {}\npublic typealias ID = String\npublic let count = 1\nvar mutable = 2\npublic func run() {}\n",
			want: []Symbol{
				{Name: "User", Kind: "class", Line: 1, Exported: true},
				{Name: "Value", Kind: "type", Line: 2, Exported: true},
				{Name: "Color", Kind: "type", Line: 3, Exported: true},
				{Name: "Greeter", Kind: "type", Line: 4, Exported: true},
				{Name: "ID", Kind: "type", Line: 5, Exported: true},
				{Name: "count", Kind: "const", Line: 6, Exported: true},
				{Name: "mutable", Kind: "var", Line: 7},
				{Name: "run", Kind: "func", Line: 8, Exported: true},
			},
		},
		{
			filename: "sample.zig",
			src:      "pub const User = struct {};\nconst count = 1;\nvar mutable: u8 = 2;\npub fn run() void {}\n",
			want: []Symbol{
				{Name: "User", Kind: "type", Line: 1, Exported: true},
				{Name: "count", Kind: "const", Line: 2},
				{Name: "mutable", Kind: "var", Line: 3},
				{Name: "run", Kind: "func", Line: 4, Exported: true},
			},
		},
		{
			filename: "sample.nim",
			src:      "type\n  User* = object\nconst MAX* = 1\nvar mutable = 2\nlet fixed = 3\nproc run*() = discard\n",
			want: []Symbol{
				{Name: "User", Kind: "type", Line: 2, Exported: true},
				{Name: "MAX", Kind: "const", Line: 3, Exported: true},
				{Name: "mutable", Kind: "var", Line: 4},
				{Name: "fixed", Kind: "const", Line: 5},
				{Name: "run", Kind: "func", Line: 6, Exported: true},
			},
		},
		{
			filename: "sample.ex",
			src:      "defmodule App do\nend\ndef run(), do: :ok\ndefp hidden(), do: :ok\n",
			want: []Symbol{
				{Name: "App", Kind: "type", Line: 1, Exported: true},
				{Name: "run", Kind: "func", Line: 3, Exported: true},
				{Name: "hidden", Kind: "func", Line: 4},
			},
		},
		{
			filename: "sample.erl",
			src:      "-module(sample).\n-export([run/0]).\n-record(user, {name}).\n-type id() :: integer().\nrun() -> ok.\nhidden() -> ok.\n",
			want: []Symbol{
				{Name: "user", Kind: "type", Line: 3},
				{Name: "id", Kind: "type", Line: 4},
				{Name: "run", Kind: "func", Line: 5, Exported: true},
				{Name: "hidden", Kind: "func", Line: 6},
			},
		},
		{
			filename: "sample.hs",
			src:      "module Sample where\ndata User = User\ntype ID = Int\nrun x = pure x\nvalue = 1\n",
			want: []Symbol{
				{Name: "User", Kind: "type", Line: 2, Exported: true},
				{Name: "ID", Kind: "type", Line: 3, Exported: true},
				{Name: "run", Kind: "func", Line: 4, Exported: true},
				{Name: "value", Kind: "var", Line: 5, Exported: true},
			},
		},
		{
			filename: "sample.clj",
			src:      "(def MAX 1)\n(defn run [] 1)\n(defn- hidden [] 2)\n(defrecord User [])\n",
			want: []Symbol{
				{Name: "MAX", Kind: "var", Line: 1, Exported: true},
				{Name: "run", Kind: "func", Line: 2, Exported: true},
				{Name: "hidden", Kind: "func", Line: 3},
				{Name: "User", Kind: "type", Line: 4, Exported: true},
			},
		},
		{
			filename: "sample.pm",
			src:      "package Sample;\nsub run {}\n",
			want:     []Symbol{{Name: "run", Kind: "func", Line: 2, Exported: true}},
		},
		{
			filename: "sample.lua",
			src:      "local value = 1\nfunction run() end\nlocal function hidden() end\n",
			want: []Symbol{
				{Name: "value", Kind: "var", Line: 1, Exported: true},
				{Name: "run", Kind: "func", Line: 2, Exported: true},
				{Name: "hidden", Kind: "func", Line: 3, Exported: true},
			},
		},
		{
			filename: "sample.R",
			src:      "run <- function() {}\nvalue <- 1\n",
			want: []Symbol{
				{Name: "run", Kind: "func", Line: 1, Exported: true},
				{Name: "value", Kind: "var", Line: 2, Exported: true},
			},
		},
		{
			filename: "sample.jl",
			src:      "module Sample\nend\nexport User, run, MAX\nstruct User\nend\nabstract type Shape end\nprimitive type Byte 8 end\nconst MAX = 1\nfunction run()\nend\n",
			want: []Symbol{
				{Name: "Sample", Kind: "type", Line: 1},
				{Name: "User", Kind: "type", Line: 4, Exported: true},
				{Name: "Shape", Kind: "type", Line: 6},
				{Name: "Byte", Kind: "type", Line: 7},
				{Name: "MAX", Kind: "const", Line: 8, Exported: true},
				{Name: "run", Kind: "func", Line: 9, Exported: true},
			},
		},
		{
			filename: "sample.ml",
			src:      "module Sample = struct end\ntype user = { name: string }\nclass thing = object end\nlet run x = x\nlet value = 1\n",
			want: []Symbol{
				{Name: "Sample", Kind: "type", Line: 1, Exported: true},
				{Name: "user", Kind: "type", Line: 2, Exported: true},
				{Name: "thing", Kind: "class", Line: 3, Exported: true},
				{Name: "run", Kind: "func", Line: 4, Exported: true},
				{Name: "value", Kind: "var", Line: 5, Exported: true},
			},
		},
		{
			filename: "sample.fs",
			src:      "module Sample\ntype User = { Name: string }\nlet run x = x\nlet value = 1\n",
			want: []Symbol{
				{Name: "Sample", Kind: "type", Line: 1, Exported: true},
				{Name: "User", Kind: "type", Line: 2, Exported: true},
				{Name: "run", Kind: "func", Line: 3, Exported: true},
				{Name: "value", Kind: "var", Line: 4, Exported: true},
			},
		},
		{
			filename: "main.tf",
			src:      "variable \"name\" {}\nresource \"thing\" \"web\" {}\nmodule \"vpc\" {}\n",
			want: []Symbol{
				{Name: "name", Kind: "var", Line: 1, Exported: true},
				{Name: "web", Kind: "type", Line: 2, Exported: true},
				{Name: "vpc", Kind: "type", Line: 3, Exported: true},
			},
		},
		{
			filename: "BUILD.bazel",
			src:      "MAX = 1\ndef run():\n    pass\n",
			want: []Symbol{
				{Name: "MAX", Kind: "var", Line: 1, Exported: true},
				{Name: "run", Kind: "func", Line: 2, Exported: true},
			},
		},
		{
			filename: "Makefile",
			src:      "VALUE = 1\nall: build\nbuild:\n\t@true\n",
			want: []Symbol{
				{Name: "VALUE", Kind: "var", Line: 1, Exported: true},
				{Name: "all", Kind: "func", Line: 2, Exported: true},
				{Name: "build", Kind: "func", Line: 3, Exported: true},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			if got := symbolsFor(t, tc.filename, tc.src); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("symbols = %#v, want %#v", got, tc.want)
			}
		})
	}
}
