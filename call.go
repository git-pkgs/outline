package outline

import ts "github.com/odvcencio/gotreesitter"

// ReceiverKind describes the syntax used to select a call receiver.
type ReceiverKind string

const (
	ReceiverBare       ReceiverKind = "bare"
	ReceiverConstant   ReceiverKind = "constant"
	ReceiverLocal      ReceiverKind = "local"
	ReceiverSelf       ReceiverKind = "self"
	ReceiverExpression ReceiverKind = "expression"
)

// DispatchKind describes how a call reaches its target.
type DispatchKind string

const (
	DispatchDirect   DispatchKind = "direct"
	DispatchSubshell DispatchKind = "subshell"
)

// Argument is one source argument to a call.
type Argument struct {
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	Start uint32 `json:"start"`
	End   uint32 `json:"end"`
}

// Call is one call site. In refers to the innermost enclosing declaration
// during graph construction and is omitted from JSON.
type Call struct {
	Receiver     string       `json:"receiver,omitempty"`
	ReceiverKind ReceiverKind `json:"receiver_kind"`
	Name         string       `json:"name"`
	Dispatch     DispatchKind `json:"dispatch"`
	Arguments    []Argument   `json:"arguments,omitempty"`
	Line         int          `json:"line"`
	Start        uint32       `json:"start"`
	End          uint32       `json:"end"`
	In           int          `json:"-"`
}

func callsFor(src []byte, l *lang, root *ts.Node, decls []decl) ([]Call, bool) {
	var calls []Call
	switch l.name {
	case "go":
		calls = goCalls(src, l.language, root)
	case "python":
		calls = pythonCalls(src, l.language, root)
	case "ruby":
		calls = rubyCalls(src, l.language, root)
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
		c := newCall(src, language, node)
		switch fn.Type(language) {
		case "identifier":
			c.Name = fn.Text(src)
			c.ReceiverKind = ReceiverBare
		case "selector_expression":
			recv := fn.ChildByFieldName("operand", language)
			member := fn.ChildByFieldName("field", language)
			if member == nil {
				return
			}
			c.Name = member.Text(src)
			if recv != nil && recv.Type(language) == "identifier" {
				c.Receiver = recv.Text(src)
				c.ReceiverKind = ReceiverLocal
			} else if recv != nil {
				c.Receiver = recv.Text(src)
				c.ReceiverKind = ReceiverExpression
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
		c := newCall(src, language, node)
		switch fn.Type(language) {
		case "identifier":
			c.Name = fn.Text(src)
			c.ReceiverKind = ReceiverBare
		case "attribute":
			recv := fn.ChildByFieldName("object", language)
			member := fn.ChildByFieldName("attribute", language)
			if member == nil {
				return
			}
			c.Name = member.Text(src)
			c.Receiver = leftmostIdentifier(src, language, recv)
			c.ReceiverKind = ReceiverLocal
			if c.Receiver == "" && recv != nil {
				c.Receiver = recv.Text(src)
				c.ReceiverKind = ReceiverExpression
			}
		default:
			return
		}
		calls = append(calls, c)
	})
	return calls
}

func rubyCalls(src []byte, language *ts.Language, root *ts.Node) []Call {
	var calls []Call
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "subshell":
			calls = append(calls, Call{
				Receiver:     "Kernel",
				ReceiverKind: ReceiverConstant,
				Name:         "`",
				Dispatch:     DispatchSubshell,
				Line:         sourceLine(node),
				Start:        node.StartByte(),
				End:          node.EndByte(),
			})
		case "call":
			method := node.ChildByFieldName("method", language)
			if method == nil {
				return
			}
			if _, ok := rubyImport(src, language, node); ok {
				return
			}
			name := method.Text(src)
			c := newCall(src, language, node)
			c.Name = name
			receiver := node.ChildByFieldName("receiver", language)
			if receiver == nil {
				c.ReceiverKind = ReceiverBare
			} else {
				c.Receiver = receiver.Text(src)
				switch receiver.Type(language) {
				case "constant", "scope_resolution":
					c.ReceiverKind = ReceiverConstant
				case "identifier":
					c.ReceiverKind = ReceiverLocal
				case "self":
					c.ReceiverKind = ReceiverSelf
				default:
					c.ReceiverKind = ReceiverExpression
				}
			}
			calls = append(calls, c)
		}
	})
	return calls
}

func newCall(src []byte, language *ts.Language, node *ts.Node) Call {
	c := Call{
		Dispatch: DispatchDirect,
		Line:     sourceLine(node),
		Start:    node.StartByte(),
		End:      node.EndByte(),
	}
	args := node.ChildByFieldName("arguments", language)
	if args == nil {
		return c
	}
	for i := range args.NamedChildCount() {
		arg := args.NamedChild(i)
		c.Arguments = append(c.Arguments, Argument{
			Kind:  arg.Type(language),
			Text:  arg.Text(src),
			Start: arg.StartByte(),
			End:   arg.EndByte(),
		})
	}
	return c
}

func rubyLoadMethod(name string) bool {
	switch name {
	case "require", "require_relative", "load", "autoload":
		return true
	default:
		return false
	}
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
