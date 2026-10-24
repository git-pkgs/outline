package outline

import (
	"path"
	"slices"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

const maxPythonAliasDepth = 64

type pythonImport struct {
	Import
	In          int
	End         uint32
	BoundModule string
	Conditional bool
}

type pythonTarget struct {
	id, module, from string
	loaded           []string
}

type pythonBinding struct {
	target    pythonTarget
	found     bool
	uncertain bool
}

type pythonLookup struct{ file, name string }

type pythonExports struct {
	names map[string]bool
	known bool
}

func pythonExportFacts(src []byte, l *lang, root *ts.Node) *pythonExports {
	var exports *pythonExports
	walkNamed(root, func(node *ts.Node) {
		if node.Type(l.language) != "assignment" || ancestorNode(node, "function_definition", l.language) != nil || ancestorNode(node, "class_definition", l.language) != nil {
			return
		}
		left := node.ChildByFieldName("left", l.language)
		if left == nil || left.Text(src) != "__all__" {
			return
		}
		if exports != nil {
			exports.known = false
			return
		}
		exports = &pythonExports{names: make(map[string]bool)}
		value := node.ChildByFieldName("right", l.language)
		if value == nil || (value.Type(l.language) != "list" && value.Type(l.language) != "tuple") {
			return
		}
		exports.known = true
		for i := range value.NamedChildCount() {
			child := value.NamedChild(i)
			text := child.Text(src)
			if child.Type(l.language) != "string" || strings.ContainsAny(text, "\\\n\r") || sourceString(text) == "" {
				exports.known = false
				return
			}
			exports.names[sourceString(text)] = true
		}
	})
	return exports
}

func (b *pythonBinding) add(target pythonTarget) {
	if target.id == "" && target.module == "" {
		b.uncertain = true
	}
	if b.found && (b.target.id != target.id || b.target.module != target.module || b.target.from != target.from) {
		b.uncertain = true
	}
	if !b.found {
		b.target = target
	}
	for _, loaded := range target.loaded {
		if !slices.Contains(b.target.loaded, loaded) {
			b.target.loaded = append(b.target.loaded, loaded)
		}
	}
	b.found = true
}

func (b pythonBinding) resolved() pythonTarget {
	if b.uncertain {
		return pythonTarget{}
	}
	return b.target
}

func pythonImportFacts(src []byte, l *lang, root *ts.Node, decls []decl) []pythonImport {
	var facts []pythonImport
	walkNamed(root, func(node *ts.Node) {
		var imports []Import
		switch node.Type(l.language) {
		case importStatement:
			imports = pythonModuleImports(src, l.language, node)
		case importFromStatement, futureImportStatement:
			if imp, ok := pythonFromImport(src, l.language, node); ok {
				imports = []Import{imp}
			}
		}
		for i, imp := range imports {
			bound := imp.Module
			if imp.Kind == ImportModule && node.NamedChild(i).Type(l.language) != aliasedImport {
				bound, _, _ = strings.Cut(imp.Module, ".")
			}
			facts = append(facts, pythonImport{
				Import: imp, In: enclosing(decls, node.StartByte()), End: node.EndByte(), BoundModule: bound,
				Conditional: pythonConditional(node, l.language),
			})
		}
	})
	return facts
}

func pythonConditional(node *ts.Node, language *ts.Language) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Type(language) {
		case "function_definition", "class_definition", "module":
			return false
		case "block", "decorated_definition":
		default:
			return true
		}
	}
	return false
}

func (r *resolver) pyLookup(module, from, name string, seen map[pythonLookup]bool) pythonBinding {
	files := r.pyModuleFiles(module, from)
	if len(files) != 1 || r.files[files[0]].a == nil {
		return pythonBinding{}
	}
	return r.pyFileLookup(r.files[files[0]], name, seen)
}

