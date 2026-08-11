package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

func dartImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "library_import" {
			return
		}
		specification := firstDescendantType(node, language, "import_specification")
		if specification == nil {
			return
		}

		alias := directChildText(src, language, specification, "identifier")
		var names []Name
		for i := range specification.NamedChildCount() {
			child := specification.NamedChild(i)
			if child.Type(language) != "combinator" {
				continue
			}
			hidden := strings.HasPrefix(strings.TrimSpace(child.Text(src)), "hide ")
			if !hidden {
				for _, value := range directChildTexts(src, language, child, "identifier") {
					names = append(names, Name{Name: value})
				}
			}
		}

		for _, literal := range descendantTexts(src, language, specification, "string_literal") {
			module := sourceString(literal)
			if module == "" {
				continue
			}
			imported := Import{Module: module, Kind: ImportWildcard, Line: sourceLine(node)}
			switch {
			case alias != "":
				imported.Kind = ImportNamespace
				imported.Names = []Name{{Alias: alias}}
			case len(names) > 0:
				imported.Kind = ImportNamed
				imported.Names = names
			}
			imports = append(imports, imported)
		}
	})
	return imports
}

func swiftImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "import_declaration" {
			return
		}
		value := node.Text(src)
		index := strings.Index(value, "import ")
		if index < 0 {
			return
		}
		fields := strings.Fields(value[index+len("import "):])
		if len(fields) == 0 {
			return
		}
		if swiftDeclarationImport(fields[0]) && len(fields) > 1 {
			index := strings.LastIndex(fields[1], ".")
			if index > 0 && index < len(fields[1])-1 {
				imports = append(imports, Import{
					Module: fields[1][:index],
					Kind:   ImportNamed,
					Names:  []Name{{Name: fields[1][index+1:]}},
					Line:   sourceLine(node),
				})
			}
			return
		}
		module := fields[0]
		imports = append(imports, Import{
			Module: module,
			Kind:   ImportModule,
			Names:  []Name{{Alias: module}},
			Line:   sourceLine(node),
		})
	})
	return imports
}

func swiftDeclarationImport(value string) bool {
	switch value {
	case "typealias", "struct", "class", "enum", "protocol", "let", "var", "func", "operator":
		return true
	default:
		return false
	}
}

func haskellImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "import" {
			return
		}
		moduleNode := node.ChildByFieldName("module", language)
		if moduleNode == nil {
			return
		}
		module := strings.TrimSpace(moduleNode.Text(src))
		imported := Import{Module: module, Kind: ImportWildcard, Line: sourceLine(node)}
		if strings.Contains(node.Text(src), "qualified") {
			alias := module
			if aliasNode := node.ChildByFieldName("alias", language); aliasNode != nil {
				alias = strings.TrimSpace(aliasNode.Text(src))
			}
			imported.Kind = ImportNamespace
			imported.Names = []Name{{Alias: alias}}
		} else if namesNode := node.ChildByFieldName("names", language); namesNode != nil &&
			!strings.Contains(node.Text(src), "hiding") {
			for _, value := range directChildTexts(src, language, namesNode, "import_name") {
				name := strings.TrimPrefix(strings.TrimPrefix(value, "type "), "pattern ")
				if name != "" {
					imported.Names = append(imported.Names, Name{Name: name})
				}
			}
			if len(imported.Names) > 0 {
				imported.Kind = ImportNamed
			}
		}
		imports = append(imports, imported)
	})
	return imports
}

func perlImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "use_statement":
			moduleNode := node.ChildByFieldName("module", language)
			if moduleNode == nil || moduleNode.Type(language) != "package" {
				return
			}
			imported := Import{Module: moduleNode.Text(src), Kind: ImportWildcard, Line: sourceLine(node)}
			if firstDescendantType(node, language, "stub_expression") != nil {
				imported.Kind = ImportSideEffect
			} else if words := firstDescendantType(node, language, "quoted_word_list"); words != nil {
				content := firstDescendantType(words, language, "string_content")
				if content != nil {
					for _, name := range strings.Fields(content.Text(src)) {
						imported.Names = append(imported.Names, Name{Name: name})
					}
				}
				if len(imported.Names) > 0 {
					imported.Kind = ImportNamed
				}
			}
			imports = append(imports, imported)
		case "require_expression":
			if module := perlRequiredModule(src, language, node); module != "" {
				imports = append(imports, Import{Module: module, Kind: ImportSideEffect, Line: sourceLine(node)})
			}
		}
	})
	return imports
}

