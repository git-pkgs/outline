package outline

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

//go:embed queries/*.scm
var queryFS embed.FS

const Separator = "⋮----"

// DefaultParseTimeout caps how long the parser will spend on a single file.
// Past this point the file is treated as unsupported and falls through as raw
// content. Pathological inputs (large table-driven test files, generated
// parser tables) can otherwise dominate runtime.
const DefaultParseTimeout = time.Second

// parseTimeoutMicros is read by lang.init when configuring its ParserPool. It
// is package-level so all goroutines share the same value, set once via
// SetParseTimeout before any Outline call.
var parseTimeoutMicros uint64 = uint64(DefaultParseTimeout / time.Microsecond)

// SetParseTimeout overrides the per-file parse timeout. Must be called before
// the first Outline or Pack call; later calls are ignored once a language
// pool has been created. A zero duration disables the timeout.
func SetParseTimeout(d time.Duration) {
	parseTimeoutMicros = uint64(d / time.Microsecond)
}

type lang struct {
	name     string
	load     func() *ts.Language
	language *ts.Language
	pool     *ts.ParserPool
	query    *ts.Query
	once     sync.Once
	err      error
}

func (l *lang) init() {
	l.once.Do(func() {
		src, err := queryFS.ReadFile("queries/" + l.name + ".scm")
		if err != nil {
			l.err = err
			return
		}
		l.language = l.load()
		l.query, l.err = ts.NewQuery(string(src), l.language)
		if l.err != nil {
			return
		}
		l.pool = ts.NewParserPool(l.language, ts.WithParserPoolTimeoutMicros(parseTimeoutMicros))
	})
}

var langs = map[string]*lang{
	"go":         {name: "go", load: grammars.GoLanguage},
	"ruby":       {name: "ruby", load: grammars.RubyLanguage},
	"python":     {name: "python", load: grammars.PythonLanguage},
	"javascript": {name: "javascript", load: grammars.JavascriptLanguage},
	"typescript": {name: "typescript", load: grammars.TypescriptLanguage},
	"tsx":        {name: "typescript", load: grammars.TsxLanguage},
	"rust":       {name: "rust", load: grammars.RustLanguage},
	"java":       {name: "java", load: grammars.JavaLanguage},
	"c":          {name: "c", load: grammars.CLanguage},
	"cpp":        {name: "cpp", load: grammars.CppLanguage},
	"csharp":     {name: "csharp", load: grammars.CSharpLanguage},
	"php":        {name: "php", load: grammars.PhpLanguage},
	"kotlin":     {name: "kotlin", load: grammars.KotlinLanguage},
	"swift":      {name: "swift", load: grammars.SwiftLanguage},
	"scala":      {name: "scala", load: grammars.ScalaLanguage},
	"dart":       {name: "dart", load: grammars.DartLanguage},
	"elixir":     {name: "elixir", load: grammars.ElixirLanguage},
	"erlang":     {name: "erlang", load: grammars.ErlangLanguage},
	"haskell":    {name: "haskell", load: grammars.HaskellLanguage},
	"clojure":    {name: "clojure", load: grammars.ClojureLanguage},
	"perl":       {name: "perl", load: grammars.PerlLanguage},
	"lua":        {name: "lua", load: grammars.LuaLanguage},
	"r":          {name: "r", load: grammars.RLanguage},
	"julia":      {name: "julia", load: grammars.JuliaLanguage},
	"ocaml":      {name: "ocaml", load: grammars.OcamlLanguage},
	"fsharp":     {name: "fsharp", load: grammars.FsharpLanguage},
	"crystal":    {name: "crystal", load: grammars.CrystalLanguage},
	"nim":        {name: "nim", load: grammars.NimLanguage},
	"zig":        {name: "zig", load: grammars.ZigLanguage},
	"d":          {name: "d", load: grammars.DLanguage},
	"groovy":     {name: "groovy", load: grammars.GroovyLanguage},
	"hcl":        {name: "hcl", load: grammars.HclLanguage},
	"starlark":   {name: "starlark", load: grammars.StarlarkLanguage},
	"cmake":      {name: "cmake", load: grammars.CmakeLanguage},
	"bash":       {name: "bash", load: grammars.BashLanguage},
	"make":       {name: "make", load: grammars.MakeLanguage},
}

