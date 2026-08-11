package outline

import ts "github.com/odvcencio/gotreesitter"

const (
	minimumMemberChildren = 2
	quotedStringOverhead  = 2
)

func walkNamed(node *ts.Node, visit func(*ts.Node)) {
	if node == nil {
		return
	}
	visit(node)
	for i := range node.NamedChildCount() {
		walkNamed(node.NamedChild(i), visit)
	}
}

func sourceLine(node *ts.Node) int {
	return int(node.StartPoint().Row) + 1
}

func ancestorNodeType(node *ts.Node, nodeType string, language *ts.Language) bool {
	return ancestorNode(node, nodeType, language) != nil
}

func ancestorNode(node *ts.Node, nodeType string, language *ts.Language) *ts.Node {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type(language) == nodeType {
			return parent
		}
	}
	return nil
}

func firstDescendantType(node *ts.Node, language *ts.Language, nodeType string) *ts.Node {
	if node == nil {
		return nil
	}
	if node.Type(language) == nodeType {
		return node
	}
	for i := range node.NamedChildCount() {
		if found := firstDescendantType(node.NamedChild(i), language, nodeType); found != nil {
			return found
		}
	}
	return nil
}

func descendantTexts(src []byte, language *ts.Language, node *ts.Node, nodeType string) []string {
	var values []string
	walkNamed(node, func(child *ts.Node) {
		if child.Type(language) == nodeType {
			values = append(values, child.Text(src))
		}
	})
	return values
}