func (r *resolver) pyFileLookup(f *fileAnalysis, name string, seen map[pythonLookup]bool) pythonBinding {
	key := pythonLookup{f.path, name}
	if seen[key] || len(seen) >= maxPythonAliasDepth {
		return pythonBinding{found: true, uncertain: true}
	}
	seen[key] = true
	defer delete(seen, key)
	var result pythonBinding
	for _, d := range f.a.Decls {
		if d.Parent == -1 && d.Name == name {
			result.add(pythonTarget{id: d.symID(f.path)})
		}
	}
	for _, imp := range f.a.PyImports {
		if imp.In != -1 {
			continue
		}
		if binding := r.pyImportName(f, imp, name, seen); binding.found {
			result.add(binding.resolved())
		}
	}
	return result
}

func (r *resolver) pyImportName(f *fileAnalysis, imp pythonImport, name string, seen map[pythonLookup]bool) pythonBinding {
	if imp.Kind == ImportWildcard {
		return r.pyWildcardLookup(imp.Module, f.path, name, seen)
	}
	var result pythonBinding
	for _, n := range imp.Names {
		local := n.Alias
		if local == "" {
			local = n.Name
		}
		if local != name {
			continue
		}
		if imp.Kind == ImportModule {
			target := pythonTarget{module: imp.BoundModule, from: f.path}
			if imp.Module != imp.BoundModule {
				target.loaded = []string{imp.Module}
			}
			result.add(target)
		} else {
			result.add(r.pyNamedImport(f, imp, n.Name, seen))
		}
	}
	return result
}

func (r *resolver) pyNamedImport(f *fileAnalysis, imp pythonImport, name string, seen map[pythonLookup]bool) pythonTarget {
	module := pythonChildModule(imp.Module, name)
	files := r.pyModuleFiles(imp.Module, f.path)
	packageModule := len(files) == 0 || path.Base(files[0]) == "__init__.py"
	if packageModule && len(files) == 1 && files[0] == f.path && len(r.pyModuleFiles(module, f.path)) == 1 {
		return pythonTarget{module: module, from: f.path}
	}
	if binding := r.pyLookup(imp.Module, f.path, name, seen); binding.found {
		return binding.resolved()
	}
	if files := r.pyModuleFiles(module, f.path); packageModule && len(files) == 1 {
		return pythonTarget{module: module, from: f.path}
	}
	if len(r.pyModuleFiles(imp.Module, f.path)) == 0 {
		return pythonTarget{id: ExtID("python", modName(r.moduleID(f, imp.Import)), name)}
	}
	return pythonTarget{}
}

func (r *resolver) pyWildcardLookup(module, from, name string, seen map[pythonLookup]bool) pythonBinding {
	files := r.pyModuleFiles(module, from)
	if len(files) != 1 || r.files[files[0]].a == nil {
		return pythonBinding{}
	}
	f := r.files[files[0]]
	if exports := f.a.PyExports; exports != nil {
		if !exports.known {
			return pythonBinding{found: true, uncertain: true}
		}
		if !exports.names[name] {
			return pythonBinding{}
		}
	} else if strings.HasPrefix(name, "_") {
		return pythonBinding{}
	}
	return r.pyFileLookup(f, name, seen)
}

func (r *resolver) emitPythonSubmodules(g *Graph, seen map[string]bool) {
	for _, p := range r.paths {
		f := r.files[p]
		if f.a == nil || f.a.Lang != "python" {
			continue
		}
		for _, imp := range f.a.PyImports {
			if imp.Kind != ImportNamed {
				continue
			}
			for _, n := range imp.Names {
				target := r.pyNamedImport(f, imp, n.Name, make(map[pythonLookup]bool))
				if target.module == "" {
					continue
				}
				mid := r.moduleID(r.files[target.from], Import{Module: target.module})
				if !seen[mid] {
					seen[mid] = true
					g.Nodes = append(g.Nodes, Node{ID: mid, Kind: KindModule, Name: target.module})
				}
				for _, file := range r.pyModuleFiles(target.module, target.from) {
					if !slices.Contains(r.modules[mid], file) {
						r.modules[mid] = append(r.modules[mid], file)
					}
				}
				g.Edges = append(g.Edges, Edge{
					From: FileID(p), To: mid, Rel: RelImports, Conf: ConfInferred, File: p, Line: imp.Line,
				})
			}
		}
	}
}

