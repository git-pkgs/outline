package outline

import (
	"reflect"
	"testing"
)

func TestPythonImports(t *testing.T) {
	t.Parallel()
	src := []byte(`import flask
import yaml as y
import flask.json
from werkzeug.http import (
    parse_authorization_header as parse,
    dump_header,
)
from flask import *
`)
	want := []Import{
		{Module: "flask", Kind: ImportModule, Names: []Name{{Alias: "flask"}}, Line: 1},
		{Module: "yaml", Kind: ImportModule, Names: []Name{{Alias: "y"}}, Line: 2},
		{Module: "flask.json", Kind: ImportModule, Names: []Name{{Alias: "flask"}}, Line: 3},
		{
			Module: "werkzeug.http",
			Kind:   ImportNamed,
			Names: []Name{
				{Name: "parse_authorization_header", Alias: "parse"},
				{Name: "dump_header"},
			},
			Line: 4,
		},
		{Module: "flask", Kind: ImportWildcard, Line: 8},
	}

	got, ok := Imports(src, "app.py")
	if !ok {
		t.Fatal("Imports() supported = false")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Imports() = %#v, want %#v", got, want)
	}
}

func TestPythonFutureAndRelativeImports(t *testing.T) {
	t.Parallel()
	src := []byte(`from __future__ import annotations
from .local import thing
from ..pkg.mod import value as local
`)
	want := []Import{
		{Module: "__future__", Kind: ImportNamed, Names: []Name{{Name: "annotations"}}, Line: 1},
		{Module: ".local", Kind: ImportNamed, Names: []Name{{Name: "thing"}}, Line: 2},
		{Module: "..pkg.mod", Kind: ImportNamed, Names: []Name{{Name: "value", Alias: "local"}}, Line: 3},
	}

	got, ok := Imports(src, "app.py")
	if !ok {
		t.Fatal("Imports() supported = false")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Imports() = %#v, want %#v", got, want)
	}
}

func TestJavascriptImports(t *testing.T) {
	t.Parallel()
	src := []byte(`import ws from "ws";
import { Server, WebSocket as Socket } from "ws";
import * as WS from "ws";
import "side-effect";
const { OPEN, Server: WSServer } = require("ws");
const Receiver = require("dep").Server;
const direct = require("dep2");
require("dep3");
const wrapped = consume(require("dep4"));
`)
	want := []Import{
		{Module: "ws", Kind: ImportDefault, Names: []Name{{Alias: "ws"}}, Line: 1},
		{
			Module: "ws",
			Kind:   ImportNamed,
			Names: []Name{
				{Name: "Server"},
				{Name: "WebSocket", Alias: "Socket"},
			},
			Line: 2,
		},
		{Module: "ws", Kind: ImportNamespace, Names: []Name{{Alias: "WS"}}, Line: 3},
		{Module: "side-effect", Kind: ImportSideEffect, Line: 4},
		{
			Module: "ws",
			Kind:   ImportNamed,
			Names: []Name{
				{Name: "OPEN"},
				{Name: "Server", Alias: "WSServer"},
			},
			Line: 5,
		},
		{Module: "dep", Kind: ImportNamed, Names: []Name{{Name: "Server", Alias: "Receiver"}}, Line: 6},
		{Module: "dep2", Kind: ImportModule, Names: []Name{{Alias: "direct"}}, Line: 7},
		{Module: "dep3", Kind: ImportSideEffect, Line: 8},
		{Module: "dep4", Kind: ImportSideEffect, Line: 9},
	}

	got, ok := Imports(src, "app.js")
	if !ok {
		t.Fatal("Imports() supported = false")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Imports() = %#v, want %#v", got, want)
	}
}

