package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// goRefs matches package selectors in both value and type position:
// selector_expression covers `pkg.Func()` and `pkg.Var`, qualified_type
// covers `pkg.Type{}` and `var x pkg.Type`.
func goRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	selectors := memberRefsWithFields(
		src, language, root, wanted,
		"selector_expression", "operand", "field", "identifier",
	)
	types := memberRefsWithFields(
		src, language, root, wanted,
		"qualified_type", "package", "name", "package_identifier",
	)
	return append(selectors, types...)
}

func rubyRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.NamedChildCount() < minimumMemberChildren {
			return
		}
		var receiver *ts.Node
		var member *ts.Node
		switch node.Type(language) {
		case "scope_resolution":
			receiver = node.NamedChild(0)
			member = node.ChildByFieldName("name", language)
		case "call":
			// The last named child of a call may be an argument list or a
			// block; the method identifier is the named `method` field.
			receiver = node.ChildByFieldName("receiver", language)
			member = node.ChildByFieldName("method", language)
			if receiver == nil || receiver.Type(language) != "constant" {
				return
			}
		}
		if receiver == nil || member == nil || !wanted[receiver.Text(src)] {
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

func phpRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "qualified_name" || ancestorNodeType(node, "namespace_use_declaration", language) {
			return
		}
		value := strings.TrimPrefix(node.Text(src), `\`)
		parts := strings.Split(value, `\`)
		if len(parts) < 2 || !wanted[parts[0]] {
			return
		}
		refs = append(refs, Ref{Receiver: parts[0], Member: parts[1], Line: sourceLine(node)})
	})
	return refs
}
