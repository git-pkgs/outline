package outline

import (
	"reflect"
	"testing"
)

func TestRegistryLanguageRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		filename  string
		src       string
		receivers []string
		want      []Ref
	}{
		{
			filename:  "app.dart",
			src:       "void main() {\n  http.get(uri);\n  other.get(uri);\n}\n",
			receivers: []string{"http"},
			want:      []Ref{{Receiver: "http", Member: "get", Line: 2}},
		},
		{
			filename:  "App.swift",
			src:       "Alamofire.request(\"url\")\nOther.request(\"url\")\n",
			receivers: []string{"Alamofire"},
			want:      []Ref{{Receiver: "Alamofire", Member: "request", Line: 1}},
		},
		{
			filename:  "App.hs",
			src:       "module App where\nimport qualified Data.Text as T\nx = T.pack \"x\"\n",
			receivers: []string{"T"},
			want:      []Ref{{Receiver: "T", Member: "pack", Line: 3}},
		},
		{
			filename:  "App.pm",
			src:       "my $x = JSON::MaybeXS->new;\nmy $y = Other->new;\n",
			receivers: []string{"JSON::MaybeXS"},
			want:      []Ref{{Receiver: "JSON::MaybeXS", Member: "new", Line: 1}},
		},
		{
			filename:  "app.lua",
			src:       "json.decode(\"{}\")\nother.decode(\"{}\")\n",
			receivers: []string{"json"},
			want:      []Ref{{Receiver: "json", Member: "decode", Line: 1}},
		},
		{
			filename:  "app.R",
			src:       "x <- dplyr::filter(data)\ny <- other::filter(data)\n",
			receivers: []string{"dplyr"},
			want:      []Ref{{Receiver: "dplyr", Member: "filter", Line: 1}},
		},
		{
			filename:  "app.jl",
			src:       "x = DataFrames.DataFrame()\ny = Other.DataFrame()\n",
			receivers: []string{"DataFrames"},
			want:      []Ref{{Receiver: "DataFrames", Member: "DataFrame", Line: 1}},
		},
		{
			filename:  "app.ml",
			src:       "let x = J.from_string \"{}\"\nlet y = Other.from_string \"{}\"\n",
			receivers: []string{"J"},
			want:      []Ref{{Receiver: "J", Member: "from_string", Line: 1}},
		},
		{
			filename:  "app.cr",
			src:       "x = JSON.parse(\"{}\")\ny = Other.parse(\"{}\")\n",
			receivers: []string{"JSON"},
			want:      []Ref{{Receiver: "JSON", Member: "parse", Line: 1}},
		},
		{
			filename:  "app.nim",
			src:       "log.info \"hello\"\nother.info \"hello\"\n",
			receivers: []string{"log"},
			want:      []Ref{{Receiver: "log", Member: "info", Line: 1}},
		},
		{
			filename:  "app.zig",
			src:       "const x = std.debug;\nconst y = other.debug;\n",
			receivers: []string{"std"},
			want:      []Ref{{Receiver: "std", Member: "debug", Line: 1}},
		},
		{
			filename:  "app.d",
			src:       "void f() { io . readText(\"x\"); auto x = io.value; other.readText(\"x\"); }\n",
			receivers: []string{"io"},
			want: []Ref{
				{Receiver: "io", Member: "readText", Line: 1},
				{Receiver: "io", Member: "value", Line: 1},
			},
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