func pythonChildModule(module, name string) string {
	if strings.HasSuffix(module, ".") {
		return module + name
	}
	return module + "." + name
}

func (r *resolver) pyMember(target pythonTarget, name string, seen map[pythonLookup]bool) pythonTarget {
	if target.module == "" {
		return pythonTarget{}
	}
	if binding := r.pyLookup(target.module, target.from, name, seen); binding.found {
		return binding.resolved()
	}
	child := pythonChildModule(target.module, name)
	for _, loaded := range target.loaded {
		if loaded == child || strings.HasPrefix(loaded, child+".") {
			return pythonTarget{module: child, from: target.from, loaded: target.loaded}
		}
	}
	if len(r.pyModuleFiles(target.module, target.from)) == 0 {
		return pythonTarget{id: ExtID("python", target.module, name)}
	}
	return pythonTarget{}
}

type pythonScopeBinding struct {
	pythonBinding
	at       uint32
	deferred bool
}

func (b *pythonScopeBinding) assign(target pythonTarget, at, pos uint32, conditional bool) {
	// Enclosing scopes can change before a nested body runs.
	if b.deferred {
		if conditional {
			target = pythonTarget{}
		}
		b.add(target)
		return
	}
	b.found = true
	if at > pos || at < b.at {
		return
	}
	if target.module != "" && target.module == b.target.module && target.from == b.target.from && !b.uncertain {
		for _, loaded := range b.target.loaded {
			if !slices.Contains(target.loaded, loaded) {
				target.loaded = append(target.loaded, loaded)
			}
		}
	}
	b.at = at
	b.target = target
	b.uncertain = conditional
}

func (r *resolver) pyLocalBinding(f *fileAnalysis, sc scope, c Call, at int, name string, seen map[pythonLookup]bool) pythonBinding {
	result := pythonScopeBinding{deferred: at != c.In}
	result.found = slices.Contains(f.a.Decls[at].Params, name)
	for _, b := range sc.bindings[at] {
		if b.Name == name && b.Start <= c.Start && c.Start < b.End {
			result.assign(pythonTarget{}, b.At, c.Start, false)
		}
	}
	for _, ci := range sc.children[at] {
		d := f.a.Decls[ci]
		if d.Name != name {
			continue
		}
		result.assign(pythonTarget{id: d.symID(f.path)}, d.End, c.Start, d.Conditional)
	}
	for _, imp := range f.a.PyImports {
		if imp.In != at {
			continue
		}
		if binding := r.pyImportName(f, imp, name, seen); binding.found {
			result.assign(binding.resolved(), imp.End, c.Start, imp.Conditional)
		}
	}
	return result.pythonBinding
}

func (r *resolver) pyCallBinding(f *fileAnalysis, sc scope, c Call, name string, seen map[pythonLookup]bool) pythonBinding {
	for at := c.In; at >= 0; at = f.a.Decls[at].Parent {
		if at != c.In && f.a.Decls[at].Kind == KindClass {
			continue
		}
		if result := r.pyLocalBinding(f, sc, c, at, name, seen); result.found {
			return result
		}
	}
	return r.pyFileLookup(f, name, seen)
}

func (r *resolver) resolvePythonCall(f *fileAnalysis, sc scope, c Call) (string, string) {
	name := c.Name
	if c.Receiver != "" {
		name, _, _ = strings.Cut(c.Receiver, ".")
	}
	if c.ReceiverKind == ReceiverExpression {
		return ExtID("python", c.Receiver, c.Name), ConfInferred
	}
	seen := make(map[pythonLookup]bool)
	target := r.pyCallBinding(f, sc, c, name, seen).resolved()
	if c.Receiver != "" {
		for _, member := range strings.Split(c.Receiver, ".")[1:] {
			target = r.pyMember(target, member, seen)
		}
		target = r.pyMember(target, c.Name, seen)
	}
	if target.id != "" {
		conf := ConfInferred
		if c.Receiver == "" && strings.HasPrefix(target.id, "sym:"+idEscape(f.path)+":") {
			conf = ConfExtracted
		}
		return target.id, conf
	}
	return ExtID("python", c.Receiver, c.Name), ConfInferred
}
