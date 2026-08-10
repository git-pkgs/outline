package outline

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	ts "github.com/odvcencio/gotreesitter"
)

// Symbol is a top-level declaration captured while outlining a file.
type Symbol struct {
	Name     string
	Kind     string // func, type, class, const, or var
	Line     int    // one-based source line
	Exported bool
}

type parsedSymbol struct {
	Symbol
	start uint32
}

func extractSymbols(src []byte, l *lang, root *ts.Node, matches []ts.QueryMatch) []Symbol {
	parsed := make([]parsedSymbol, 0)
	for _, match := range matches {
		parsed = append(parsed, symbolsFromMatch(src, l, match)...)
	}

	applyExplicitExports(parsed, l, root, src)

	sort.SliceStable(parsed, func(i, j int) bool {
		return parsed[i].start < parsed[j].start
	})
	return deduplicateSymbols(parsed)
}

func symbolsFromMatch(src []byte, l *lang, match ts.QueryMatch) []parsedSymbol {
	var definition *ts.Node
	var kind string
	exported := false
	names := make([]*ts.Node, 0, 1)
	for _, capture := range match.Captures {
		switch capture.Name {
		case "symbol.name":
			names = append(names, capture.Node)
		case "symbol.exported":
			exported = true
		case "symbol.func", "symbol.type", "symbol.class", "symbol.const", "symbol.var":
			definition = capture.Node
			kind = strings.TrimPrefix(capture.Name, "symbol.")
		}
	}
	if definition == nil || kind == "" || !topLevelSymbol(definition, l.language) {
		return nil
	}

	parsed := make([]parsedSymbol, 0, len(names))
	for _, nameNode := range names {
		if nameNode == nil {
			continue
		}
		name := strings.TrimSpace(nameNode.Text(src))
		if name == "" || name == "_" {
			continue
		}
		parsed = append(parsed, parsedSymbol{
			Symbol: Symbol{
				Name:     name,
				Kind:     normalizeSymbolKind(l.name, kind, name, definition, src, l.language),
				Line:     int(nameNode.StartPoint().Row) + 1,
				Exported: exported || symbolExported(l.name, name, definition, src, l.language),
			},
			start: nameNode.StartByte(),
		})
	}
	return parsed
}

func applyExplicitExports(parsed []parsedSymbol, l *lang, root *ts.Node, src []byte) {
	var exported map[string]bool
	merge := false
	switch l.name {
	case "javascript", "typescript":
		exported = javascriptExports(root, src, l.language)
		merge = true
	case "erlang":
		exported = exportsFromNodes(root, src, l.language, "export_attribute", "export_type_attribute", "atom")
	case "julia":
		exported = exportsFromNodes(root, src, l.language, "export_statement", "public_statement", "identifier")
	default:
		return
	}
	for i := range parsed {
		if merge {
			parsed[i].Exported = parsed[i].Exported || exported[parsed[i].Name]
		} else {
			parsed[i].Exported = exported[parsed[i].Name]
		}
	}
}

func deduplicateSymbols(parsed []parsedSymbol) []Symbol {
	type symbolKey struct {
		name string
		line int
	}
	seen := make(map[symbolKey]int, len(parsed))
	symbols := make([]Symbol, 0, len(parsed))
	for _, item := range parsed {
		key := symbolKey{name: item.Name, line: item.Line}
		if index, ok := seen[key]; ok {
			symbols[index].Exported = symbols[index].Exported || item.Exported
			if symbolKindRank(item.Kind) > symbolKindRank(symbols[index].Kind) {
				symbols[index].Kind = item.Kind
			}
			continue
		}
		seen[key] = len(symbols)
		symbols = append(symbols, item.Symbol)
	}
	return symbols
}

func topLevelSymbol(node *ts.Node, language *ts.Language) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Parent() == nil {
			return true
		}
		switch parent.Type(language) {
		case "ambient_declaration", "body", "const_declaration", "declarations", "decorated_definition", "export_statement", "expression_statement", "named_module", "template_declaration", "type_declaration", "var_declaration":
			continue
		default:
			return false
		}
	}
	return false
}

