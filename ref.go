package outline

import ts "github.com/odvcencio/gotreesitter"

// Refs returns direct member accesses on the requested receiver identifiers.
// The second return is false when the file's language or syntax tree is
// unsupported.
func Refs(src []byte, filename string, receivers []string) ([]Ref, bool) {
	l, tree, ok := parseSource(src, filename)
	if !ok {
		return nil, false
	}
	defer tree.Release()

	wanted := make(map[string]bool, len(receivers))
	for _, receiver := range receivers {
		if receiver != "" {
			wanted[receiver] = true
		}
	}

	var refs []Ref
	switch l.name {
	case "go":
		refs = memberRefs(src, l.language, tree.RootNode(), wanted, "selector_expression", "field", "identifier")
	case "ruby":
		refs = rubyRefs(src, l.language, tree.RootNode(), wanted)
	case "python":
		refs = memberRefs(src, l.language, tree.RootNode(), wanted, "attribute", "attribute", "identifier")
	case "javascript", "typescript":
		refs = memberRefs(src, l.language, tree.RootNode(), wanted, "member_expression", "property", "identifier")
	case "rust":
		refs = memberRefs(src, l.language, tree.RootNode(), wanted, "scoped_identifier", "name", "identifier")
	case "php":
		refs = phpRefs(src, l.language, tree.RootNode(), wanted)
	case "elixir":
		refs = memberRefs(src, l.language, tree.RootNode(), wanted, "dot", "right", "alias")
	case "dart":
		refs = dartRefs(src, l.language, tree.RootNode(), wanted)
	case "swift":
		refs = swiftRefs(src, l.language, tree.RootNode(), wanted)
	case "haskell":
		refs = haskellRefs(src, l.language, tree.RootNode(), wanted)
	case "perl":
		refs = memberRefsWithFields(
			src, l.language, tree.RootNode(), wanted,
			"method_call_expression", "invocant", "method", "bareword",
		)
	case "lua":
		refs = memberRefsWithFields(
			src, l.language, tree.RootNode(), wanted,
			"dot_index_expression", "table", "field", "identifier",
		)
	case "r":
		refs = memberRefsWithFields(
			src, l.language, tree.RootNode(), wanted,
			"namespace_operator", "lhs", "rhs", "identifier",
		)
	case "julia":
		refs = memberRefsWithFields(
			src, l.language, tree.RootNode(), wanted,
			"field_expression", "value", "", "identifier",
		)
	case "ocaml":
		refs = ocamlRefs(src, l.language, tree.RootNode(), wanted)
	case "crystal":
		refs = memberRefsWithFields(
			src, l.language, tree.RootNode(), wanted,
			"call", "receiver", "method", "constant",
		)
	case "nim":
		refs = memberRefsWithFields(
			src, l.language, tree.RootNode(), wanted,
			"dot_expression", "left", "right", "identifier",
		)
	case "zig":
		refs = memberRefs(src, l.language, tree.RootNode(), wanted, "field_expression", "member", "identifier")
	case "d":
		refs = dRefs(src, l.language, tree.RootNode(), wanted)
	default:
		return nil, false
	}
	return refs, true
}

func memberRefs(
	src []byte,
	language *ts.Language,
	root *ts.Node,
	wanted map[string]bool,
	nodeType string,
	memberField string,
	receiverType string,
) []Ref {
	return memberRefsWithFields(
		src, language, root, wanted,
		nodeType, "object", memberField, receiverType,
	)
}

func memberRefsWithFields(
	src []byte,
	language *ts.Language,
	root *ts.Node,
	wanted map[string]bool,
	nodeType string,
	receiverField string,
	memberField string,
	receiverType string,
) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != nodeType || node.NamedChildCount() < 2 {
			return
		}
		receiver := node.ChildByFieldName(receiverField, language)
		var member *ts.Node
		if memberField != "" {
			member = node.ChildByFieldName(memberField, language)
		}
		if receiver == nil {
			receiver = node.NamedChild(0)
		}
		if member == nil {
			member = node.NamedChild(node.NamedChildCount() - 1)
		}
		if receiver.Type(language) != receiverType || !wanted[receiver.Text(src)] {
			return
		}
		refs = append(refs, Ref{
			Receiver: receiver.Text(src),
			Member:   member.Text(src),
			Line:     sourceLine(member),
		})
	})
	return refs
}