func perlRequiredModule(src []byte, language *ts.Language, node *ts.Node) string {
	if bareword := firstDescendantType(node, language, "bareword"); bareword != nil {
		return bareword.Text(src)
	}
	if literal := firstDescendantType(node, language, "interpolated_string_literal"); literal != nil {
		return sourceString(literal.Text(src))
	}
	return ""
}

func luaImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "function_call" {
			return
		}
		name := node.ChildByFieldName("name", language)
		if name == nil || name.Type(language) != "identifier" || name.Text(src) != "require" {
			return
		}
		content := firstDescendantType(node.ChildByFieldName("arguments", language), language, "string_content")
		if content == nil || content.Text(src) == "" {
			return
		}
		imported := Import{Module: content.Text(src), Kind: ImportSideEffect, Line: sourceLine(node)}
		parent := node.Parent()
		if parent != nil && parent.Type(language) == "expression_list" && parent.NamedChildCount() == 1 {
			assignment := ancestorNode(node, "assignment_statement", language)
			if assignment == nil {
				imports = append(imports, imported)
				return
			}
			variables := firstDescendantType(assignment, language, "variable_list")
			if variables != nil && variables.NamedChildCount() == 1 {
				aliasNode := firstDescendantType(variables, language, "identifier")
				imported.Kind = ImportModule
				imported.Names = []Name{{Alias: aliasNode.Text(src)}}
			}
		}
		imports = append(imports, imported)
	})
	return imports
}

func rImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "call":
			if imported, ok := rLoadImport(src, language, node); ok {
				imports = append(imports, imported)
			}
		case "namespace_operator":
			if imported, ok := rNamespaceImport(src, language, node); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func rLoadImport(src []byte, language *ts.Language, call *ts.Node) (Import, bool) {
	function := call.ChildByFieldName("function", language)
	if function == nil || function.Type(language) != "identifier" {
		return Import{}, false
	}
	form := function.Text(src)
	if form != "library" && form != "require" && form != "requireNamespace" {
		return Import{}, false
	}
	arguments := call.ChildByFieldName("arguments", language)
	argument := firstDescendantType(arguments, language, "argument")
	if argument == nil || argument.NamedChildCount() == 0 {
		return Import{}, false
	}
	value := argument.NamedChild(0)
	module := value.Text(src)
	if value.Type(language) == "string" {
		module = sourceString(module)
	}
	if module == "" {
		return Import{}, false
	}
	imported := Import{Module: module, Kind: ImportWildcard, Line: sourceLine(call)}
	if form == "requireNamespace" {
		imported.Kind = ImportModule
		imported.Names = []Name{{Alias: module}}
	}
	return imported, true
}

func rNamespaceImport(src []byte, language *ts.Language, node *ts.Node) (Import, bool) {
	module := node.ChildByFieldName("lhs", language)
	member := node.ChildByFieldName("rhs", language)
	if module == nil || member == nil ||
		module.Type(language) != "identifier" || member.Type(language) != "identifier" {
		return Import{}, false
	}
	return Import{
		Module: module.Text(src),
		Kind:   ImportNamed,
		Names:  []Name{{Name: member.Text(src)}},
		Line:   sourceLine(node),
	}, true
}

func juliaImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "using_statement" && node.Type(language) != importStatement {
			return
		}
		for i := range node.NamedChildCount() {
			imports = append(imports, juliaImportNode(src, language, node.NamedChild(i), sourceLine(node))...)
		}
	})
	return imports
}

