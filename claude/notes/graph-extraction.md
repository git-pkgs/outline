# Symbol and edge extraction for outline

August 2026. Prompted by looking at https://github.com/Graphify-Labs/graphify (Python, ~55k LOC, YC S26) which builds a queryable code graph from tree-sitter and pitches itself as "the thing an AI agent runs first". brief occupies the same slot. graphify's code-extraction tier is deterministic, local, and tree-sitter based, which is exactly outline's model, so the interesting question is which of its capabilities belong here.

outline's real consumers are `brief outline` (LLM context packing, https://github.com/git-pkgs/brief), and via brief the two downstream Go projects scrutineer (https://github.com/alpha-omega-security/scrutineer, containerised security scanning and disclosure triage) and upkeep (https://github.com/upkeep/upkeep, autonomous package-maintenance bots across 13 ecosystems). Feature choices below are filtered through what those three need and through brief's design principles in `brief/prd.md`: deterministic, single binary, zero runtime deps, offline, fast, stateless. graphify's LLM semantic tier, watch mode, persisted `graphify-out/` state, HTML/SVG/Neo4j exporters, Leiden clustering, and 20-platform skill installers all fall outside those principles and are out of scope. MCP is deferred.

## What changes in outline

Today `outline.Outline()` runs a tree-sitter query with three capture names (`@keep`, `@signature`, `@body`) and emits kept line ranges as text. The plan is to widen the capture vocabulary so the same `.scm` files also tag definitions and references, and to add a second output shape alongside the text chunks.

New captures, applied per language in `queries/*.scm`:

    @def.func @def.method @def.type @def.const @def.var @def.module
    @ref.call @ref.import @ref.type @ref.inherit
    @name           (child capture: the identifier within a @def.* or @ref.*)
    @container      (child capture: enclosing class/module for a @def.method)

Each `@def.*` match becomes a `Symbol{Kind, Name, Container, File, StartLine, EndLine}`. Each `@ref.*` match becomes a `Ref{Kind, Target string, File, Line, From *Symbol}`. `appendMatch` in `outline.go:270` grows a second branch that populates these instead of line chunks. The existing `@keep`/`@signature`/`@body` behaviour is unchanged; both run off one parse.

`go.scm` today is 24 lines. With the new captures it grows by roughly this:

    (function_declaration name: (identifier) @name) @def.func
    (method_declaration name: (field_identifier) @name
      receiver: (parameter_list (parameter_declaration type: (_) @container))) @def.method
    (type_spec name: (type_identifier) @name) @def.type
    (call_expression function: [(identifier) (selector_expression)] @name) @ref.call
    (import_spec path: (interpreted_string_literal) @name) @ref.import

graphify does the equivalent with ~5000 lines of imperative AST walking in `extractors/engine.py` plus 200-700 LOC per language of overrides. The declarative approach should hold for the definition captures across all 35 languages outline already supports. `@ref.call` is where per-language variance shows up (method call syntax differs a lot) and a few languages may need a small Go-side helper alongside the query, the way graphify's `LanguageConfig` carries per-language accessor functions.

New file `graph.go` with:

    type Symbol struct { Kind, Name, Container, File string; StartLine, EndLine int }
    type Ref    struct { Kind, Target, File string; Line int; From, To *Symbol }
    type Graph  struct { Symbols []Symbol; Refs []Ref; byName map[string][]*Symbol }

    func Extract(src []byte, filename string) ([]Symbol, []Ref, bool)
    func BuildGraph(root string, opts Options) (*Graph, error)
    func (g *Graph) Callers(name string) []*Symbol
    func (g *Graph) Affected(name string, depth int) []*Symbol
    func (g *Graph) Path(from, to string) []*Ref
    func (g *Graph) Imports(file string, reverse bool) []string

`BuildGraph` reuses `Pack`'s file walk, gitignore filtering, binary detection, and worker pool. After per-file extraction it runs one resolve pass: for each `Ref`, look up `Target` in `byName` scoped by the source file's import edges, and set `Ref.To` when exactly one definition matches. Unresolved refs keep `To == nil` and the raw target string. This is EXTRACTED-only resolution: no receiver-type inference, no chasing assignments through returns. graphify spends ~2700 LOC on the inferred tier and tags results INFERRED/AMBIGUOUS; scrutineer prefers precision over recall so a false-positive caller costs more than a missed one, and the inferred tier can come later if the hit rate is too low in practice.