func symbolExported(language, name string, definition *ts.Node, src []byte, grammar *ts.Language) bool {
	text := strings.TrimSpace(definition.Text(src))
	switch language {
	case "go":
		r, _ := utf8.DecodeRuneInString(name)
		return unicode.IsUpper(r)
	case "javascript", "typescript":
		return ancestorType(definition, "export_statement", grammar)
	case "python", "dart", "starlark":
		return !strings.HasPrefix(name, "_")
	case "rust", "zig":
		return startsWithWord(text, "pub") && !strings.HasPrefix(text, "pub(")
	case "java", "csharp":
		return containsWord(text, "public")
	case "swift":
		return containsWord(text, "public") || containsWord(text, "open")
	case "elixir":
		return startsWithWord(text, "defmodule") ||
			startsWithWord(text, "defprotocol") ||
			startsWithWord(text, "defimpl") ||
			startsWithWord(text, "def") &&
				!startsWithWord(text, "defp") &&
				!startsWithWord(text, "defmacrop") &&
				!startsWithWord(text, "defguardp")
	case "clojure":
		return !strings.HasPrefix(text, "(defn-")
	case "nim":
		return false
	case "kotlin", "scala", "d", "fsharp", "groovy":
		return !containsWord(text, "private") &&
			!containsWord(text, "protected") &&
			!containsWord(text, "internal")
	case "c", "cpp":
		return !containsWord(text, "static")
	case "julia", "erlang":
		return false
	default:
		return true
	}
}

func normalizeSymbolKind(language, kind, name string, definition *ts.Node, src []byte, grammar *ts.Language) string {
	text := strings.TrimSpace(definition.Text(src))
	switch language {
	case "kotlin":
		return normalizeKotlinKind(kind, text)
	case "swift":
		return normalizeSwiftKind(kind, text)
	case "zig":
		return normalizeZigKind(kind, text, definition, grammar)
	case "fsharp":
		return normalizeFSharpKind(kind, name, text)
	}
	return kind
}

func normalizeKotlinKind(kind, text string) string {
	if kind == "class" && (containsWord(text, "interface") || containsWord(text, "enum")) {
		return "type"
	}
	if kind == "var" && containsWord(text, "val") {
		return "const"
	}
	return kind
}

func normalizeSwiftKind(kind, text string) string {
	if kind == "class" && (containsWord(text, "struct") || containsWord(text, "enum")) {
		return "type"
	}
	if kind == "var" && containsWord(text, "let") {
		return "const"
	}
	return kind
}

func normalizeZigKind(kind, text string, definition *ts.Node, grammar *ts.Language) string {
	if kind != "var" {
		return kind
	}
	if hasDescendantType(definition, grammar, "struct_declaration", "enum_declaration", "union_declaration", "opaque_declaration") {
		return "type"
	}
	if containsWord(text, "const") {
		return "const"
	}
	return kind
}

func normalizeFSharpKind(kind, name, text string) string {
	if kind != "var" {
		return kind
	}
	beforeEquals, _, _ := strings.Cut(text, "=")
	index := strings.Index(beforeEquals, name)
	if index < 0 {
		return kind
	}
	remainder := strings.TrimSpace(beforeEquals[index+len(name):])
	if remainder != "" && !strings.HasPrefix(remainder, ":") {
		return "func"
	}
	return kind
}

func symbolKindRank(kind string) int {
	const definitionRank = 2
	switch kind {
	case "func", "type", "class":
		return definitionRank
	case "const":
		return 1
	default:
		return 0
	}
}

func hasDescendantType(node *ts.Node, language *ts.Language, types ...string) bool {
	if node == nil {
		return false
	}
	wanted := make(map[string]bool, len(types))
	for _, nodeType := range types {
		wanted[nodeType] = true
	}
	var visit func(*ts.Node) bool
	visit = func(current *ts.Node) bool {
		if wanted[current.Type(language)] {
			return true
		}
		for i := range current.NamedChildCount() {
			if visit(current.NamedChild(i)) {
				return true
			}
		}
		return false
	}
	return visit(node)
}

