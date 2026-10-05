package outline

import (
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

const rubyModuleFunction = "module_function"

type rubyMethodKey struct {
	owner, name string
	singleton   bool
}

type rubyScope struct {
	methods   map[rubyMethodKey]string
	constants map[string]string
	copies    map[string]bool
}

func rubyLoadFacts(src []byte, l *lang, root *ts.Node) []Import {
	var loads []Import
	walkNamed(root, func(node *ts.Node) {
		imp, ok := rubyImport(src, l.language, node)
		if !ok || !rubyImmediateLoad(node, l.language) {
			return
		}
		loads = append(loads, imp)
	})
	return loads
}

func rubyImmediateLoad(node *ts.Node, language *ts.Language) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Type(language) {
		case "program", "body_statement", "module", "class", "singleton_class":
		default:
			return false
		}
	}
	return true
}

type rubyMode struct {
	In      int
	Start   uint32
	Enabled bool
}

func rubyModeFacts(a *analysis, src []byte, l *lang, root *ts.Node) []rubyMode {
	var modes []rubyMode
	add := func(name string, start uint32, in int) {
		if name == rubyModuleFunction || name == "public" || name == "private" || name == "protected" {
			modes = append(modes, rubyMode{in, start, name == rubyModuleFunction})
		}
	}
	for _, c := range a.Calls {
		if len(c.Arguments) == 0 && (c.ReceiverKind == ReceiverBare || c.ReceiverKind == ReceiverSelf) {
			add(c.Name, c.Start, c.In)
		}
	}
	walkNamed(root, func(node *ts.Node) {
		if node.Type(l.language) == "identifier" && node.Parent() != nil && node.Parent().Type(l.language) == "body_statement" {
			add(node.Text(src), node.StartByte(), enclosing(a.Decls, node.StartByte()))
		}
	})
	sort.Slice(modes, func(i, j int) bool { return modes[i].Start < modes[j].Start })
	return modes
}

func rubyModuleFunctions(a *analysis, src []byte, l *lang, root *ts.Node) {
	modes := rubyModeFacts(a, src, l, root)
	for i := range a.Decls {
		d := &a.Decls[i]
		if d.Kind != KindFunc || d.Singleton || d.Parent < 0 || a.Decls[d.Parent].Kind != KindType {
			continue
		}
		if rubyCustomModuleFunction(a, d.Parent) {
			continue
		}
		mode := false
		for _, event := range modes {
			if event.In == d.Parent && event.Start < d.Start {
				mode = event.Enabled
			}
		}
		d.ModuleFunction = mode || rubyNamedModuleCopy(a, *d)
	}
}

func rubyCustomModuleFunction(a *analysis, owner int) bool {
	name := rubyQualified(a.Decls, owner)
	for _, d := range a.Decls {
		if d.Singleton && d.Name == rubyModuleFunction && (d.Parent == owner || d.Owner == name) {
			return true
		}
	}
	return false
}

func rubyNamedModuleCopy(a *analysis, d decl) bool {
	for _, c := range a.Calls {
		if c.Name != rubyModuleFunction || c.In != d.Parent || d.End > c.Start {
			continue
		}
		if c.ReceiverKind != ReceiverBare && c.ReceiverKind != ReceiverSelf {
			continue
		}
		for _, arg := range c.Arguments {
			if rubyLiteralMethod(arg) == d.Name {
				return true
			}
		}
	}
	return false
}

func rubyLiteralMethod(arg Argument) string {
	switch arg.Kind {
	case "simple_symbol":
		return strings.TrimPrefix(arg.Text, ":")
	case "string", "delimited_symbol":
		text := strings.TrimPrefix(arg.Text, ":")
		if !strings.ContainsAny(text, "\\#\n\r") {
			return sourceString(text)
		}
	}
	return ""
}

func (r *resolver) rubyFiles(f *fileAnalysis) []*fileAnalysis {
	var files []*fileAnalysis
	queue := []string{f.path}
	seen := make(map[string]bool)
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if seen[p] {
			continue
		}
		seen[p] = true
		loaded := r.files[p]
		if loaded == nil || loaded.a == nil || loaded.a.Lang != "ruby" {
			continue
		}
		files = append(files, loaded)
		for _, imp := range loaded.a.RubyLoads {
			queue = append(queue, r.rubyModuleFiles(imp, p)...)
		}
	}
	return files
}

