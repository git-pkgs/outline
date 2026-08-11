package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

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
		receiver := value[:index]
		if wanted[receiver] {
			refs = append(refs, Ref{
				Receiver: receiver,
				Member:   value[index+1:],
				Line:     sourceLine(node),
			})
		}
	})
	return refs
}