func juliaImportNode(src []byte, language *ts.Language, node *ts.Node, line int) []Import {
	switch node.Type(language) {
	case "identifier":
		module := node.Text(src)
		return []Import{{Module: module, Kind: ImportModule, Names: []Name{{Alias: module}}, Line: line}}
	case "import_alias":
		if node.NamedChildCount() < minimumMemberChildren {
			return nil
		}
		return []Import{{
			Module: node.NamedChild(0).Text(src),
			Kind:   ImportModule,
			Names:  []Name{{Alias: node.NamedChild(1).Text(src)}},
			Line:   line,
		}}
	case "selected_import":
		return juliaSelectedImport(src, language, node, line)
	default:
		return nil
	}
}

func juliaSelectedImport(src []byte, language *ts.Language, node *ts.Node, line int) []Import {
	if node.NamedChildCount() < minimumMemberChildren {
		return nil
	}
	module := node.NamedChild(0).Text(src)
	imported := Import{Module: module, Kind: ImportNamed, Line: line}
	for i := 1; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		name := Name{}
		if child.Type(language) == "import_alias" && child.NamedChildCount() > 1 {
			name.Name = child.NamedChild(0).Text(src)
			name.Alias = child.NamedChild(1).Text(src)
		} else {
			name.Name = child.Text(src)
		}
		if name.Name != "" {
			imported.Names = append(imported.Names, name)
		}
	}
	if len(imported.Names) == 0 {
		return nil
	}
	return []Import{imported}
}

func ocamlImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "open_module", "include_module":
			module := node.ChildByFieldName("module", language)
			if module != nil {
				imports = append(imports, Import{
					Module: module.Text(src),
					Kind:   ImportWildcard,
					Line:   sourceLine(node),
				})
			}
		case "module_binding":
			if imported, ok := ocamlModuleBinding(src, language, node); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func ocamlModuleBinding(src []byte, language *ts.Language, binding *ts.Node) (Import, bool) {
	body := binding.ChildByFieldName("body", language)
	if body == nil || body.Type(language) != "module_path" {
		return Import{}, false
	}
	alias := directChildText(src, language, binding, "module_name")
	if alias == "" {
		return Import{}, false
	}
	return Import{
		Module: body.Text(src),
		Kind:   ImportModule,
		Names:  []Name{{Alias: alias}},
		Line:   sourceLine(binding),
	}, true
}

func crystalImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "require" {
			return
		}
		literal := firstDescendantType(node, language, "string")
		if literal == nil {
			return
		}
		module := sourceString(literal.Text(src))
		if module != "" {
			imports = append(imports, Import{Module: module, Kind: ImportSideEffect, Line: sourceLine(node)})
		}
	})
	return imports
}

func nimImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case importStatement:
			list := firstDescendantType(node, language, "expression_list")
			if list == nil {
				return
			}
			for i := range list.NamedChildCount() {
				imports = append(imports, nimImportExpression(src, language, list.NamedChild(i), sourceLine(node))...)
			}
		case "import_from_statement":
			if imported, ok := nimFromImport(src, language, node); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func nimImportExpression(src []byte, language *ts.Language, node *ts.Node, line int) []Import {
	value := strings.TrimSpace(node.Text(src))
	if module, alias, ok := strings.Cut(value, " as "); ok {
		return []Import{{
			Module: strings.TrimSpace(module),
			Kind:   ImportModule,
			Names:  []Name{{Alias: strings.TrimSpace(alias)}},
			Line:   line,
		}}
	}
	if group := firstDescendantType(node, language, "array_construction"); group != nil {
		prefix := node.NamedChild(0).Text(src)
		var imports []Import
		for _, name := range directChildTexts(src, language, group, "identifier") {
			imports = append(imports, Import{
				Module: prefix + "/" + name,
				Kind:   ImportModule,
				Names:  []Name{{Alias: name}},
				Line:   line,
			})
		}
		return imports
	}
	if value == "" {
		return nil
	}
	return []Import{{
		Module: value,
		Kind:   ImportModule,
		Names:  []Name{{Alias: moduleBase(value)}},
		Line:   line,
	}}
}

