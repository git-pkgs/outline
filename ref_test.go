package outline

import (
	"reflect"
	"testing"
)

func TestPythonRefs(t *testing.T) {
	t.Parallel()
	src := []byte(`other.jsonify()
f.jsonify()
value = f.response.status
text = "f.not_a_ref"
# f.also_not_a_ref
`)
	want := []Ref{
		{Receiver: "f", Member: "jsonify", Line: 2},
		{Receiver: "f", Member: "response", Line: 3},
	}

	got, ok := Refs(src, "app.py", []string{"f"})
	if !ok {
		t.Fatal("Refs() supported = false")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Refs() = %#v, want %#v", got, want)
	}
}

func TestJavascriptRefs(t *testing.T) {
	t.Parallel()
	src := []byte(`WS.Server;
Socket.OPEN;
Other.Server;
const text = "WS.not_a_ref";
// WS.also_not_a_ref
`)
	want := []Ref{
		{Receiver: "WS", Member: "Server", Line: 1},
		{Receiver: "Socket", Member: "OPEN", Line: 2},
	}

	for _, filename := range []string{"app.js", "app.ts", "app.tsx"} {
		got, ok := Refs(src, filename, []string{"WS", "Socket"})
		if !ok {
			t.Fatalf("Refs(%q) supported = false", filename)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Refs(%q) = %#v, want %#v", filename, got, want)
		}
	}
}

func TestRefsSupportResult(t *testing.T) {
	t.Parallel()
	if got, ok := Refs([]byte("value = 1\n"), "app.py", []string{"f"}); !ok || len(got) != 0 {
		t.Fatalf("supported empty Refs() = %#v, %v", got, ok)
	}
	if got, ok := Refs([]byte("value = 1\n"), "app.txt", []string{"f"}); ok || got != nil {
		t.Fatalf("unsupported Refs() = %#v, %v", got, ok)
	}
}

func TestHyrumLanguageRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		filename  string
		src       string
		receivers []string
		want      []Ref
	}{
		{
			filename:  "main.go",
			src:       "package main\nfunc f() { alias.Member(); other.Member() }\n",
			receivers: []string{"alias"},
			want:      []Ref{{Receiver: "alias", Member: "Member", Line: 2}},
		},
		{
			filename:  "app.rb",
			src:       "Octokit::Client.new\nOctokit.configure\nOther::Client.new\n",
			receivers: []string{"Octokit"},
			want: []Ref{
				{Receiver: "Octokit", Member: "Client", Line: 1},
				{Receiver: "Octokit", Member: "configure", Line: 2},
			},
		},
		{
			filename:  "lib.rs",
			src:       "fn f() { serde::value::Value; other::value::Value; }\n",
			receivers: []string{"serde"},
			want:      []Ref{{Receiver: "serde", Member: "value", Line: 1}},
		},
		{
			filename:  "app.php",
			src:       "<?php\nGuzzleHttp\\Utils::jsonDecode($value);\nOther\\Utils::jsonDecode($value);\n",
			receivers: []string{"GuzzleHttp"},
			want:      []Ref{{Receiver: "GuzzleHttp", Member: "Utils", Line: 2}},
		},
		{
			filename:  "app.ex",
			src:       "Jason.encode!(value)\nOther.encode!(value)\n",
			receivers: []string{"Jason"},
			want:      []Ref{{Receiver: "Jason", Member: "encode!", Line: 1}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.filename, func(t *testing.T) {
			t.Parallel()
			got, ok := Refs([]byte(test.src), test.filename, test.receivers)
			if !ok {
				t.Fatal("Refs() supported = false")
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Refs() = %#v, want %#v", got, test.want)
			}
		})
	}
}
