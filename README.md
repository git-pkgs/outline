# outline

Reduce a source tree to a structural skeleton suitable for feeding to an LLM.
Function and method bodies are dropped; signatures, types, comments and imports
are kept. Unsupported file types pass through unchanged.

Pure Go, no CGo. Parsing is done by [gotreesitter], file selection respects
`.gitignore` via [git-pkgs/gitignore].

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

`Pack(root string, opts Options) (*Result, error)` walks `root`, applies
`.gitignore` plus a built-in ignore list (vendored deps, build output,
lockfiles), skips binaries and oversized files, and outlines what it can.
`Options` lets you set size and file-count limits, extra ignore patterns,
concurrency, and whether to compress.

`Result` carries `[]File` and a rendered `Tree` string. `Result.Markdown(w)`
and `Result.XML(w)` write the packed document.

`Tree(paths []string) string` renders a box-drawing directory tree from a flat
path list.

## Languages

35 languages have body-stripping queries: Go, Ruby, Python, JavaScript,
TypeScript/TSX, Rust, Java, C, C++, C#, PHP, Kotlin, Swift, Scala, Dart,
Elixir, Erlang, Haskell, Clojure, Perl, Lua, R, Julia, OCaml, F#, Crystal, Nim,
Zig, D, Groovy, HCL/Terraform, Starlark/Bazel, CMake, Bash and Make.
gotreesitter ships ~200 grammars so adding a language means writing one `.scm`
query file.

## Compared to repomix

This is roughly the `--compress` core of [repomix] reimplemented in Go. On the
same inputs it produces equivalent declaration coverage, preserves indentation
(repomix strips it), and avoids a few cases where repomix leaks function bodies
in Ruby and Python. It does not do token counting, secret scanning, or remote
cloning; those belong elsewhere in git-pkgs.

## Performance

On an M1 Pro, outlining runs at ~2.5 MB/s per core and scales linearly across
cores via a parser pool. Packing a ~40-file Go module takes about 45ms. The
bottleneck is gotreesitter's full-parse path; our chunk extraction and
rendering are negligible by comparison.

[gotreesitter]: https://github.com/odvcencio/gotreesitter
[git-pkgs/gitignore]: https://github.com/git-pkgs/gitignore
[repomix]: https://github.com/yamadashy/repomix

## License

MIT
