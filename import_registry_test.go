package outline

import (
	"reflect"
	"testing"
)

func TestRegistryLanguageImports(t *testing.T) {
	t.Parallel()
	tests := []struct {
		filename string
		src      string
		want     []Import
	}{
		{
			filename: "app.dart",
			src: `import 'package:http/http.dart' as http;
import 'package:collection/collection.dart' show DeepCollectionEquality, IterableExtension;
import 'package:foo/foo.dart' hide Internal;
import 'package:lazy/lazy.dart' deferred as lazy;
`,
			want: []Import{
				{Module: "package:http/http.dart", Kind: ImportNamespace, Names: []Name{{Alias: "http"}}, Line: 1},
				{
					Module: "package:collection/collection.dart",
					Kind:   ImportNamed,
					Names: []Name{
						{Name: "DeepCollectionEquality"},
						{Name: "IterableExtension"},
					},
					Line: 2,
				},
				{Module: "package:foo/foo.dart", Kind: ImportWildcard, Line: 3},
				{Module: "package:lazy/lazy.dart", Kind: ImportNamespace, Names: []Name{{Alias: "lazy"}}, Line: 4},
			},
		},
		{
			filename: "App.swift",
			src: `import Alamofire
@testable import XCTest
import struct Foundation.Date
import func Darwin.sqrt
`,
			want: []Import{
				{Module: "Alamofire", Kind: ImportModule, Names: []Name{{Alias: "Alamofire"}}, Line: 1},
				{Module: "XCTest", Kind: ImportModule, Names: []Name{{Alias: "XCTest"}}, Line: 2},
				{Module: "Foundation", Kind: ImportNamed, Names: []Name{{Name: "Date"}}, Line: 3},
				{Module: "Darwin", Kind: ImportNamed, Names: []Name{{Name: "sqrt"}}, Line: 4},
			},
		},
		{
			filename: "App.hs",
			src: `module App where

import Data.List
import qualified Data.Text as T
import Data.Map (Map, lookup)
import Data.Set hiding (map)
`,
			want: []Import{
				{Module: "Data.List", Kind: ImportWildcard, Line: 3},
				{Module: "Data.Text", Kind: ImportNamespace, Names: []Name{{Alias: "T"}}, Line: 4},
				{Module: "Data.Map", Kind: ImportNamed, Names: []Name{{Name: "Map"}, {Name: "lookup"}}, Line: 5},
				{Module: "Data.Set", Kind: ImportWildcard, Line: 6},
			},
		},
		{
			filename: "App.pm",
			src: `use JSON::MaybeXS qw(encode_json decode_json);
use Mojo::UserAgent;
use Foo::Bar ();
require HTTP::Tiny;
require "Path/Tiny.pm";
`,
			want: []Import{
				{
					Module: "JSON::MaybeXS",
					Kind:   ImportNamed,
					Names:  []Name{{Name: "encode_json"}, {Name: "decode_json"}},
					Line:   1,
				},
				{Module: "Mojo::UserAgent", Kind: ImportWildcard, Line: 2},
				{Module: "Foo::Bar", Kind: ImportSideEffect, Line: 3},
				{Module: "HTTP::Tiny", Kind: ImportSideEffect, Line: 4},
				{Module: "Path/Tiny.pm", Kind: ImportSideEffect, Line: 5},
			},
		},
		{
			filename: "app.lua",
			src: `local json = require("cjson")
local inspect = require "inspect"
require("side_effect")
`,
			want: []Import{
				{Module: "cjson", Kind: ImportModule, Names: []Name{{Alias: "json"}}, Line: 1},
				{Module: "inspect", Kind: ImportModule, Names: []Name{{Alias: "inspect"}}, Line: 2},
				{Module: "side_effect", Kind: ImportSideEffect, Line: 3},
			},
		},
		{
			filename: "app.R",
			src: `library(dplyr)
require("ggplot2")
requireNamespace("jsonlite")
x <- dplyr::filter(data, value > 1)
y <- jsonlite:::simplify
`,
			want: []Import{
				{Module: "dplyr", Kind: ImportWildcard, Line: 1},
				{Module: "ggplot2", Kind: ImportWildcard, Line: 2},
				{Module: "jsonlite", Kind: ImportModule, Names: []Name{{Alias: "jsonlite"}}, Line: 3},
				{Module: "dplyr", Kind: ImportNamed, Names: []Name{{Name: "filter"}}, Line: 4},
				{Module: "jsonlite", Kind: ImportNamed, Names: []Name{{Name: "simplify"}}, Line: 5},
			},
		},
		{
			filename: "app.jl",
			src: `using DataFrames
using CSV: File, Rows
import JSON
import HTTP: get as fetch
import StatsBase as Stats
`,
			want: []Import{
				{Module: "DataFrames", Kind: ImportModule, Names: []Name{{Alias: "DataFrames"}}, Line: 1},
				{Module: "CSV", Kind: ImportNamed, Names: []Name{{Name: "File"}, {Name: "Rows"}}, Line: 2},
				{Module: "JSON", Kind: ImportModule, Names: []Name{{Alias: "JSON"}}, Line: 3},
				{Module: "HTTP", Kind: ImportNamed, Names: []Name{{Name: "get", Alias: "fetch"}}, Line: 4},
				{Module: "StatsBase", Kind: ImportModule, Names: []Name{{Alias: "Stats"}}, Line: 5},
			},
		},
		{
			filename: "app.ml",
			src: `open Core
open! Base
include Foo
module J = Yojson.Safe
`,
			want: []Import{
				{Module: "Core", Kind: ImportWildcard, Line: 1},
				{Module: "Base", Kind: ImportWildcard, Line: 2},
				{Module: "Foo", Kind: ImportWildcard, Line: 3},
				{Module: "Yojson.Safe", Kind: ImportModule, Names: []Name{{Alias: "J"}}, Line: 4},
			},
		},
		{
			filename: "app.cr",
			src:      "require \"json\"\nrequire \"http/client\"\nrequire \"./local\"\n",
			want: []Import{
				{Module: "json", Kind: ImportSideEffect, Line: 1},
				{Module: "http/client", Kind: ImportSideEffect, Line: 2},
				{Module: "./local", Kind: ImportSideEffect, Line: 3},
			},
		},
		{
			filename: "app.nim",
			src: `import strutils
import std/[sequtils, tables]
import chronicles as log
from json import parseJson, JsonNode
`,
			want: []Import{
				{Module: "strutils", Kind: ImportModule, Names: []Name{{Alias: "strutils"}}, Line: 1},
				{Module: "std/sequtils", Kind: ImportModule, Names: []Name{{Alias: "sequtils"}}, Line: 2},
				{Module: "std/tables", Kind: ImportModule, Names: []Name{{Alias: "tables"}}, Line: 2},
				{Module: "chronicles", Kind: ImportModule, Names: []Name{{Alias: "log"}}, Line: 3},
				{Module: "json", Kind: ImportNamed, Names: []Name{{Name: "parseJson"}, {Name: "JsonNode"}}, Line: 4},
			},
		},
		{
			filename: "app.zig",
			src: `const std = @import("std");
const clap = @import("clap");
_ = @import("side_effect");
`,
			want: []Import{
				{Module: "std", Kind: ImportModule, Names: []Name{{Alias: "std"}}, Line: 1},
				{Module: "clap", Kind: ImportModule, Names: []Name{{Alias: "clap"}}, Line: 2},
				{Module: "side_effect", Kind: ImportSideEffect, Line: 3},
			},
		},
		{
			filename: "app.d",
			src: `import std.stdio;
import io = std.file;
import std.algorithm : map, mapped = filter;
static import vibe.data.json;
`,
			want: []Import{
				{Module: "std.stdio", Kind: ImportWildcard, Line: 1},
				{Module: "std.file", Kind: ImportModule, Names: []Name{{Alias: "io"}}, Line: 2},
				{
					Module: "std.algorithm",
					Kind:   ImportNamed,
					Names:  []Name{{Name: "map"}, {Name: "filter", Alias: "mapped"}},
					Line:   3,
				},
				{Module: "vibe.data.json", Kind: ImportModule, Names: []Name{{Alias: "vibe.data.json"}}, Line: 4},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.filename, func(t *testing.T) {
			t.Parallel()
			got, ok := Imports([]byte(test.src), test.filename)
			if !ok {
				t.Fatal("Imports() supported = false")
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Imports() = %#v, want %#v", got, test.want)
			}
		})
	}
}