brief's detection output feeds resolution. brief already reports the package manager, source layout, and module conventions, which is exactly what import-path-to-file resolution needs. graphify reimplements this per language inside its resolver; running `BuildGraph` after `brief.Detect()` means the resolver can ask "where does `./config` resolve for a Go module" or "what does `require 'foo'` map to in this gem" using knowledge brief already has. The `Options` struct gains a `*report.Report` field so callers can pass detection results in.

New renderer `Result.JSON(w)` in `render.go` alongside `Markdown` and `XML`, emitting `{symbols: [...], refs: [...]}`.

## What changes in `brief outline`

The markdown pack gets smarter without changing how it's invoked. All of this runs off the `Symbol`/`Ref` data the extraction pass now produces.

Per-file summary line before each code block: `defines: Server, Handler, Run() | imports: net/http, ./config | called by: cmd/main.go`. An agent reading the pack can skip a file if the header answers the question.

A structural preamble before `## Files`: entry points (symbols with zero inbound `calls`), most-referenced types by degree, exported symbols with no in-repo callers, import cycles if any. These are structural facts about the code in the same way `brief .` reports facts about the toolchain.

Cross-reference annotations on kept signatures. When a stripped signature mentions a type defined in-repo, append `// Request -> server/types.go:12`. The body is still elided but the signature points somewhere.

Centrality-aware truncation. `MaxFiles` currently drops whatever walks last. With degree counts, `--budget N` keeps the highest-degree files first and drops leaves, so the router isn't the file that fell off the end.

Subsystem grouping. Order files by import-graph connected component rather than alphabetically, so related code is adjacent in the pack. Cheap approximation of graphify's community detection without a clustering dependency.

`brief outline --diff <ref>` packs only files changed since `<ref>` plus their callers and callees to a configurable depth. This is the outline analogue of `brief diff` and it's what an upkeep bot reviewing a PR wants: the diff plus its blast radius, not the whole repo.

## New `brief graph` subcommand

    brief graph [path|url]                 emit JSON graph
    brief graph defs <name>                where is X defined
    brief graph callers <name>             what calls X
    brief graph affected <name> [--depth]  reverse reachability from X
    brief graph path <a> <b>               shortest ref-path from A to B
    brief graph imports <file> [--reverse] import edges in/out of a file

Same `remote.Resolve` plumbing as `brief outline`, so `brief graph pypi:requests` and `brief graph https://github.com/foo/bar` work. JSON on pipe, human table on TTY, matching the rest of brief.

scrutineer uses `Path` from HTTP handlers to `brief sinks` output for reachability, and `Affected` on a vulnerable function for blast radius. upkeep uses `Imports --reverse` on a bumped dependency, `Callers` on a changed signature, and `Affected` for dead-code candidates. Both link the Go package directly rather than shelling out.

## Straight quality fixes to current outline

These are independent of the graph work and each closes a gap graphify already handles.

Shebang dispatch in `detect()` (`outline.go:184`). Extensionless files with `#!/usr/bin/env python|ruby|node|bash|...` currently fall through as unsupported.

Content sniffing for ambiguous extensions. `.h` is currently always C; graphify checks for `@interface`/`namespace`/`template` to route to ObjC or C++. `.m` is ObjC vs MATLAB. `.inc` and `.t` similarly.

`.outlineignore` file read from the walk root, merged after `.gitignore` with the same semantics graphify uses for `.graphifyignore` (evaluated last, `!` negation supported, never re-includes what `.gitignore` excluded).

Rationale-comment collection. `@keep` already retains `# NOTE:`/`# WHY:`/`# HACK:`/`# TODO:` lines in place. A `@rationale` capture on those patterns lets the pack surface them as a "design notes" block per file or in the preamble, and lets `brief graph` emit them as nodes if wanted later.

## Open questions

Does the declarative `.scm` approach carry `@ref.call` across all 35 languages, or do method-call-heavy languages (Ruby, Python, JS) need a Go-side visitor? First spike should be Go plus one dynamic language to find out.

Is EXTRACTED-only resolution good enough for scrutineer's sink reachability, or does missing receiver-type inference leave too many `Ref.To == nil` on the paths that matter? Measure hit rate on a few real repos before building the inferred tier.

Language priority is upkeep's 13 ecosystems: Ruby, Go, JS/TS, Python, Rust, Java, PHP, C#, Perl, Swift, Elixir, Dart, Haskell. outline already parses all of them; the work is writing the new captures.

