package outline

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

func goImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "import_spec" {
			return
		}
		var module string
		var alias string
		for i := range node.NamedChildCount() {
			child := node.NamedChild(i)
			switch child.Type(language) {
			case "interpreted_string_literal", "raw_string_literal":
				module = sourceString(child.Text(src))
			case "package_identifier":
				alias = child.Text(src)
			case "blank_identifier":
				alias = "_"
			case "dot":
				alias = "."
			}
		}
		if module == "" {
			return
		}
		imported := Import{Module: module, Kind: ImportModule, Line: sourceLine(node)}
		switch alias {
		case "_":
			imported.Kind = ImportSideEffect
		case ".":
			imported.Kind = ImportWildcard
		case "":
		default:
			imported.Names = []Name{{Alias: alias}}
		}
		imports = append(imports, imported)
	})
	return imports
}

func rubyImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "call" || node.NamedChildCount() < 2 {
			return
		}
		function := node.NamedChild(0)
		if function.Type(language) != "identifier" || function.Text(src) != "require" {
			return
		}
		arguments := node.NamedChild(1)
		moduleNode := firstDescendantType(arguments, language, "string")
		if moduleNode == nil {
			return
		}
		module := sourceString(moduleNode.Text(src))
		if module != "" {
			imports = append(imports, Import{Module: module, Kind: ImportSideEffect, Line: sourceLine(node)})
		}
	})
	return imports
}

func rustImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		switch node.Type(language) {
		case "use_declaration":
			if imported, ok := rustUseImport(node.Text(src), sourceLine(node)); ok {
				imports = append(imports, imported)
			}
		case "extern_crate_declaration":
			if imported, ok := rustExternCrateImport(node.Text(src), sourceLine(node)); ok {
				imports = append(imports, imported)
			}
		}
	})
	return imports
}

func rustUseImport(value string, line int) (Import, bool) {
	rest, ok := afterWord(value, "use")
	if !ok {
		return Import{}, false
	}
	rest = strings.TrimSpace(strings.TrimSuffix(rest, ";"))
	module, tail, hasTail := strings.Cut(rest, "::")
	module = strings.TrimSpace(module)
	if module == "" {
		return Import{}, false
	}
	if !hasTail {
		name, alias := splitImportAlias(module)
		if name == "" {
			return Import{}, false
		}
		imported := Import{Module: name, Kind: ImportModule, Names: []Name{{Alias: name}}, Line: line}
		if alias != "" {
			imported.Names[0].Alias = alias
		}
		return imported, true
	}

	tail = strings.TrimSpace(tail)
	if tail == "*" {
		return Import{Module: module, Kind: ImportWildcard, Line: line}, true
	}
	if strings.HasPrefix(tail, "{") && strings.HasSuffix(tail, "}") {
		items := strings.Split(strings.TrimSpace(tail[1:len(tail)-1]), ",")
		names := make([]Name, 0, len(items))
		for _, item := range items {
			name, alias := splitImportAlias(strings.TrimSpace(item))
			if name != "" {
				names = append(names, Name{Name: name, Alias: alias})
			}
		}
		if len(names) == 0 {
			return Import{}, false
		}
		return Import{Module: module, Kind: ImportNamed, Names: names, Line: line}, true
	}
	name, alias := splitImportAlias(tail)
	if name == "" {
		return Import{}, false
	}
	return Import{
		Module: module,
		Kind:   ImportNamed,
		Names:  []Name{{Name: name, Alias: alias}},
		Line:   line,
	}, true
}

func rustExternCrateImport(value string, line int) (Import, bool) {
	rest, ok := afterWord(value, "crate")
	if !ok {
		return Import{}, false
	}
	name, alias := splitImportAlias(strings.TrimSpace(strings.TrimSuffix(rest, ";")))
	if name == "" {
		return Import{}, false
	}
	if alias == "" {
		alias = name
	}
	return Import{Module: name, Kind: ImportModule, Names: []Name{{Alias: alias}}, Line: line}, true
}

func splitImportAlias(value string) (string, string) {
	name, alias, found := strings.Cut(strings.TrimSpace(value), " as ")
	if !found {
		return name, ""
	}
	return strings.TrimSpace(name), strings.TrimSpace(alias)
}

func afterWord(value, word string) (string, bool) {
	for _, prefix := range []string{word + " ", "pub " + word + " "} {
		if index := strings.Index(value, prefix); index >= 0 {
			return value[index+len(prefix):], true
		}
	}
	return "", false
}

func phpImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "namespace_use_clause" {
			return
		}
		value := strings.TrimSpace(node.Text(src))
		name, alias := splitCaseInsensitiveAlias(value)
		name = strings.TrimPrefix(name, `\`)
		if name == "" {
			return
		}
		if alias == "" {
			parts := strings.Split(name, `\`)
			alias = parts[len(parts)-1]
		}
		imports = append(imports, Import{
			Module: name,
			Kind:   ImportModule,
			Names:  []Name{{Alias: alias}},
			Line:   sourceLine(node),
		})
	})
	return imports
}

func splitCaseInsensitiveAlias(value string) (string, string) {
	lower := strings.ToLower(value)
	index := strings.LastIndex(lower, " as ")
	if index < 0 {
		return value, ""
	}
	return strings.TrimSpace(value[:index]), strings.TrimSpace(value[index+4:])
}

func elixirImports(src []byte, language *ts.Language, root *ts.Node) []Import {
	var imports []Import
	walkNamed(root, func(node *ts.Node) {
		if node.Type(language) != "call" || node.NamedChildCount() < 2 {
			return
		}
		function := node.NamedChild(0)
		if function.Type(language) != "identifier" {
			return
		}
		form := function.Text(src)
		if form != "alias" && form != "import" && form != "require" && form != "use" {
			return
		}
		arguments := node.NamedChild(1)
		aliases := descendantTexts(src, language, arguments, "alias")
		if len(aliases) == 0 {
			return
		}
		module := aliases[0]
		imported := Import{Module: module, Kind: ImportModule, Line: sourceLine(node)}
		if form == "import" {
			imported.Kind = ImportWildcard
		} else {
			alias := module[strings.LastIndex(module, ".")+1:]
			if len(aliases) > 1 {
				alias = aliases[len(aliases)-1]
			}
			imported.Names = []Name{{Alias: alias}}
		}
		imports = append(imports, imported)
	})
	return imports
}
