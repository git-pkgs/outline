package outline

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/git-pkgs/gitignore"
)

// Graph holds the symbols and references extracted from a source tree, with
// Refs[i].To resolved where a single unambiguous definition exists.
type Graph struct {
	Root    string
	Symbols []Symbol
	Refs    []Ref

	byName map[string][]*Symbol
}

// Stats summarises resolution coverage for a graph.
type Stats struct {
	Files       int
	Symbols     int
	Refs        map[RefKind]int
	Resolved    map[RefKind]int
	Unambiguous map[RefKind]int
}

// BuildGraph walks root, extracts symbols and references from every supported
// file, and resolves each reference to a definition when exactly one symbol
// with a matching name exists in the tree.
func BuildGraph(root string, opts Options) (*Graph, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = defaultMaxFileSize
	}
	if opts.MaxFiles == 0 {
		opts.MaxFiles = defaultMaxFiles
	}
	if opts.Concurrency == 0 {
		opts.Concurrency = runtime.NumCPU()
	}

	m := gitignore.New(abs)
	m.AddPatterns(defaultIgnore, "")
	for _, p := range opts.Ignore {
		m.AddPatterns([]byte(p+"\n"), "")
	}

	paths, _, err := collect(abs, m, opts.MaxFiles)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	g := &Graph{Root: abs}
	g.extract(paths, opts)
	g.index()
	g.resolve()
	return g, nil
}

type fileExtract struct {
	syms []Symbol
	refs []Ref
}

func (g *Graph) extract(paths []string, opts Options) {
	results := make([]fileExtract, len(paths))
	work := make(chan int)

	var wg sync.WaitGroup
	for range opts.Concurrency {
		wg.Go(func() {
			for i := range work {
				results[i] = extractFile(g.Root, paths[i], opts.MaxFileSize)
			}
		})
	}
	for i := range paths {
		work <- i
	}
	close(work)
	wg.Wait()

	for _, r := range results {
		g.Symbols = append(g.Symbols, r.syms...)
		g.Refs = append(g.Refs, r.refs...)
	}
}

func extractFile(root, path string, maxSize int64) fileExtract {
	full := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(full)
	if err != nil || info.Size() > maxSize {
		return fileExtract{}
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return fileExtract{}
	}
	syms, refs, ok := Extract(data, path)
	if !ok {
		return fileExtract{}
	}
	return fileExtract{syms: syms, refs: refs}
}

func (g *Graph) index() {
	g.byName = make(map[string][]*Symbol, len(g.Symbols))
	for i := range g.Symbols {
		s := &g.Symbols[i]
		g.byName[s.Name] = append(g.byName[s.Name], s)
	}
}

func (g *Graph) resolve() {
	for i := range g.Refs {
		r := &g.Refs[i]
		if r.Kind == RefImport {
			continue
		}
		cands := g.byName[r.Target]
		if len(cands) == 0 {
			continue
		}
		r.To = pick(cands, r)
	}
}

// pick returns the best candidate for a ref: prefer same-file, otherwise
// return the sole candidate, otherwise nil (ambiguous).
func pick(cands []*Symbol, r *Ref) *Symbol {
	if len(cands) == 1 {
		return cands[0]
	}
	var sameFile *Symbol
	for _, c := range cands {
		if c.File == r.File {
			if sameFile != nil {
				return nil
			}
			sameFile = c
		}
	}
	return sameFile
}

// Lookup returns all symbols with the given name.
func (g *Graph) Lookup(name string) []*Symbol {
	return g.byName[name]
}

// Callers returns symbols that contain a resolved call ref to the named
// symbol.
func (g *Graph) Callers(name string) []*Symbol {
	var out []*Symbol
	for i := range g.Refs {
		r := &g.Refs[i]
		if r.Kind != RefCall || r.To == nil || r.To.Name != name {
			continue
		}
		if s := g.enclosing(r); s != nil {
			out = append(out, s)
		}
	}
	return out
}

// enclosing returns the innermost func/method symbol whose range contains r.
func (g *Graph) enclosing(r *Ref) *Symbol {
	var best *Symbol
	for i := range g.Symbols {
		s := &g.Symbols[i]
		if s.File != r.File {
			continue
		}
		if s.Kind != KindFunc && s.Kind != KindMethod {
			continue
		}
		if s.StartLine <= r.Line && r.Line <= s.EndLine {
			if best == nil || s.StartLine > best.StartLine {
				best = s
			}
		}
	}
	return best
}

// Stats reports symbol and reference counts and how many refs of each kind
// resolved to exactly one definition.
func (g *Graph) Stats() Stats {
	files := make(map[string]struct{})
	for _, s := range g.Symbols {
		files[s.File] = struct{}{}
	}
	st := Stats{
		Files:       len(files),
		Symbols:     len(g.Symbols),
		Refs:        make(map[RefKind]int),
		Resolved:    make(map[RefKind]int),
		Unambiguous: make(map[RefKind]int),
	}
	for i := range g.Refs {
		r := &g.Refs[i]
		st.Refs[r.Kind]++
		if len(g.byName[r.Target]) > 0 {
			st.Resolved[r.Kind]++
		}
		if r.To != nil {
			st.Unambiguous[r.Kind]++
		}
	}
	return st
}
