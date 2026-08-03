package outline

// Symbol/Ref extraction: a second output shape alongside Outline's text
// skeleton. Where Outline runs queries/<lang>.scm to decide which line ranges
// to keep, Extract runs queries/<lang>.symbols.scm to produce structured
// definition and reference records. BuildGraph in graph.go collects these
// across a tree and resolves references to definitions by name.
//
// Spike stage: only Go and Ruby have .symbols.scm files. Design and results
// are in claude/notes/graph-extraction.md.

import (
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// SymbolKind classifies a definition.
type SymbolKind string

const (
	KindFunc   SymbolKind = "func"
	KindMethod SymbolKind = "method"
	KindType   SymbolKind = "type"
	KindModule SymbolKind = "module"
	KindConst  SymbolKind = "const"
	KindVar    SymbolKind = "var"
)

// RefKind classifies a reference.
type RefKind string

const (
	RefCall    RefKind = "call"
	RefImport  RefKind = "import"
	RefType    RefKind = "type"
	RefInherit RefKind = "inherit"
)

// Symbol is a named definition found in a source file. Line numbers are
// 1-based.
type Symbol struct {
	Kind      SymbolKind `json:"kind"`
	Name      string     `json:"name"`
	Container string     `json:"container,omitempty"`
	File      string     `json:"file"`
	StartLine int        `json:"start_line"`
	EndLine   int        `json:"end_line"`
}

// Ref is a use of a name found in a source file. Target is the raw name text
// as written; From and To are populated by BuildGraph.
type Ref struct {
	Kind   RefKind `json:"kind"`
	Target string  `json:"target"`
	File   string  `json:"file"`
	Line   int     `json:"line"`
	From   *Symbol `json:"-"`
	To     *Symbol `json:"-"`
}

// Extract parses src and returns definitions and references. The third return
// is false when the file's language is not supported or has no symbol query.
func Extract(src []byte, filename string) ([]Symbol, []Ref, bool) {
	l, ok := detect(filename)
	if !ok {
		return nil, nil, false
	}
	l.init()
	if l.err != nil || l.symQuery == nil {
		return nil, nil, false
	}

	tree, err := l.pool.Parse(src)
	if err != nil || tree == nil {
		return nil, nil, false
	}
	defer tree.Release()
	if tree.ParseStoppedEarly() {
		return nil, nil, false
	}

	var syms []Symbol
	var refs []Ref
	for _, m := range l.symQuery.Execute(tree) {
		appendSymbolMatch(&syms, &refs, m, src, filename)
	}

	assignContainers(syms)
	dedupTypeRefs(syms, &refs)
	return syms, refs, true
}

func appendSymbolMatch(syms *[]Symbol, refs *[]Ref, m ts.QueryMatch, src []byte, file string) {
	var kind, name, container string
	var span *ts.Node
	for _, cap := range m.Captures {
		switch {
		case strings.HasPrefix(cap.Name, "def."):
			kind = cap.Name[4:]
			span = cap.Node
		case strings.HasPrefix(cap.Name, "ref."):
			kind = cap.Name
			span = cap.Node
		case cap.Name == "name":
			name = nodeText(cap.Node, src)
		case cap.Name == "container":
			container = nodeText(cap.Node, src)
		}
	}
	if span == nil || name == "" {
		return
	}
	if strings.HasPrefix(kind, "ref.") {
		*refs = append(*refs, Ref{
			Kind:   RefKind(kind[4:]),
			Target: strings.Trim(name, `"`),
			File:   file,
			Line:   int(span.StartPoint().Row) + 1,
		})
		return
	}
	*syms = append(*syms, Symbol{
		Kind:      SymbolKind(kind),
		Name:      name,
		Container: container,
		File:      file,
		StartLine: int(span.StartPoint().Row) + 1,
		EndLine:   int(span.EndPoint().Row) + 1,
	})
}

func nodeText(n *ts.Node, src []byte) string {
	return string(src[n.StartByte():n.EndByte()])
}

// assignContainers fills Container on symbols that lack one by finding the
// innermost enclosing type/module/class definition by line range. Requires
// syms to all be from the same file.
func assignContainers(syms []Symbol) {
	var scopes []*Symbol
	for i := range syms {
		s := &syms[i]
		if s.Kind == KindType || s.Kind == KindModule {
			scopes = append(scopes, s)
		}
	}
	sort.Slice(scopes, func(i, j int) bool {
		if scopes[i].StartLine != scopes[j].StartLine {
			return scopes[i].StartLine < scopes[j].StartLine
		}
		return scopes[i].EndLine > scopes[j].EndLine
	})
	for i := range syms {
		s := &syms[i]
		if s.Container != "" || s.Kind == KindType || s.Kind == KindModule {
			continue
		}
		for j := len(scopes) - 1; j >= 0; j-- {
			sc := scopes[j]
			if sc.StartLine <= s.StartLine && s.EndLine <= sc.EndLine {
				s.Container = sc.Name
				break
			}
		}
	}
}

// dedupTypeRefs drops @ref.type hits that coincide with a definition or
// another ref on the same line, since a bare (type_identifier)/(constant)
// pattern also matches inside (type_spec name:) and (call receiver:).
func dedupTypeRefs(syms []Symbol, refs *[]Ref) {
	defLines := make(map[int]string, len(syms))
	for _, s := range syms {
		defLines[s.StartLine] = s.Name
	}
	seen := make(map[[2]string]int)
	out := (*refs)[:0]
	for _, r := range *refs {
		if r.Kind == RefType {
			if defLines[r.Line] == r.Target {
				continue
			}
		}
		key := [2]string{string(r.Kind) + ":" + r.Target, r.File}
		if line, ok := seen[key]; ok && line == r.Line {
			continue
		}
		seen[key] = r.Line
		out = append(out, r)
	}
	*refs = out
}