func ancestorType(node *ts.Node, want string, language *ts.Language) bool {
	if language == nil {
		return false
	}
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type(language) == want {
			return true
		}
	}
	return false
}

func startsWithWord(text, word string) bool {
	if !strings.HasPrefix(text, word) {
		return false
	}
	return len(text) == len(word) || !isIdentifierByte(text[len(word)])
}

func containsWord(text, word string) bool {
	for i := 0; i+len(word) <= len(text); i++ {
		if text[i:i+len(word)] != word {
			continue
		}
		if i > 0 && isIdentifierByte(text[i-1]) {
			continue
		}
		if i+len(word) < len(text) && isIdentifierByte(text[i+len(word)]) {
			continue
		}
		return true
	}
	return false
}

func isIdentifierByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func javascriptExports(root *ts.Node, src []byte, language *ts.Language) map[string]bool {
	exported := make(map[string]bool)
	if root == nil {
		return exported
	}
	for i := range root.NamedChildCount() {
		node := root.NamedChild(i)
		switch node.Type(language) {
		case "export_statement":
			collectJavascriptExportStatement(exported, node, src, language)
		case "expression_statement":
			collectCommonJSExport(exported, node, src, language)
		}
	}
	return exported
}

func collectJavascriptExportStatement(exported map[string]bool, node *ts.Node, src []byte, language *ts.Language) {
	if hasDirectChildType(node, "string", language) {
		return
	}
	for i := range node.NamedChildCount() {
		child := node.NamedChild(i)
		switch child.Type(language) {
		case "identifier":
			exported[child.Text(src)] = true
		case "export_clause":
			for j := range child.NamedChildCount() {
				specifier := child.NamedChild(j)
				if specifier.NamedChildCount() > 0 {
					exported[specifier.NamedChild(0).Text(src)] = true
				}
			}
		}
	}
}

func collectCommonJSExport(exported map[string]bool, statement *ts.Node, src []byte, language *ts.Language) {
	if statement.NamedChildCount() == 0 {
		return
	}
	assignment := statement.NamedChild(0)
	if assignment.Type(language) != "assignment_expression" || assignment.NamedChildCount() < 2 {
		return
	}
	left := strings.ReplaceAll(assignment.NamedChild(0).Text(src), " ", "")
	if left != "module.exports" && !strings.HasPrefix(left, "module.exports.") && !strings.HasPrefix(left, "exports.") {
		return
	}
	collectJavascriptExportValue(exported, assignment.NamedChild(1), src, language)
}

func collectJavascriptExportValue(exported map[string]bool, value *ts.Node, src []byte, language *ts.Language) {
	switch value.Type(language) {
	case "identifier", "shorthand_property_identifier":
		exported[value.Text(src)] = true
	case "object":
		for i := range value.NamedChildCount() {
			child := value.NamedChild(i)
			switch child.Type(language) {
			case "shorthand_property_identifier":
				exported[child.Text(src)] = true
			case "pair":
				if child.NamedChildCount() > 1 {
					collectJavascriptExportValue(exported, child.NamedChild(1), src, language)
				}
			}
		}
	}
}

func hasDirectChildType(node *ts.Node, want string, language *ts.Language) bool {
	for i := range node.NamedChildCount() {
		if node.NamedChild(i).Type(language) == want {
			return true
		}
	}
	return false
}

func exportsFromNodes(root *ts.Node, src []byte, language *ts.Language, statementType, alternateType, nameType string) map[string]bool {
	exported := make(map[string]bool)
	if root == nil {
		return exported
	}
	var collectNames func(*ts.Node)
	collectNames = func(node *ts.Node) {
		if node.Type(language) == nameType {
			exported[node.Text(src)] = true
			return
		}
		for i := range node.NamedChildCount() {
			collectNames(node.NamedChild(i))
		}
	}
	for i := range root.NamedChildCount() {
		node := root.NamedChild(i)
		if node.Type(language) == statementType || node.Type(language) == alternateType {
			collectNames(node)
		}
	}
	return exported
}
