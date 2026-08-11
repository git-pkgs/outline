package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

const javascriptCallExpression = "call_expression"

// Imports returns structured imports from one source file. The second return
// is false when the file's language or syntax tree is unsupported.
func Imports(src []byte, filename string) ([]Import, bool) {
	l, tree, ok := parseSource(src, filename)
	if !ok {
		return nil, false
	}
	defer tree.Release()

	switch l.name {
	case "go":
		return goImports(src, l.language, tree.RootNode()), true
	case "ruby":
		return rubyImports(src, l.language, tree.RootNode()), true
	case "python":
		return pythonImports(src, l.language, tree.RootNode()), true
	case "javascript", "typescript":
		return javascriptImports(src, l.language, tree.RootNode()), true
	case "rust":
		return rustImports(src, l.language, tree.RootNode()), true
	case "php":
		return phpImports(src, l.language, tree.RootNode()), true
	case "elixir":
		return elixirImports(src, l.language, tree.RootNode()), true
	default:
		return nil, false
	}
}

func pythonImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "import_statement":
			imports = append(imports, pythonModuleImports(src, language, node)...)
		case "import_from_statement", "future_import_statement":
			if imported, ok := pythonFromImport(src, language, node); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func pythonModuleImports(src []byte, language *ts.Language, statement *ts.Node) []Import {
	imports := make([]Import, 0, statement.NamedChildCount())
	for i := range statement.NamedChildCount() {
		child := statement.NamedChild(i)
		module := ""
		alias := ""
		switch child.Type(language) {
		case "aliased_import":
			if child.NamedChildCount() > 0 {
				module = child.NamedChild(0).Text(src)
			}
			if child.NamedChildCount() > 1 {
				alias = child.NamedChild(child.NamedChildCount() - 1).Text(src)
			}
		case "dotted_name", "identifier":
			module = child.Text(src)
		}
		if module == "" {
			continue
		}
		if alias == "" {
			alias, _, _ = strings.Cut(module, ".")
		}
		imports = append(imports, Import{
			Module: module,
			Kind:   ImportModule,
			Names:  []Name{{Alias: alias}},
			Line:   sourceLine(statement),
		})
	}
	return imports
}

func pythonFromImport(
	src []byte,
	language *ts.Language,
	statement *ts.Node,
) (Import, bool) {
	if statement.NamedChildCount() == 0 {
		return Import{}, false
	}
	moduleNode := statement.NamedChild(0)
	module := moduleNode.Text(src)
	firstName := 1
	if statement.Type(language) == "future_import_statement" {
		module = "__future__"
		firstName = 0
	}
	if module == "" {
		return Import{}, false
	}

	imported := Import{Module: module, Kind: ImportNamed, Line: sourceLine(statement)}
	for i := firstName; i < statement.NamedChildCount(); i++ {
		child := statement.NamedChild(i)
		switch child.Type(language) {
		case "wildcard_import":
			imported.Kind = ImportWildcard
			imported.Names = nil
			return imported, true
		case "aliased_import":
			if child.NamedChildCount() == 0 {
				continue
			}
			name := Name{Name: child.NamedChild(0).Text(src)}
			if child.NamedChildCount() > 1 {
				name.Alias = child.NamedChild(child.NamedChildCount() - 1).Text(src)
			}
			imported.Names = append(imported.Names, name)
		case "dotted_name", "identifier":
			imported.Names = append(imported.Names, Name{Name: child.Text(src)})
		}
	}
	return imported, len(imported.Names) > 0
}

func javascriptImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "import_statement":
			imports = append(imports, javascriptImportStatement(src, language, node)...)
		case "variable_declarator":
			if imported, ok := javascriptRequireDeclarator(src, language, node); ok {
				imports = append(imports, imported)
			}
		case javascriptCallExpression:
			if imported, ok := javascriptStandaloneRequire(src, language, node); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func javascriptImportStatement(
	src []byte,
	language *ts.Language,
	statement *ts.Node,
) []Import {
	var clause *ts.Node
	var module string
	for i := range statement.NamedChildCount() {
		child := statement.NamedChild(i)
		switch child.Type(language) {
		case "import_clause":
			clause = child
		case "string":
			module = sourceString(child.Text(src))
		}
	}
	if module == "" {
		return nil
	}
	line := sourceLine(statement)
	if clause == nil {
		return []Import{{Module: module, Kind: ImportSideEffect, Line: line}}
	}

	var imports []Import
	for i := range clause.NamedChildCount() {
		child := clause.NamedChild(i)
		switch child.Type(language) {
		case "identifier":
			imports = append(imports, Import{
				Module: module,
				Kind:   ImportDefault,
				Names:  []Name{{Alias: child.Text(src)}},
				Line:   line,
			})
		case "namespace_import":
			if child.NamedChildCount() > 0 {
				imports = append(imports, Import{
					Module: module,
					Kind:   ImportNamespace,
					Names:  []Name{{Alias: child.NamedChild(child.NamedChildCount() - 1).Text(src)}},
					Line:   line,
				})
			}
		case "named_imports":
			names := javascriptNamedImports(src, language, child)
			if len(names) > 0 {
				imports = append(imports, Import{Module: module, Kind: ImportNamed, Names: names, Line: line})
			}
		}
	}
	return imports
}

func javascriptNamedImports(src []byte, language *ts.Language, imports *ts.Node) []Name {
	var names []Name
	for i := range imports.NamedChildCount() {
		specifier := imports.NamedChild(i)
		if specifier.Type(language) != "import_specifier" || specifier.NamedChildCount() == 0 {
			continue
		}
		name := Name{Name: specifier.NamedChild(0).Text(src)}
		if specifier.NamedChildCount() > 1 {
			name.Alias = specifier.NamedChild(specifier.NamedChildCount() - 1).Text(src)
		}
		names = append(names, name)
	}
	return names
}