func (r *resolver) rubyFileScope(f *fileAnalysis) rubyScope {
	if sc, ok := r.rubyScopes[f.path]; ok {
		return sc
	}
	files := r.rubyFiles(f)
	key := rubyScopeKey(files)
	if sc, ok := r.rubyScopes[key]; ok {
		r.rubyScopes[f.path] = sc
		return sc
	}
	sc := rubyScope{
		methods: make(map[rubyMethodKey]string), constants: make(map[string]string), copies: make(map[string]bool),
	}
	for _, loaded := range files {
		for i, d := range loaded.a.Decls {
			if d.Kind != KindClass && d.Kind != KindType && d.Kind != KindConst {
				continue
			}
			name := strings.TrimPrefix(rubyQualified(loaded.a.Decls, i), "::")
			previous, exists := sc.constants[name]
			if exists && previous != d.Kind {
				sc.constants[name] = ""
			} else if !exists {
				sc.constants[name] = d.Kind
			}
		}
	}
	for _, loaded := range files {
		rubyIndexMethods(sc, loaded)
	}
	r.rubyScopes[f.path] = sc
	r.rubyScopes[key] = sc
	return sc
}

func rubyScopeKey(files []*fileAnalysis) string {
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	// NUL cannot occur in a file path.
	return "\x00" + strings.Join(paths, "\x00")
}

func rubyIndexMethods(sc rubyScope, f *fileAnalysis) {
	for i, d := range f.a.Decls {
		if d.Kind != KindFunc {
			continue
		}
		owner, _ := rubyCallContext(f.a.Decls, i)
		name := ""
		if owner >= 0 {
			name = strings.TrimPrefix(rubyQualified(f.a.Decls, owner), "::")
		}
		if d.Singleton && d.Owner != "" && d.Owner != "self" {
			name, _ = rubyReceiverOwner(sc, f.a.Decls, d.Parent, d.Owner)
		}
		id := d.symID(f.path)
		keys := []rubyMethodKey{{name, d.Name, d.Singleton}}
		if d.ModuleFunction {
			keys = append(keys, rubyMethodKey{name, d.Name, true})
			sc.copies[id] = true
		}
		for _, key := range keys {
			if previous, exists := sc.methods[key]; exists && previous != id {
				sc.methods[key] = ""
			} else if !exists {
				sc.methods[key] = id
			}
		}
	}
}

func rubyReceiverOwner(sc rubyScope, decls []decl, in int, receiver string) (string, bool) {
	name := strings.TrimPrefix(receiver, "::")
	candidates := []string{name}
	if !strings.HasPrefix(receiver, "::") {
		candidates = nil
		for at := in; at >= 0; at = decls[at].Parent {
			if decls[at].Kind == KindClass || decls[at].Kind == KindType {
				candidates = append(candidates, rubyQualified(decls, at)+"::"+name)
			}
		}
		candidates = append(candidates, name)
	}
	for _, candidate := range candidates {
		if _, exists := sc.constants[candidate]; exists {
			return candidate, true
		}
		head, _, _ := strings.Cut(name, "::")
		head = strings.TrimSuffix(candidate, name) + head
		if _, exists := sc.constants[head]; exists {
			return candidate, true
		}
		for key := range sc.methods {
			if key.owner == candidate {
				return candidate, true
			}
		}
	}
	return name, false
}

func rubyBlockedOwner(sc rubyScope, owner string) bool {
	parts := strings.Split(owner, "::")
	for i := range parts {
		if kind, exists := sc.constants[strings.Join(parts[:i+1], "::")]; exists && (kind == "" || kind == KindConst) {
			return true
		}
	}
	return false
}

func (r *resolver) resolveRubyCall(f *fileAnalysis, c Call) (string, string) {
	sc := r.rubyFileScope(f)
	owner := ""
	singleton := false
	local := false
	switch c.ReceiverKind {
	case ReceiverBare, ReceiverSelf:
		at, classMethod := rubyCallContext(f.a.Decls, c.In)
		singleton = classMethod
		if at >= 0 {
			owner = rubyQualified(f.a.Decls, at)
		}
		if c.In >= 0 {
			d := f.a.Decls[c.In]
			if d.Singleton && d.Owner != "" && d.Owner != "self" {
				owner, _ = rubyReceiverOwner(sc, f.a.Decls, d.Parent, d.Owner)
			}
		}
	case ReceiverConstant:
		owner, local = rubyReceiverOwner(sc, f.a.Decls, c.In, c.Receiver)
		singleton = true
	default:
		return ExtID("ruby", c.Receiver, c.Name), ConfInferred
	}
	if !rubyBlockedOwner(sc, owner) {
		if id := sc.methods[rubyMethodKey{owner, c.Name, singleton}]; id != "" {
			conf := ConfInferred
			if strings.HasPrefix(id, "sym:"+idEscape(f.path)+":") && !sc.copies[id] {
				conf = ConfExtracted
			}
			return id, conf
		}
	}
	if local {
		return ExtID("ruby", "local:"+owner, c.Name), ConfInferred
	}
	return ExtID("ruby", c.Receiver, c.Name), ConfInferred
}