var byExt = map[string]string{
	".go":    "go",
	".rb":    "ruby",
	".py":    "python",
	".pyi":   "python",
	".js":    "javascript",
	".jsx":   "javascript",
	".mjs":   "javascript",
	".cjs":   "javascript",
	".ts":    "typescript",
	".mts":   "typescript",
	".cts":   "typescript",
	".tsx":   "tsx",
	".rs":    "rust",
	".java":  "java",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".hpp":   "cpp",
	".hh":    "cpp",
	".hxx":   "cpp",
	".cs":    "csharp",
	".php":   "php",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".swift": "swift",
	".scala": "scala",
	".sc":    "scala",
	".dart":  "dart",
	".ex":    "elixir",
	".exs":   "elixir",
	".erl":   "erlang",
	".hrl":   "erlang",
	".hs":    "haskell",
	".clj":    "clojure",
	".cljs":   "clojure",
	".cljc":   "clojure",
	".edn":    "clojure",
	".pl":     "perl",
	".pm":     "perl",
	".lua":    "lua",
	".r":      "r",
	".R":      "r",
	".jl":     "julia",
	".ml":     "ocaml",
	".mli":    "ocaml",
	".fs":     "fsharp",
	".fsi":    "fsharp",
	".fsx":    "fsharp",
	".cr":     "crystal",
	".nim":    "nim",
	".nims":   "nim",
	".zig":    "zig",
	".d":      "d",
	".groovy": "groovy",
	".gradle": "groovy",
	".tf":     "hcl",
	".tfvars": "hcl",
	".hcl":    "hcl",
	".bzl":    "starlark",
	".star":   "starlark",
	".cmake":  "cmake",
	".sh":     "bash",
	".bash":   "bash",
	".zsh":    "bash",
	".mk":     "make",
}

var byName = map[string]string{
	"Makefile":        "make",
	"GNUmakefile":     "make",
	"makefile":        "make",
	"CMakeLists.txt":  "cmake",
	"BUILD":           "starlark",
	"BUILD.bazel":     "starlark",
	"WORKSPACE":       "starlark",
	"WORKSPACE.bazel": "starlark",
}

func detect(filename string) (*lang, bool) {
	base := filename
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	if name, ok := byName[base]; ok {
		return langs[name], true
	}
	dot := strings.LastIndexByte(base, '.')
	if dot < 0 {
		return nil, false
	}
	name, ok := byExt[base[dot:]]
	if !ok {
		return nil, false
	}
	return langs[name], true
}

// chunk is a contiguous run of source lines to keep.
type chunk struct {
	startRow uint32
	endRow   uint32
	startOff int
	endOff   int
}

// Outline reduces source to a structural skeleton: declarations, signatures
// and comments are kept, function bodies are dropped. Returns the outline and
// true if the file's language is supported, otherwise "", false.
func Outline(src []byte, filename string) (string, bool) {
	out, _, ok := outlineSource(src, filename, false)
	return out, ok
}

func outlineFile(src []byte, filename string) (string, []Symbol, bool) {
	return outlineSource(src, filename, true)
}

func outlineSource(src []byte, filename string, collectSymbols bool) (string, []Symbol, bool) {
	l, tree, ok := parseSource(src, filename)
	if !ok {
		return "", nil, false
	}
	defer tree.Release()

	lineStart, lineEnd := indexLines(src)
	matches := l.query.Execute(tree)
	var symbols []Symbol
	if collectSymbols {
		symbols = extractSymbols(src, l, tree.RootNode(), matches)
	}

	chunks := make([]chunk, 0, len(matches))
	for _, m := range matches {
		chunks = appendMatch(chunks, m, lineStart, lineEnd)
	}
	if len(chunks) == 0 {
		return "", symbols, true
	}

	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].startRow != chunks[j].startRow {
			return chunks[i].startRow < chunks[j].startRow
		}
		return chunks[i].endRow < chunks[j].endRow
	})

	merged := merge(dedup(chunks))

	var b strings.Builder
	for i, c := range merged {
		if i > 0 {
			b.WriteByte('\n')
			b.WriteString(Separator)
			b.WriteByte('\n')
		}
		b.Write(bytes.TrimRight(src[c.startOff:c.endOff], " \t\n"))
	}
	return b.String(), symbols, true
}