Does `Graph` stay in `git-pkgs/outline` or split to `git-pkgs/codegraph`? The `.scm` files, `langs` table, and parser pools want to be shared either way, and `Extract` sits naturally next to `Outline` since both consume the same parse. Leaning towards keeping it here until the file gets unwieldy.

`brief outline` enrichments: on by default, or behind `--graph`? The per-file summary line and preamble are cheap and probably default-on. Cross-reference annotations change the output format enough that existing consumers parsing it might break, so flag-gated to start.

## Not doing

LLM semantic pass over docs/PDFs/images/video. Watch mode and on-disk graph cache. HTML/SVG/Mermaid/Obsidian/wiki export. Neo4j/FalkorDB. Leiden or any community detection needing a graph-algorithm dependency. INFERRED-confidence receiver-type heuristics (deferred, not ruled out). AI-assistant skill installers. MCP server (on brief's todo separately, deferred here).

## First spike

Extend `queries/go.scm` and `queries/ruby.scm` with the new captures, add `Extract()` and a minimal `BuildGraph()` with name-index resolution, run it on brief itself and on one of upkeep's Ruby target packages, and count how many `@ref.call` targets resolve. That answers the two biggest open questions (does declarative hold, is EXTRACTED-only enough) before touching the other 33 languages or any of the `brief outline` rendering changes.

## Spike results (2026-08-03)

~490 LOC of new Go across `symbols.go` + `graph.go` + tests, plus 34-line `go.symbols.scm` and 51-line `ruby.symbols.scm`. Separate `.symbols.scm` files rather than extending the existing ones, loaded best-effort into a new `lang.symQuery` field so languages without one just return `ok=false` from `Extract`. Existing `Outline`/`Pack` behaviour untouched; full test suite and golangci-lint pass.

Resolution rate on real repos (`outline-compare -graph <dir>`):

    brief (Go, 37 files, 734 syms)
      call     3228 total   991 in-repo   953 unique   29.5%
      type     2065 total   578 in-repo   556 unique   26.9%

    jekyll (Ruby, 156 files, 1481 syms)
      call     9825 total  4641 in-repo  3084 unique   31.4%
      type     3005 total  2002 in-repo  1471 unique   49.0%
      inherit   130 total    87 in-repo    79 unique   60.8%

    outline itself (Go, 21 files, 191 syms)
      call      544 total   159 in-repo   154 unique   28.3%
      type      507 total   160 in-repo   159 unique   31.4%

The ~30% overall call-resolution ceiling is expected: most calls in any repo target stdlib or third-party packages, not in-repo symbols. Of calls whose target name *does* exist in-repo, Go resolves 96% unambiguously (953/991) and Ruby 66% (3084/4641). The Ruby gap is method-name collision (`render` is defined 10+ times in jekyll) and is exactly where graphify's receiver-type inference would help.

`Callers()` output is correct on every probe tried. `outline-compare -graph . -callers detect` returns Outline, Supported, readFile, Extract, TestDetectByName. On brief, `-callers FilterByChangedFiles` returns cmdDiff plus its six test callers. On jekyll, `-callers render` returns Site#process, LiquidRenderer::File, ThemeBuilder#template, and the tag classes.

Answers to the open questions:

The declarative `.scm` approach holds for both languages. Go was clean. Ruby needed one non-obvious pattern beyond `(call)`: bare identifiers in `(body_statement)` position, since parenless argless method calls parse as plain `(identifier)` not `(call)`. That over-captures local-var references but those don't resolve to a symbol so they wash out as noise in the unresolved bucket. No Go-side per-language visitor was needed.

EXTRACTED-only resolution is enough to make `Callers`/`Affected` useful now. It is not enough to make Ruby `path A B` reliable because a third of in-repo Ruby calls stay ambiguous. The next resolution improvement is not receiver-type inference; it is scoping the name index by the ref's `Container` (a call inside `Cart#total` should prefer a `Cart#foo` definition over a `Site#foo` one). That is still deterministic and cheap.

Loose ends noted: `Callers` returns duplicate entries when one function calls the target more than once (dedupe on caller identity, not per-ref). `@ref.type` on Go over-matches builtins (`string`, `error`, `int`) which should be filtered like graphify's `_PYTHON_TYPE_CONTAINERS`. Imports resolve 0% because there is no import-path-to-file mapping yet; that is where brief's ecosystem knowledge plugs in.
