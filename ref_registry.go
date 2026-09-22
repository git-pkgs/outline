package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

func javaRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		var member *ts.Node
		switch node.Type(language) {
		case "method_invocation":
			member = node.ChildByFieldName("name", language)
		case "field_access":
			member = node.ChildByFieldName("field", language)
		default:
			return
		}
		receiver := node.ChildByFieldName("object", language)
		if receiver == nil || member == nil || receiver.Type(language) != "identifier" || !wanted[receiver.Text(src)] {
			return
		}
		refs = append(refs, Ref{Receiver: receiver.Text(src), Member: member.Text(src), Line: sourceLine(member)})
	})
	return refs
}

func kotlinRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "navigation_expression" {
			return
		}
		receiver := firstDescendantType(node, language, "simple_identifier")
		if receiver == nil || receiver.Parent() != node || !wanted[receiver.Text(src)] {
			return
		}
		for i := range node.NamedChildCount() {
			suffix := node.NamedChild(i)
			if suffix.Type(language) != "navigation_suffix" {
				continue
			}
			member := firstDescendantType(suffix, language, "simple_identifier")
			if member != nil {
				refs = append(refs, Ref{Receiver: receiver.Text(src), Member: member.Text(src), Line: sourceLine(member)})
			}
			return
		}
	})
	return refs
}

func csharpRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		var receiver, member *ts.Node
		switch node.Type(language) {
		case "member_access_expression":
			receiver = node.ChildByFieldName("expression", language)
			member = node.ChildByFieldName("name", language)
		case "conditional_access_expression":
			receiver = node.ChildByFieldName("condition", language)
			for i := range node.NamedChildCount() {
				binding := node.NamedChild(i)
				if binding.Type(language) == "member_binding_expression" {
					member = binding.ChildByFieldName("name", language)
				}
			}
		default:
			return
		}
		if receiver == nil || member == nil || receiver.Type(language) != "identifier" || !wanted[receiver.Text(src)] {
			return
		}
		if member.Type(language) == "generic_name" {
			member = firstDescendantType(member, language, "identifier")
			if member == nil {
				return
			}
		}
		refs = append(refs, Ref{Receiver: receiver.Text(src), Member: member.Text(src), Line: sourceLine(member)})
	})
	return refs
}

func dartRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		for i := 0; i+1 < node.NamedChildCount(); i++ {
			receiver := node.NamedChild(i)
			selector := node.NamedChild(i + 1)
			if receiver.Type(language) != "identifier" ||
				selector.Type(language) != "selector" ||
				!wanted[receiver.Text(src)] {
				continue
			}
			member := firstDescendantType(selector, language, "identifier")
			if member != nil {
				refs = append(refs, Ref{
					Receiver: receiver.Text(src),
					Member:   member.Text(src),
					Line:     sourceLine(member),
				})
			}
		}
	})
	return refs
}

func swiftRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "navigation_expression" {
			return
		}
		receiver := node.ChildByFieldName("target", language)
		suffix := node.ChildByFieldName("suffix", language)
		if receiver == nil || suffix == nil || !wanted[receiver.Text(src)] {
			return
		}
		member := firstDescendantType(suffix, language, "simple_identifier")
		if member != nil {
			refs = append(refs, Ref{
				Receiver: receiver.Text(src),
				Member:   member.Text(src),
				Line:     sourceLine(member),
			})
		}
	})
	return refs
}

func haskellRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "qualified" {
			return
		}
		module := node.ChildByFieldName("module", language)
		member := node.ChildByFieldName("id", language)
		if module == nil || member == nil {
			return
		}
		receiver := strings.TrimSuffix(module.Text(src), ".")
		if wanted[receiver] {
			refs = append(refs, Ref{Receiver: receiver, Member: member.Text(src), Line: sourceLine(member)})
		}
	})
	return refs
}

func ocamlRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "value_path" {
			return
		}
		module := firstDescendantType(node, language, "module_path")
		member := firstDescendantType(node, language, "value_name")
		if module != nil && member != nil && wanted[module.Text(src)] {
			refs = append(refs, Ref{
				Receiver: module.Text(src),
				Member:   member.Text(src),
				Line:     sourceLine(member),
			})
		}
	})
	return refs
}

func dRefs(src []byte, language *ts.Language, root *ts.Node, wanted map[string]bool) []Ref {
	var refs []Ref
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "property_expression" {
			return
		}
		value := node.Text(src)
		index := strings.LastIndex(value, ".")
		if index <= 0 || index == len(value)-1 {
			return
		}
		receiver := strings.TrimSpace(value[:index])
		if wanted[receiver] {
			refs = append(refs, Ref{
				Receiver: receiver,
				Member:   strings.TrimSpace(value[index+1:]),
				Line:     sourceLine(node),
			})
		}
	})
	return refs
}
