package outline

import (
	"sort"

	ts "github.com/odvcencio/gotreesitter"
)

// decl is a declaration captured for graph construction. Unlike Symbol it
// retains the full definition span and nesting so callers can compute node
// identities and containment before the tree is released.
type decl struct {
	Name     string
	Kind     string
	Line     int
	Exported bool
	Start    uint32
	End      uint32
	SigEnd   uint32
	Parent   int
}

// analysis is the per-file fact set that graph resolution consumes.
type analysis struct {
	Lang    string
	Decls   []decl
	Imports []Import
	Calls   []Call
}

// analyse parses src once and returns every fact the graph builder needs
// from a single file. The bool is false when the file's language is
// unsupported or the parse failed.
func analyse(src []byte, filename string) (*analysis, bool) {
	l, tree, ok := parseSource(src, filename)
	if !ok {
		return nil, false
	}
	defer tree.Release()

	root := tree.RootNode()
	matches := l.query.Execute(tree)

	a := &analysis{Lang: l.name}
	a.Decls = extractDecls(src, l, matches)
	a.Imports, _ = importsFor(src, l, root)
	a.Calls, _ = callsFor(src, l, root, a.Decls)
	return a, true
}

func extractDecls(src []byte, l *lang, matches []ts.QueryMatch) []decl {
	raw := make([]decl, 0, len(matches))
	for _, m := range matches {
		raw = append(raw, declsFromMatch(src, l, m)...)
	}
	if len(raw) == 0 {
		return nil
	}
	sort.Slice(raw, func(i, j int) bool {
		if raw[i].Start != raw[j].Start {
			return raw[i].Start < raw[j].Start
		}
		return raw[i].End > raw[j].End
	})
	assignParents(raw)
	return raw
}

func declsFromMatch(src []byte, l *lang, m ts.QueryMatch) []decl {
	var definition *ts.Node
	var kind string
	names := make([]*ts.Node, 0, 1)
	for _, cap := range m.Captures {
		switch cap.Name {
		case "symbol.name":
			names = append(names, cap.Node)
		case "symbol.func", "symbol.type", "symbol.class", "symbol.const", "symbol.var":
			definition = cap.Node
			kind = cap.Name[len("symbol."):]
		}
	}
	if definition == nil || kind == "" {
		return nil
	}
	start := definition.StartByte()
	end := definition.EndByte()
	sigEnd := end
	if body := definition.ChildByFieldName("body", l.language); body != nil {
		sigEnd = body.StartByte()
	}
	out := make([]decl, 0, len(names))
	for _, n := range names {
		if n == nil {
			continue
		}
		name := n.Text(src)
		if name == "" || name == "_" {
			continue
		}
		out = append(out, decl{
			Name:     name,
			Kind:     normalizeSymbolKind(l.name, kind, name, definition, src, l.language),
			Line:     int(n.StartPoint().Row) + 1,
			Exported: symbolExported(l.name, name, definition, src, l.language),
			Start:    start,
			End:      end,
			SigEnd:   sigEnd,
			Parent:   -1,
		})
	}
	return out
}

// assignParents sets Parent on each decl to the index of the innermost
// enclosing decl. Input must be sorted by Start ascending, End descending.
func assignParents(decls []decl) {
	var stack []int
	for i := range decls {
		for len(stack) > 0 && decls[stack[len(stack)-1]].End <= decls[i].Start {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			decls[i].Parent = stack[len(stack)-1]
		}
		stack = append(stack, i)
	}
}

// enclosing returns the index of the innermost decl whose span contains pos,
// or -1 if none does.
func enclosing(decls []decl, pos uint32) int {
	best := -1
	for i := range decls {
		if decls[i].Start > pos {
			break
		}
		if decls[i].End > pos {
			best = i
		}
	}
	return best
}
