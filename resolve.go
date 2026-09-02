package outline

import (
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// resolver joins per-file analyses into cross-file graph edges.
type resolver struct {
	root  string
	hints ResolutionHints
	files map[string]*fileAnalysis

	goModule string
	goPkgs   map[string]map[string]string
	pyRoots  []string

	// modules maps a mod: ID to the repository-relative files that
	// implement it, once resolved.
	modules map[string][]string
	// exports maps a mod: ID to that module's top-level name → sym ID.
	exports map[string]map[string]string

	warnings []string
}

// scope is a file's local name table.
type scope struct {
	// syms maps a bare name to a repository sym ID.
	syms map[string]string
	// mods maps a local module alias to its mod ID.
	mods map[string]string
}

func newResolver(root string, files []fileAnalysis, hints ResolutionHints) *resolver {
	r := &resolver{
		root:    root,
		hints:   hints,
		files:   make(map[string]*fileAnalysis, len(files)),
		modules: make(map[string][]string),
		exports: make(map[string]map[string]string),
	}
	for i := range files {
		r.files[files[i].path] = &files[i]
	}
	r.goModule = readGoModule(root)
	r.goPkgs = indexGoPackages(files)
	r.pyRoots = append([]string{""}, hints.SourceRoots...)
	return r
}

// indexGoPackages groups top-level declarations from every Go file by
// directory so same-package calls resolve without an import edge.
func indexGoPackages(files []fileAnalysis) map[string]map[string]string {
	pkgs := make(map[string]map[string]string)
	for i := range files {
		f := &files[i]
		if f.a == nil || f.a.Lang != "go" {
			continue
		}
		dir := path.Dir(f.path)
		if pkgs[dir] == nil {
			pkgs[dir] = make(map[string]string)
		}
		for j, d := range f.a.Decls {
			if d.Parent == -1 {
				pkgs[dir][d.Name] = SymID(f.path, f.a.Decls[j].Start)
			}
		}
	}
	return pkgs
}

func readGoModule(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// emitModules creates one mod: node per distinct import, emits file→mod
// imports edges, and resolves each module to repository files where the
// language resolver supports it.
func (r *resolver) emitModules(g *Graph) {
	seen := make(map[string]bool)
	for p, f := range r.files {
		if f.a == nil {
			continue
		}
		fid := FileID(p)
		for _, imp := range f.a.Imports {
			mid := ModID(f.a.Lang, imp.Module)
			if !seen[mid] {
				seen[mid] = true
				g.Nodes = append(g.Nodes, Node{
					ID: mid, Kind: KindModule, Name: imp.Module,
				})
				r.modules[mid] = r.moduleFiles(f.a.Lang, imp.Module, p)
			}
			g.Edges = append(g.Edges, Edge{
				From: fid, To: mid, Rel: RelImports, Conf: ConfExtracted,
				File: p, Line: imp.Line,
			})
		}
	}
	for mid, paths := range r.modules {
		for _, p := range paths {
			g.Edges = append(g.Edges, Edge{
				From: mid, To: FileID(p), Rel: RelContains, Conf: ConfInferred,
			})
		}
	}
}

// moduleFiles resolves an import module string to zero or more
// repository-relative file paths already in r.files.
func (r *resolver) moduleFiles(lang, module, from string) []string {
	switch lang {
	case "go":
		return r.goModuleFiles(module)
	case "python":
		return r.pyModuleFiles(module, from)
	}
	return nil
}

func (r *resolver) goModuleFiles(module string) []string {
	if r.goModule == "" {
		return nil
	}
	rel, ok := strings.CutPrefix(module, r.goModule)
	if !ok {
		return nil
	}
	dir := strings.TrimPrefix(rel, "/")
	var files []string
	for p, f := range r.files {
		if f.a == nil || f.a.Lang != "go" {
			continue
		}
		if path.Dir(p) == dir || (dir == "" && path.Dir(p) == ".") {
			files = append(files, p)
		}
	}
	return files
}

func (r *resolver) pyModuleFiles(module, from string) []string {
	if strings.HasPrefix(module, ".") {
		return r.pyRelative(module, from)
	}
	rel := strings.ReplaceAll(module, ".", "/")
	for _, root := range r.pyRoots {
		for _, cand := range []string{
			path.Join(root, rel+".py"),
			path.Join(root, rel, "__init__.py"),
		} {
			if _, ok := r.files[cand]; ok {
				return []string{cand}
			}
		}
	}
	return nil
}

func (r *resolver) pyRelative(module, from string) []string {
	dots := 0
	for dots < len(module) && module[dots] == '.' {
		dots++
	}
	dir := path.Dir(from)
	for i := 1; i < dots; i++ {
		dir = path.Dir(dir)
	}
	rest := strings.ReplaceAll(module[dots:], ".", "/")
	base := dir
	if rest != "" {
		base = path.Join(dir, rest)
	}
	for _, cand := range []string{base + ".py", path.Join(base, "__init__.py")} {
		if _, ok := r.files[cand]; ok {
			return []string{cand}
		}
	}
	return nil
}

// moduleExports returns the top-level exported name → sym ID map for a
// resolved module, computing and caching it on first request.
func (r *resolver) moduleExports(mid string) map[string]string {
	if m, ok := r.exports[mid]; ok {
		return m
	}
	m := make(map[string]string)
	for _, p := range r.modules[mid] {
		f := r.files[p]
		if f == nil || f.a == nil {
			continue
		}
		for i, d := range f.a.Decls {
			if d.Parent != -1 || !d.Exported {
				continue
			}
			m[d.Name] = SymID(p, f.a.Decls[i].Start)
		}
	}
	r.exports[mid] = m
	return m
}

// fileScope builds the local name table for one analysed file.
func (r *resolver) fileScope(f *fileAnalysis) scope {
	sc := scope{syms: make(map[string]string), mods: make(map[string]string)}
	if f.a.Lang == "go" {
		maps.Copy(sc.syms, r.goPkgs[path.Dir(f.path)])
	}
	for i, d := range f.a.Decls {
		if d.Parent == -1 {
			sc.syms[d.Name] = SymID(f.path, f.a.Decls[i].Start)
		}
	}
	for _, imp := range f.a.Imports {
		mid := ModID(f.a.Lang, imp.Module)
		switch imp.Kind {
		case ImportModule, ImportNamespace, ImportDefault:
			alias := ""
			if len(imp.Names) > 0 {
				alias = imp.Names[0].Alias
			}
			if alias == "" {
				alias = defaultAlias(f.a.Lang, imp.Module)
			}
			if alias != "" {
				sc.mods[alias] = mid
			}
		case ImportNamed:
			exp := r.moduleExports(mid)
			for _, n := range imp.Names {
				local := n.Alias
				if local == "" {
					local = n.Name
				}
				if sid, ok := exp[n.Name]; ok {
					sc.syms[local] = sid
				} else {
					sc.syms[local] = ExtID(f.a.Lang, imp.Module, n.Name)
				}
			}
		case ImportWildcard:
			maps.Copy(sc.syms, r.moduleExports(mid))
		}
	}
	return sc
}

func defaultAlias(lang, module string) string {
	switch lang {
	case "go":
		if i := strings.LastIndexByte(module, '/'); i >= 0 {
			return module[i+1:]
		}
		return module
	case "python":
		if i := strings.IndexByte(module, '.'); i > 0 {
			return module[:i]
		}
		return module
	}
	return module
}

// emitCalls resolves each call site to a target node and emits a calls
// edge. Unresolved targets get an ext: node.
func (r *resolver) emitCalls(g *Graph) {
	ext := make(map[string]bool)
	for p, f := range r.files {
		if f.a == nil {
			continue
		}
		sc := r.fileScope(f)
		for _, c := range f.a.Calls {
			from := FileID(p)
			if c.In >= 0 {
				from = SymID(p, f.a.Decls[c.In].Start)
			}
			to, conf := r.resolveCall(f, sc, c)
			if strings.HasPrefix(to, "ext:") && !ext[to] {
				ext[to] = true
				g.Nodes = append(g.Nodes, Node{
					ID: to, Kind: KindExternal, Name: c.Name,
					Qualified: extQualified(c),
				})
			}
			g.Edges = append(g.Edges, Edge{
				From: from, To: to, Rel: RelCalls, Conf: conf,
				File: p, Line: c.Line,
			})
		}
	}
}

func (r *resolver) resolveCall(f *fileAnalysis, sc scope, c Call) (string, string) {
	if c.Receiver == "" {
		if sid, ok := sc.syms[c.Name]; ok {
			if strings.HasPrefix(sid, "sym:"+idEscape(f.path)+":") {
				return sid, ConfExtracted
			}
			return sid, ConfInferred
		}
		return ExtID(f.a.Lang, "", c.Name), ConfInferred
	}
	if mid, ok := sc.mods[c.Receiver]; ok {
		if sid, ok := r.moduleExports(mid)[c.Name]; ok {
			return sid, ConfInferred
		}
		return ExtID(f.a.Lang, modName(mid), c.Name), ConfInferred
	}
	return ExtID(f.a.Lang, c.Receiver, c.Name), ConfInferred
}

func extQualified(c Call) string {
	if c.Receiver == "" {
		return c.Name
	}
	return c.Receiver + "." + c.Name
}

func modName(mid string) string {
	if i := strings.LastIndexByte(mid, ':'); i >= 0 {
		return idUnescape(mid[i+1:])
	}
	return mid
}