func javascriptRequireDeclarator(
	src []byte,
	language *ts.Language,
	declarator *ts.Node,
) (Import, bool) {
	nameNode := declarator.ChildByFieldName("name", language)
	valueNode := declarator.ChildByFieldName("value", language)
	if nameNode == nil && declarator.NamedChildCount() > 0 {
		nameNode = declarator.NamedChild(0)
	}
	if valueNode == nil && declarator.NamedChildCount() > 1 {
		valueNode = declarator.NamedChild(1)
	}
	module, member, ok := javascriptRequireValue(src, language, valueNode)
	if !ok || nameNode == nil {
		return Import{}, false
	}

	imported := Import{Module: module, Line: sourceLine(declarator)}
	switch nameNode.Type(language) {
	case "identifier":
		alias := nameNode.Text(src)
		if member != "" {
			imported.Kind = ImportNamed
			imported.Names = []Name{{Name: member, Alias: alias}}
		} else {
			imported.Kind = ImportModule
			imported.Names = []Name{{Alias: alias}}
		}
	case "object_pattern":
		imported.Kind = ImportNamed
		imported.Names = javascriptObjectPatternNames(src, language, nameNode)
	default:
		return Import{}, false
	}
	return imported, len(imported.Names) > 0
}

func javascriptObjectPatternNames(src []byte, language *ts.Language, pattern *ts.Node) []Name {
	var names []Name
	for i := range pattern.NamedChildCount() {
		child := pattern.NamedChild(i)
		switch child.Type(language) {
		case "shorthand_property_identifier_pattern", "identifier":
			names = append(names, Name{Name: child.Text(src)})
		case "pair_pattern":
			if child.NamedChildCount() == 0 {
				continue
			}
			name := Name{Name: child.NamedChild(0).Text(src)}
			if child.NamedChildCount() > 1 {
				name.Alias = child.NamedChild(child.NamedChildCount() - 1).Text(src)
			}
			names = append(names, name)
		}
	}
	return names
}

func javascriptRequireValue(
	src []byte,
	language *ts.Language,
	value *ts.Node,
) (module, member string, ok bool) {
	if value == nil {
		return "", "", false
	}
	if value.Type(language) == javascriptCallExpression {
		module, ok = javascriptRequireCall(src, language, value)
		return module, "", ok
	}
	if value.Type(language) != "member_expression" || value.NamedChildCount() < 2 {
		return "", "", false
	}
	object := value.ChildByFieldName("object", language)
	property := value.ChildByFieldName("property", language)
	if object == nil {
		object = value.NamedChild(0)
	}
	if property == nil {
		property = value.NamedChild(value.NamedChildCount() - 1)
	}
	module, ok = javascriptRequireCall(src, language, object)
	if !ok {
		return "", "", false
	}
	return module, property.Text(src), true
}

func javascriptStandaloneRequire(
	src []byte,
	language *ts.Language,
	call *ts.Node,
) (Import, bool) {
	module, ok := javascriptRequireCall(src, language, call)
	if !ok || javascriptRequireHandledByDeclarator(src, language, call) {
		return Import{}, false
	}
	if parent := call.Parent(); parent != nil && parent.Type(language) == "member_expression" {
		object := parent.ChildByFieldName("object", language)
		if object == nil && parent.NamedChildCount() > 0 {
			object = parent.NamedChild(0)
		}
		if object == call {
			property := parent.ChildByFieldName("property", language)
			if property == nil && parent.NamedChildCount() > 1 {
				property = parent.NamedChild(parent.NamedChildCount() - 1)
			}
			if property != nil {
				return Import{
					Module: module,
					Kind:   ImportNamed,
					Names:  []Name{{Name: property.Text(src)}},
					Line:   sourceLine(call),
				}, true
			}
		}
	}
	return Import{Module: module, Kind: ImportSideEffect, Line: sourceLine(call)}, true
}

func javascriptRequireHandledByDeclarator(src []byte, language *ts.Language, call *ts.Node) bool {
	declarator := ancestorNode(call, "variable_declarator", language)
	if declarator == nil {
		return false
	}
	value := declarator.ChildByFieldName("value", language)
	if value == nil && declarator.NamedChildCount() > 1 {
		value = declarator.NamedChild(1)
	}
	_, _, ok := javascriptRequireValue(src, language, value)
	return ok
}

func javascriptRequireCall(src []byte, language *ts.Language, call *ts.Node) (string, bool) {
	if call == nil || call.Type(language) != javascriptCallExpression || call.NamedChildCount() < minimumMemberChildren {
		return "", false
	}
	function := call.ChildByFieldName("function", language)
	arguments := call.ChildByFieldName("arguments", language)
	if function == nil {
		function = call.NamedChild(0)
	}
	if arguments == nil {
		arguments = call.NamedChild(1)
	}
	if function.Type(language) != "identifier" || function.Text(src) != "require" || arguments == nil {
		return "", false
	}
	for i := range arguments.NamedChildCount() {
		argument := arguments.NamedChild(i)
		if argument.Type(language) == "string" {
			module := sourceString(argument.Text(src))
			return module, module != ""
		}
	}
	return "", false
}

func sourceString(value string) string {
	if len(value) < quotedStringOverhead {
		return ""
	}
	first := value[0]
	last := value[len(value)-1]
	if first == last && (first == '\'' || first == '"' || first == '`') {
		return value[1 : len(value)-1]
	}
	return value
}