func parseSource(src []byte, filename string) (*lang, *ts.Tree, bool) {
	l, ok := detect(filename)
	if !ok {
		return nil, nil, false
	}
	l.init()
	if l.err != nil {
		return nil, nil, false
	}

	tree, err := l.pool.Parse(src)
	if err != nil || tree == nil {
		return nil, nil, false
	}
	if tree.ParseStoppedEarly() {
		tree.Release()
		return nil, nil, false
	}
	return l, tree, true
}

// appendMatch converts a query match into kept line ranges. Each @keep
// capture emits its full node range. A @signature capture emits from the node
// start up to (but not including) the line where the same match's @body
// begins, or just the start line if there is no @body.
func appendMatch(dst []chunk, m ts.QueryMatch, lineStart, lineEnd []int) []chunk {
	var sig, body *ts.Node
	for _, cap := range m.Captures {
		switch cap.Name {
		case "keep":
			dst = append(dst, lineChunk(cap.Node.StartPoint().Row, cap.Node.EndPoint().Row, lineStart, lineEnd))
		case "signature":
			sig = cap.Node
		case "body":
			body = cap.Node
		}
	}
	if sig != nil {
		startRow := sig.StartPoint().Row
		endRow := startRow
		if body != nil && body.StartPoint().Row > startRow {
			endRow = body.StartPoint().Row - 1
		}
		dst = append(dst, lineChunk(startRow, endRow, lineStart, lineEnd))
	}
	return dst
}

func lineChunk(startRow, endRow uint32, lineStart, lineEnd []int) chunk {
	if int(endRow) >= len(lineEnd) {
		endRow = uint32(len(lineEnd) - 1)
	}
	return chunk{
		startRow: startRow,
		endRow:   endRow,
		startOff: lineStart[startRow],
		endOff:   lineEnd[endRow],
	}
}

// dedup keeps only the shortest chunk for each distinct start row, so a
// @signature truncation beats a @keep over the same node. Input must be sorted
// by startRow ascending, endRow ascending.
func dedup(chunks []chunk) []chunk {
	out := chunks[:1]
	for _, c := range chunks[1:] {
		if c.startRow == out[len(out)-1].startRow {
			continue
		}
		out = append(out, c)
	}
	return out
}

// merge collapses chunks that overlap or sit on adjacent lines. Input must be
// sorted by startRow ascending with one chunk per start row.
func merge(chunks []chunk) []chunk {
	out := chunks[:1]
	for _, c := range chunks[1:] {
		last := &out[len(out)-1]
		if c.startRow <= last.endRow+1 {
			if c.endRow > last.endRow {
				last.endRow = c.endRow
				last.endOff = c.endOff
			}
		} else {
			out = append(out, c)
		}
	}
	return out
}

// indexLines returns parallel slices of byte offsets such that
// src[lineStart[i]:lineEnd[i]] is the content of line i without its newline.
func indexLines(src []byte) (lineStart, lineEnd []int) {
	n := bytes.Count(src, []byte{'\n'}) + 1
	lineStart = make([]int, 1, n)
	lineEnd = make([]int, 0, n)
	for i, b := range src {
		if b == '\n' {
			lineEnd = append(lineEnd, i)
			lineStart = append(lineStart, i+1)
		}
	}
	lineEnd = append(lineEnd, len(src))
	return
}

// Supported reports whether Outline can handle the given filename.
func Supported(filename string) bool {
	_, ok := detect(filename)
	return ok
}

func init() {
	for _, l := range langs {
		if _, err := queryFS.ReadFile("queries/" + l.name + ".scm"); err != nil {
			panic(fmt.Sprintf("outline: missing query for %s: %v", l.name, err))
		}
	}
}
