package outline

import ts "github.com/odvcencio/gotreesitter"

// Call is one call site. Receiver is the leftmost identifier of a member
// call (a.b()) or empty for a bare call (b()). In is the index into the
// file's decl slice of the innermost enclosing declaration, or -1 for
// top-level calls.
type Call struct {
	Receiver string
	Name     string
	Line     int
	Start    uint32
	In       int
}

func callsFor(src []byte, l *lang, root *ts.Node, decls []decl) ([]Call, bool) {
	var calls []Call
	switch l.name {
	case "go":
		calls = goCalls(src, l.language, root)
	case "python":
		calls = pythonCalls(src, l.language, root)
	default:
		return nil, false
	}
	for i := range calls {
		calls[i].In = enclosing(decls, calls[i].Start)
	}
	return calls, true
}

func goCalls(src []byte, language *ts.Language, root *ts.Node) []Call {
	var calls []Call
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "call_expression" {
			return
		}
		fn := node.ChildByFieldName("function", language)
		if fn == nil {
			return
		}
		c := Call{Line: sourceLine(node), Start: node.StartByte()}
		switch fn.Type(language) {
		case "identifier":
			c.Name = fn.Text(src)
		case "selector_expression":
			recv := fn.ChildByFieldName("operand", language)
			member := fn.ChildByFieldName("field", language)
			if member == nil {
				return
			}
			c.Name = member.Text(src)
			if recv != nil && recv.Type(language) == "identifier" {
				c.Receiver = recv.Text(src)
			}
		default:
			return
		}
		calls = append(calls, c)
	})
	return calls
}

func pythonCalls(src []byte, language *ts.Language, root *ts.Node) []Call {
	var calls []Call
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "call" {
			return
		}
		fn := node.ChildByFieldName("function", language)
		if fn == nil {
			return
		}
		c := Call{Line: sourceLine(node), Start: node.StartByte()}
		switch fn.Type(language) {
		case "identifier":
			c.Name = fn.Text(src)
		case "attribute":
			recv := fn.ChildByFieldName("object", language)
			member := fn.ChildByFieldName("attribute", language)
			if member == nil {
				return
			}
			c.Name = member.Text(src)
			c.Receiver = leftmostIdentifier(src, language, recv)
		default:
			return
		}
		calls = append(calls, c)
	})
	return calls
}

// leftmostIdentifier walks a chained attribute expression (a.b.c) and
// returns the base identifier text, or "" if the base is not a plain name.
func leftmostIdentifier(src []byte, language *ts.Language, node *ts.Node) string {
	for node != nil {
		switch node.Type(language) {
		case "identifier":
			return node.Text(src)
		case "attribute":
			node = node.ChildByFieldName("object", language)
		default:
			return ""
		}
	}
	return ""
}
