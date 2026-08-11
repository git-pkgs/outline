package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

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
			member = node.NamedChild(node.NamedChildCount() - 1)
		case "call":
			if node.NamedChild(0).Type(language) == "constant" {
				receiver = node.NamedChild(0)
				member = node.NamedChild(node.NamedChildCount() - 1)
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
