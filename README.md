# outline

Reduce a source tree to a structural skeleton suitable for feeding to an LLM.
Function and method bodies are dropped; signatures, types, comments and imports
are kept. Unsupported file types pass through unchanged.

Pure Go, no CGo. Parsing is done by [gotreesitter], file selection respects
`.gitignore` via [git-pkgs/gitignore]. Full API docs are on [pkg.go.dev].

```go
import "github.com/git-pkgs/outline"

r, err := outline.Pack(".", outline.Options{Compress: true})
if err != nil {
    return err
}
r.Markdown(os.Stdout)
```

Or per file:

```go
src, _ := os.ReadFile("main.go")
out, ok := outline.Outline(src, "main.go")
```

A Go file like

```go
func SayHello(name string) {
    fmt.Printf("Hello, %s!\n", name)
}
```

becomes

```
func SayHello(name string) {
⋮----
```

with the body elided and gaps marked by `⋮----`.

## API

`Outline(src []byte, filename string) (string, bool)` compresses one file. The
second return is false if the language is not supported.

`Imports(src []byte, filename string) ([]Import, bool)` extracts module imports,
their source-language form, named imports, local aliases, and one-based source
lines. A statement containing both default and named imports returns one value
for each form.

`Refs(src []byte, filename string, receivers []string) ([]Ref, bool)` extracts
direct member accesses on the supplied receiver identifiers. This lets callers
pass the local aliases returned by `Imports` without collecting unrelated
member expressions from the file.

```go
imports, ok := outline.Imports(src, "app.py")
if !ok {
    return
}

refs, _ := outline.Refs(src, "app.py", []string{"flask", "f"})
```

For both functions, false means the language is unsupported or parsing did not
complete, including a parse timeout. A true result with an empty slice means
the language is supported but the file contains no matches. Import and
reference extraction currently cover Go, Ruby, Python, JavaScript,
TypeScript/TSX, Rust, PHP, Elixir, Dart, Swift, Haskell, Perl, Lua,
R, Julia, OCaml, Crystal, Nim, Zig, and D.

`Pack(root string, opts Options) (*Result, error)` walks `root`, applies
`.gitignore` plus a built-in ignore list (vendored deps, build output,
lockfiles), skips binaries and oversized files, and outlines what it can.
`Options` lets you set size and file-count limits, extra ignore patterns,
concurrency, and whether to compress.

`Result` carries `[]File` and a rendered `Tree` string. `Result.Markdown(w)`
and `Result.XML(w)` write the packed document.

`Build(root string, opts Options) (*Graph, error)` walks `root` the same way as
`Pack` and returns a directed code graph. Nodes are files, modules,
declarations, and unresolved external call targets; each edge has a relation
(`contains`, `imports`, `calls`) and a confidence (`extracted` for facts read
directly from one AST, `inferred` for cross-file name resolution). `Graph`
methods query the result: `Def(name)` looks up by node ID, bare name, or
qualified name; `Callers`/`Callees` return one-hop call edges;
`Affected(seeds, opts)` returns reverse-reachable evidence paths from a set of
sinks; `Path(from, to, opts)` returns the shortest forward call chain;
`JSON(w)` writes sorted output so repeated builds of unchanged input are
byte-identical. Call extraction and cross-file resolution currently cover Go
and Python; for other supported languages the graph contains file, symbol,
module, and import structure with call edges omitted.

`Tree(paths []string) string` renders a box-drawing directory tree from a flat
path list.

`Supported(filename string) bool` reports whether a file's extension maps to a
language with an outlining query.

`SetParseTimeout(d time.Duration)` overrides the per-file parse timeout
(default 1s). Must be called before the first `Outline` or `Pack` call.

## outline binary

```
go install github.com/git-pkgs/outline/cmd/outline@latest
```

`outline graph [dir]` builds the graph and prints a summary; `-json` writes it
to stdout and `-o file` to disk. `outline def <name>`,
`outline callers <name>`, `outline callees <name>`,
`outline affected <name>...`, and `outline path <from> <to>` query it. `-g
file` reads a saved graph instead of rebuilding; `-inferred` includes
cross-file name-matched edges alongside extracted ones; `-depth N` caps
traversal hops; `-relations r,...` selects which edge kinds to follow (default
`calls`); `-budget N` caps output size in tokens. Flags precede positional
arguments.

Seeds take the same forms as `Def`: `Handler`, `Handler.process`, or a full
node ID. External targets use `ext:<lang>:<module>:<name>`, so tracing every
declaration that reaches `exec.Command` from the `testdata/cli-go` fixture
produces:

```
$ outline affected -inferred -dir testdata/cli-go ext:go:os/exec:Command
2 paths, 3 nodes
NODE Handler func main.go:9 exported=true sig=func Handler(name string) error
NODE exec.Command external  exported=false sig=
NODE main func main.go:17 exported=false sig=func main()
EDGE Handler --calls[inferred]--> exec.Command at main.go:14
EDGE Handler --calls[inferred]--> Load at main.go:10
NODE Load func store/store.go:7 exported=true sig=func Load(name string) (Record, error)
EDGE main --calls[extracted]--> Handler at main.go:18
...
```

## Languages

35 languages have body-stripping queries: Go, Ruby, Python, JavaScript,
TypeScript/TSX, Rust, Java, C, C++, C#, PHP, Kotlin, Swift, Scala, Dart,
Elixir, Erlang, Haskell, Clojure, Perl, Lua, R, Julia, OCaml, F#, Crystal, Nim,
Zig, D, Groovy, HCL/Terraform, Starlark/Bazel, CMake, Bash and Make.
gotreesitter ships ~200 grammars so adding a language means writing one `.scm`
query file. `cmd/outline-compare -dump <lang>` prints the S-expression tree for
stdin and is the easiest way to work out what to capture.

## Performance

On an M1 Pro, outlining runs at ~6 MB/s per core and reaches ~36 MB/s across
all eight via the parser pool. Packing a 600-file repo takes about 34ms; the
Markdown render of that result is ~140µs. Almost all the time is in
gotreesitter's full-parse path; chunk extraction and rendering barely register.

[gotreesitter]: https://github.com/odvcencio/gotreesitter
[git-pkgs/gitignore]: https://github.com/git-pkgs/gitignore
[pkg.go.dev]: https://pkg.go.dev/github.com/git-pkgs/outline

## License

MIT