func nimFromImport(src []byte, language *ts.Language, node *ts.Node) (Import, bool) {
	module := node.ChildByFieldName("module", language)
	list := firstDescendantType(node, language, "expression_list")
	if module == nil || list == nil {
		return Import{}, false
	}
	imported := Import{Module: module.Text(src), Kind: ImportNamed, Line: sourceLine(node)}
	for i := range list.NamedChildCount() {
		value := list.NamedChild(i).Text(src)
		name, alias := splitImportAlias(value)
		if name != "" {
			imported.Names = append(imported.Names, Name{Name: name, Alias: alias})
		}
	}
	return imported, len(imported.Names) > 0
}

func zigImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "builtin_function" {
			return
		}
		builtin := firstDescendantType(node, language, "builtin_identifier")
		content := firstDescendantType(node, language, "string_content")
		if builtin == nil || builtin.Text(src) != "@import" || content == nil || content.Text(src) == "" {
			return
		}
		imported := Import{Module: content.Text(src), Kind: ImportSideEffect, Line: sourceLine(node)}
		parent := node.Parent()
		if parent != nil && parent.Type(language) == "variable_declaration" {
			declaration := parent
			alias := directChildText(src, language, declaration, "identifier")
			if alias != "" {
				imported.Kind = ImportModule
				imported.Names = []Name{{Alias: alias}}
			}
		}
		imports = append(imports, imported)
	})
	return imports
}

func dImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "import_declaration" {
			return
		}
		var importedNodes []*ts.Node
		var names []Name
		for i := range node.NamedChildCount() {
			child := node.NamedChild(i)
			switch child.Type(language) {
			case "imported":
				importedNodes = append(importedNodes, child)
			case "import_bind":
				if name, ok := dImportName(src, child); ok {
					names = append(names, name)
				}
			}
		}
		for _, importedNode := range importedNodes {
			if imported, ok := dImportedModule(src, language, node, importedNode, names); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func dImportedModule(
	src []byte,
	language *ts.Language,
	declaration *ts.Node,
	importedNode *ts.Node,
	names []Name,
) (Import, bool) {
	moduleNode := firstDescendantType(importedNode, language, "module_fqn")
	if moduleNode == nil {
		return Import{}, false
	}
	module := moduleNode.Text(src)
	imported := Import{Module: module, Kind: ImportWildcard, Line: sourceLine(declaration)}
	if alias := importedNode.ChildByFieldName("alias", language); alias != nil {
		imported.Kind = ImportModule
		imported.Names = []Name{{Alias: alias.Text(src)}}
	} else if len(names) > 0 {
		imported.Kind = ImportNamed
		imported.Names = names
	} else if strings.HasPrefix(strings.TrimSpace(declaration.Text(src)), "static import") {
		imported.Kind = ImportModule
		imported.Names = []Name{{Alias: module}}
	}
	return imported, true
}

func dImportName(src []byte, node *ts.Node) (Name, bool) {
	if node.NamedChildCount() == 0 {
		return Name{}, false
	}
	if node.NamedChildCount() == 1 {
		return Name{Name: node.NamedChild(0).Text(src)}, true
	}
	return Name{
		Name:  node.NamedChild(node.NamedChildCount() - 1).Text(src),
		Alias: node.NamedChild(0).Text(src),
	}, true
}

func directChildText(src []byte, language *ts.Language, node *ts.Node, nodeType string) string {
	values := directChildTexts(src, language, node, nodeType)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func directChildTexts(src []byte, language *ts.Language, node *ts.Node, nodeType string) []string {
	if node == nil {
		return nil
	}
	var values []string
	for i := range node.NamedChildCount() {
		child := node.NamedChild(i)
		if child.Type(language) == nodeType {
			values = append(values, child.Text(src))
		}
	}
	return values
}

func moduleBase(module string) string {
	module = strings.TrimSpace(module)
	if index := strings.LastIndexAny(module, "/."); index >= 0 {
		return module[index+1:]
	}
	return module
}
