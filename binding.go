package outline

import ts "github.com/odvcencio/gotreesitter"

type binding struct {
	Name  string
	In    int
	Start uint32
	End   uint32
	Decl  int
}

func bindingsFor(src []byte, l *lang, root *ts.Node, decls []decl) []binding {
	if l.name != "go" && l.name != "python" {
		return nil
	}
	var out []binding
	walkNamed(root, func(node *ts.Node) {
		names, scope, start := bindingNode(src, l, node)
		if scope == nil || len(names) == 0 {
			return
		}
		in := enclosing(decls, scope.StartByte())
		target := bindingDeclaration(l, node, decls)
		for _, name := range names {
			if l.name != "go" || name != "_" {
				out = append(out, binding{Name: name, In: in, Start: start, End: scope.EndByte(), Decl: target})
			}
		}
	})
	return out
}

func bindingNode(src []byte, l *lang, node *ts.Node) ([]string, *ts.Node, uint32) {
	switch l.name {
	case "go":
		names := goBindingNames(src, l.language, node)
		if len(names) > 0 {
			return names, goBindingScope(node, l.language), node.EndByte()
		}
	case "python":
		switch node.Type(l.language) {
		case "assignment", "augmented_assignment", "for_statement":
			names := bindingNames(src, l.language, node.ChildByFieldName("left", l.language))
			if len(names) > 0 {
				if scope := pythonBindingScope(node, l.language); scope != nil {
					return names, scope, scope.StartByte()
				}
			}
		}
	}
	return nil, nil, 0
}

func bindingDeclaration(l *lang, node *ts.Node, decls []decl) int {
	if l.name == "go" {
		switch node.Type(l.language) {
		case "type_spec", "type_alias":
			return enclosing(decls, node.StartByte())
		}
	}
	return -1
}

func bindingNames(src []byte, language *ts.Language, node *ts.Node) []string {
	if node == nil {
		return nil
	}
	switch node.Type(language) {
	case "identifier", "type_identifier":
		return []string{node.Text(src)}
	case "expression_list", "pattern_list", "tuple_pattern", "list_pattern", "tuple", "list", "list_splat_pattern":
		var out []string
		for i := range node.NamedChildCount() {
			out = append(out, bindingNames(src, language, node.NamedChild(i))...)
		}
		return out
	}
	return nil
}

func goBindingNames(src []byte, language *ts.Language, node *ts.Node) []string {
	switch node.Type(language) {
	case "short_var_declaration":
		return bindingNames(src, language, node.ChildByFieldName("left", language))
	case "type_spec", "type_alias":
		return bindingNames(src, language, node.ChildByFieldName("name", language))
	case "range_clause":
		for i := range node.ChildCount() {
			if node.Child(i).Type(language) == ":=" {
				return bindingNames(src, language, node.ChildByFieldName("left", language))
			}
		}
	case "var_spec", "const_spec":
		var out []string
		for i := range node.NamedChildCount() {
			child := node.NamedChild(i)
			if child.Type(language) == "identifier" {
				out = append(out, child.Text(src))
			}
		}
		return out
	}
	return nil
}

func goBindingScope(node *ts.Node, language *ts.Language) *ts.Node {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Type(language) {
		case "block", "if_statement", "for_statement", "expression_switch_statement", "type_switch_statement", "communication_case", "expression_case", "type_case":
			return parent
		}
	}
	return nil
}

func pythonBindingScope(node *ts.Node, language *ts.Language) *ts.Node {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Type(language) {
		case "function_definition":
			return parent
		case "class_definition":
			return nil
		}
	}
	return nil
}
