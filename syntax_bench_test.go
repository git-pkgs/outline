package outline

import "testing"

func BenchmarkImportsPython(b *testing.B) {
	src := []byte(`import flask
import yaml as y
from werkzeug.http import (
    parse_authorization_header as parse,
    dump_header,
)
`)
	Imports(src, "warm.py")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Imports(src, "app.py")
	}
}

func BenchmarkImportsJavascript(b *testing.B) {
	src := []byte(`import ws from "ws";
import { Server, WebSocket as Socket } from "ws";
const Receiver = require("dep").Server;
`)
	Imports(src, "warm.js")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Imports(src, "app.js")
	}
}

func BenchmarkRefsJavascript(b *testing.B) {
	src := []byte(`WS.Server;
Socket.OPEN;
Other.Server;
`)
	receivers := []string{"WS", "Socket"}
	Refs(src, "warm.js", receivers)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Refs(src, "app.js", receivers)
	}
}
