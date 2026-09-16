package outline

import (
	"sort"

	ts "github.com/odvcencio/gotreesitter"
)

// decl is a declaration captured for graph construction. Unlike Symbol it
// retains the full definition span and nesting so callers can compute node
// identities and containment before the tree is released.
type decl struct {
	Name      string
	Kind      string
	Line      int
	Exported  bool
	NameAt    uint32
	Start     uint32
	End       uint32
	SigEnd    uint32
	Parent    int
	Params    []string
	Singleton bool
	Owner     string
}

func (d decl) symID(path string) string {
	return SymID(path, d.NameAt)
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
	a.Decls = extractDecls(src, l, root, matches)
	a.Imports, _ = importsFor(src, l, root)
	a.Calls, _ = callsFor(src, l, root, a.Decls)
	return a, true
}

func extractDecls(src []byte, l *lang, root *ts.Node, matches []ts.QueryMatch) []decl {
	raw := make([]decl, 0, len(matches))
	for _, m := range matches {
		raw = append(raw, declsFromMatch(src, l, m)...)
	}
	if exported, merge, ok := explicitExports(l, root, src); ok {
		for i := range raw {
			if merge {
				raw[i].Exported = raw[i].Exported || exported[raw[i].Name]
			} else {
				raw[i].Exported = exported[raw[i].Name]
			}
		}
	}
	if len(raw) == 0 {
		return nil
	}
	sort.Slice(raw, func(i, j int) bool {
		if raw[i].Start != raw[j].Start {
			return raw[i].Start < raw[j].Start
		}
		if raw[i].End != raw[j].End {
			return raw[i].End > raw[j].End
		}
		return raw[i].NameAt < raw[j].NameAt
	})
	assignParents(raw)
	return raw
}

func declsFromMatch(src []byte, l *lang, m ts.QueryMatch) []decl {
	var definition *ts.Node
	var kind string
	exported := false
	names := make([]*ts.Node, 0, 1)
	for _, cap := range m.Captures {
		switch cap.Name {
		case "symbol.name":
			names = append(names, cap.Node)
		case "symbol.exported":
			exported = true
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
	params := extractParams(src, l, definition)
	singleton := l.name == "ruby" && definition.Type(l.language) == "singleton_method"
	owner := ""
	if singleton {
		if object := definition.ChildByFieldName("object", l.language); object != nil {
			owner = object.Text(src)
		}
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
			Name:      name,
			Kind:      normalizeSymbolKind(l.name, kind, name, definition, src, l.language),
			Line:      int(n.StartPoint().Row) + 1,
			Exported:  exported || symbolExported(l.name, name, definition, src, l.language),
			NameAt:    n.StartByte(),
			Start:     start,
			End:       end,
			SigEnd:    sigEnd,
			Parent:    -1,
			Params:    params,
			Singleton: singleton,
			Owner:     owner,
		})
	}
	return out
}

// extractParams returns the parameter (and, for Go, receiver and named
// result) identifiers introduced by a function declaration, so lexical
// resolution can detect when a call target is shadowed.
func extractParams(src []byte, l *lang, def *ts.Node) []string {
	var out []string
	collect := func(list *ts.Node) {
		if list == nil {
			return
		}
		for i := range list.NamedChildCount() {
			p := list.NamedChild(i)
			switch p.Type(l.language) {
			case "identifier":
				out = append(out, p.Text(src))
			case "parameter_declaration", "variadic_parameter_declaration":
				for j := range p.NamedChildCount() {
					if c := p.NamedChild(j); c.Type(l.language) == "identifier" {
						out = append(out, c.Text(src))
					}
				}
			default:
				if n := p.ChildByFieldName("name", l.language); n != nil {
					out = append(out, n.Text(src))
				} else if p.NamedChildCount() > 0 {
					if c := p.NamedChild(0); c.Type(l.language) == "identifier" {
						out = append(out, c.Text(src))
					}
				}
			}
		}
	}
	switch l.name {
	case "go":
		collect(def.ChildByFieldName("receiver", l.language))
		collect(def.ChildByFieldName("parameters", l.language))
		collect(def.ChildByFieldName("result", l.language))
	case "python":
		collect(def.ChildByFieldName("parameters", l.language))
	case "ruby":
		collect(def.ChildByFieldName("parameters", l.language))
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
			top := stack[len(stack)-1]
			if decls[top].Start == decls[i].Start && decls[top].End == decls[i].End {
				decls[i].Parent = decls[top].Parent
			} else {
				decls[i].Parent = top
			}
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