func TestTypescriptImports(t *testing.T) {
	t.Parallel()
	src := []byte(`import ws, { Server as WSServer } from "ws";
import type { WebSocket } from "ws";
import * as WS from "ws";
`)
	want := []Import{
		{Module: "ws", Kind: ImportDefault, Names: []Name{{Alias: "ws"}}, Line: 1},
		{Module: "ws", Kind: ImportNamed, Names: []Name{{Name: "Server", Alias: "WSServer"}}, Line: 1},
		{Module: "ws", Kind: ImportNamed, Names: []Name{{Name: "WebSocket"}}, Line: 2},
		{Module: "ws", Kind: ImportNamespace, Names: []Name{{Alias: "WS"}}, Line: 3},
	}

	for _, filename := range []string{"app.ts", "app.tsx"} {
		got, ok := Imports(src, filename)
		if !ok {
			t.Fatalf("Imports(%q) supported = false", filename)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Imports(%q) = %#v, want %#v", filename, got, want)
		}
	}
}

func TestImportsSupportResult(t *testing.T) {
	t.Parallel()
	if got, ok := Imports([]byte("value = 1\n"), "app.py"); !ok || len(got) != 0 {
		t.Fatalf("supported empty Imports() = %#v, %v", got, ok)
	}
	if got, ok := Imports([]byte("value = 1\n"), "app.txt"); ok || got != nil {
		t.Fatalf("unsupported Imports() = %#v, %v", got, ok)
	}
}

func TestHyrumLanguageImports(t *testing.T) {
	t.Parallel()
	tests := []struct {
		filename string
		src      string
		want     []Import
	}{
		{
			filename: "main.go",
			src: `package main
import (
	"github.com/x/y"
	alias "github.com/x/y/sub"
	_ "github.com/x/side-effect"
	. "github.com/x/wildcard"
)
`,
			want: []Import{
				{Module: "github.com/x/y", Kind: ImportModule, Line: 3},
				{Module: "github.com/x/y/sub", Kind: ImportModule, Names: []Name{{Alias: "alias"}}, Line: 4},
				{Module: "github.com/x/side-effect", Kind: ImportSideEffect, Line: 5},
				{Module: "github.com/x/wildcard", Kind: ImportWildcard, Line: 6},
			},
		},
		{
			filename: "app.rb",
			src:      "require \"octokit\"\nrequire 'octokit/client'\n",
			want: []Import{
				{Module: "octokit", Kind: ImportSideEffect, Line: 1},
				{Module: "octokit/client", Kind: ImportSideEffect, Line: 2},
			},
		},
		{
			filename: "lib.rs",
			src: `use serde::{Deserialize, Serialize as Ser};
use tokio_util::codec;
extern crate old_crate as old;
`,
			want: []Import{
				{
					Module: "serde",
					Kind:   ImportNamed,
					Names: []Name{
						{Name: "Deserialize"},
						{Name: "Serialize", Alias: "Ser"},
					},
					Line: 1,
				},
				{Module: "tokio_util", Kind: ImportNamed, Names: []Name{{Name: "codec"}}, Line: 2},
				{Module: "old_crate", Kind: ImportModule, Names: []Name{{Alias: "old"}}, Line: 3},
			},
		},
		{
			filename: "app.php",
			src: `<?php
use GuzzleHttp\Client;
use GuzzleHttp\HandlerStack as Stack;
`,
			want: []Import{
				{Module: `GuzzleHttp\Client`, Kind: ImportModule, Names: []Name{{Alias: "Client"}}, Line: 2},
				{Module: `GuzzleHttp\HandlerStack`, Kind: ImportModule, Names: []Name{{Alias: "Stack"}}, Line: 3},
			},
		},
		{
			filename: "app.ex",
			src: `alias Jason.Encoder
alias Phoenix.HTML, as: Html
import Jason
require Jason
use Jason
`,
			want: []Import{
				{Module: "Jason.Encoder", Kind: ImportModule, Names: []Name{{Alias: "Encoder"}}, Line: 1},
				{Module: "Phoenix.HTML", Kind: ImportModule, Names: []Name{{Alias: "Html"}}, Line: 2},
				{Module: "Jason", Kind: ImportWildcard, Line: 3},
				{Module: "Jason", Kind: ImportModule, Names: []Name{{Alias: "Jason"}}, Line: 4},
				{Module: "Jason", Kind: ImportModule, Names: []Name{{Alias: "Jason"}}, Line: 5},
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
