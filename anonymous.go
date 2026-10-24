package outline

import (
	"strconv"

	ts "github.com/odvcencio/gotreesitter"
)

func anonymousName(kind string, start uint32) string {
	return "<" + kind + "@" + strconv.FormatUint(uint64(start), 10) + ">"
}

func anonymousDecls(src []byte, l *lang, root *ts.Node) []decl {
	if l.name != "go" && l.name != "python" && l.name != "ruby" {
		return nil
	}
	var out []decl
	walkNamed(root, func(node *ts.Node) {
		kind := ""
		definition := node
		switch l.name {
		case "go":
			if node.Type(l.language) == "func_literal" {
				kind = "func"
			}
		case "python":
			if node.Type(l.language) == "lambda" {
				kind = "lambda"
			}
		case "ruby":
			if node.Type(l.language) == "block" || node.Type(l.language) == "do_block" {
				kind = "block"
				if parent := node.Parent(); parent != nil && parent.Type(l.language) == "lambda" {
					definition = parent
					kind = "lambda"
				}
			}
		}
		if kind == "" {
			return
		}
		bodyStart := node.EndByte()
		if body := node.ChildByFieldName("body", l.language); body != nil {
			bodyStart = body.StartByte()
		}
		callStart := bodyStart
		if l.name == "ruby" {
			if params := definition.ChildByFieldName("parameters", l.language); params != nil {
				callStart = params.StartByte()
			}
		}
		out = append(out, decl{
			Name: anonymousName(kind, definition.StartByte()), Kind: KindFunc,
			Line: sourceLine(definition), NameAt: definition.StartByte(),
			Start: definition.StartByte(), End: definition.EndByte(), SigEnd: bodyStart,
			Parent: -1, Params: extractParams(src, l, definition),
			Anonymous: true, BodyStart: callStart,
		})
	})
	return out
}

func callEnclosing(decls []decl, pos uint32) int {
	best := -1
	for i, d := range decls {
		if d.Start > pos {
			break
		}
		// Lambda defaults execute outside their anonymous body.
		if d.End > pos && (!d.Anonymous || pos >= d.BodyStart) {
			best = i
		}
	}
	return best
}

func callFunction(node *ts.Node, language *ts.Language) *ts.Node {
	for node != nil && node.Type(language) == "parenthesized_expression" && node.NamedChildCount() == 1 {
		node = node.NamedChild(0)
	}
	return node
}
