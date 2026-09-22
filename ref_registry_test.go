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
			filename: "App.java",
			src: `import java.util.List;
import java.util.Collections;
class App {
    void run() {
        List.of("x").size();
        Object value = Collections.EMPTY_LIST;
        List.<String>of("x");
        List
            .of("x");
        Other.of("x");
        holder.List.of("x");
        String text = "List.ignored";
        // List.ignored();
    }
}
`,
			receivers: []string{"List", "Collections", "java"},
			want: []Ref{
				{Receiver: "List", Member: "of", Line: 5},
				{Receiver: "Collections", Member: "EMPTY_LIST", Line: 6},
				{Receiver: "List", Member: "of", Line: 7},
				{Receiver: "List", Member: "of", Line: 9},
			},
		},
		{
			filename: "App.kt",
			src: `import java.util.Collections as C
fun run(value: String?) {
    C.emptyList<String>().size
    val value = C.EMPTY_LIST
    value?.trim()
    C
        .emptyList<String>()
    Other.emptyList<String>()
    holder.C.emptyList<String>()
    val text = "C.ignored"
    // C.ignored()
}
`,
			receivers: []string{"C", "java", "value"},
			want: []Ref{
				{Receiver: "C", Member: "emptyList", Line: 3},
				{Receiver: "C", Member: "EMPTY_LIST", Line: 4},
				{Receiver: "value", Member: "trim", Line: 5},
				{Receiver: "C", Member: "emptyList", Line: 7},
			},
		},
		{
			filename:  "App.kts",
			src:       "import java.io.File as F\nprintln(F.separator)\n",
			receivers: []string{"F"},
			want:      []Ref{{Receiver: "F", Member: "separator", Line: 2}},
		},
		{
			filename: "App.cs",
			src: `using IO = System.IO;
using Text = System.String;
using Enumerable = System.Linq.Enumerable;
class App {
    void Run(string value) {
        IO.File.ReadAllText("x");
        Text.Concat("a", "b");
        Text.Empty.ToString();
        Enumerable.Empty<string>();
        Text
            .Concat("a", "b");
        value?.Trim();
        Other.Concat();
        holder.Text.Concat();
        string text = "Text.ignored";
        // Text.ignored();
    }
}
`,
			receivers: []string{"IO", "Text", "Enumerable", "System", "value"},
			want: []Ref{
				{Receiver: "IO", Member: "File", Line: 6},
				{Receiver: "Text", Member: "Concat", Line: 7},
				{Receiver: "Text", Member: "Empty", Line: 8},
				{Receiver: "Enumerable", Member: "Empty", Line: 9},
				{Receiver: "Text", Member: "Concat", Line: 11},
				{Receiver: "value", Member: "Trim", Line: 12},
			},
		},
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

func TestJavaKotlinCsharpEmptyResults(t *testing.T) {
	t.Parallel()
	for _, filename := range []string{"App.java", "App.kt", "App.kts", "App.cs"} {
		t.Run(filename, func(t *testing.T) {
			src := []byte("class App {}\n")
			if got, ok := Imports(src, filename); !ok || len(got) != 0 {
				t.Fatalf("Imports() = %#v, %v, want no imports and supported", got, ok)
			}
			if got, ok := Refs(src, filename, []string{"App"}); !ok || len(got) != 0 {
				t.Fatalf("Refs() = %#v, %v, want no refs and supported", got, ok)
			}
		})
	}
}
