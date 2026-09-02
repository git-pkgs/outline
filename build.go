package outline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/git-pkgs/gitignore"
)

// ResolutionHints carries caller-supplied context that improves module
// resolution beyond what the repository files alone reveal.
type ResolutionHints struct {
	SourceRoots []string
	Ecosystems  []string
	Packages    []PackageHint
}

// PackageHint maps an installed distribution to the import names it
// provides, for ecosystems where the two differ.
type PackageHint struct {
	Ecosystem string
	Name      string
	Imports   []string
}

// ToolVersion is stamped into every emitted graph.
var ToolVersion = "dev"

const sigCap = 240

// fileAnalysis holds one file's raw bytes and extracted facts.
type fileAnalysis struct {
	path    string
	src     []byte
	a       *analysis
	skipped string
}

// Build walks root, analyses every supported source file, resolves
// cross-file references for supported languages, and returns a Graph.
func Build(root string, opts Options) (*Graph, error) {
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
	paths, truncated, err := collect(abs, m, opts.MaxFiles)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	files := analyseAll(abs, paths, opts)

	g := &Graph{
		SchemaVersion: SchemaVersion,
		ToolVersion:   ToolVersion,
		SourceDigest:  sourceDigest(files),
		Complete:      !truncated,
	}
	if truncated {
		g.Warnings = append(g.Warnings, fmt.Sprintf("file limit %d reached; graph is partial", opts.MaxFiles))
	}

	emitFileNodes(g, files)
	r := newResolver(abs, files, opts.Resolution)
	r.emitModules(g)
	r.emitCalls(g)
	g.Warnings = append(g.Warnings, r.warnings...)

	g.sortStable()
	return g, nil
}

func analyseAll(root string, paths []string, opts Options) []fileAnalysis {
	files := make([]fileAnalysis, len(paths))
	work := make(chan int)
	var wg sync.WaitGroup
	for range opts.Concurrency {
		wg.Go(func() {
			for i := range work {
				files[i] = analyseOne(root, paths[i], opts)
			}
		})
	}
	for i := range paths {
		work <- i
	}
	close(work)
	wg.Wait()
	return files
}

func analyseOne(root, path string, opts Options) fileAnalysis {
	fa := fileAnalysis{path: path}
	full := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(full)
	if err != nil {
		fa.skipped = "unreadable"
		return fa
	}
	if info.Size() > opts.MaxFileSize {
		fa.skipped = "too-large"
		return fa
	}
	src, err := os.ReadFile(full)
	if err != nil {
		fa.skipped = "unreadable"
		return fa
	}
	if _, ok := detect(path); !ok {
		fa.skipped = "unsupported"
		return fa
	}
	a, ok := analyse(src, path)
	if !ok {
		fa.skipped = "parse-failed"
		return fa
	}
	fa.src = src
	fa.a = a
	return fa
}

func emitFileNodes(g *Graph, files []fileAnalysis) {
	for _, f := range files {
		fid := FileID(f.path)
		g.Nodes = append(g.Nodes, Node{ID: fid, Kind: KindFile, Name: f.path, File: f.path})
		if f.a == nil {
			if f.skipped != "" && f.skipped != "unsupported" {
				g.Warnings = append(g.Warnings, f.path+": "+f.skipped)
				g.Complete = false
			}
			continue
		}
		for i, d := range f.a.Decls {
			sid := SymID(f.path, d.Start)
			g.Nodes = append(g.Nodes, Node{
				ID:        sid,
				Kind:      d.Kind,
				Name:      d.Name,
				Qualified: qualified(f.a.Decls, i),
				File:      f.path,
				Line:      d.Line,
				Start:     int(d.Start),
				End:       int(d.End),
				Exported:  d.Exported,
				Sig:       signature(f.src, d),
			})
			from := fid
			if d.Parent >= 0 {
				from = SymID(f.path, f.a.Decls[d.Parent].Start)
			}
			g.Edges = append(g.Edges, Edge{
				From: from, To: sid, Rel: RelContains, Conf: ConfExtracted,
				File: f.path, Line: d.Line,
			})
		}
	}
}

func qualified(decls []decl, i int) string {
	parts := []string{decls[i].Name}
	for p := decls[i].Parent; p >= 0; p = decls[p].Parent {
		parts = append(parts, decls[p].Name)
	}
	for l, r := 0, len(parts)-1; l < r; l, r = l+1, r-1 {
		parts[l], parts[r] = parts[r], parts[l]
	}
	return strings.Join(parts, ".")
}

func signature(src []byte, d decl) string {
	end := min(d.SigEnd, uint32(len(src)))
	s := sanitise(string(src[d.Start:end]))
	if len(s) > sigCap {
		s = s[:sigCap]
	}
	return s
}

// sanitise collapses whitespace runs to a single space and drops control
// bytes so signatures and text output stay one-line safe.
func sanitise(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			space = true
			continue
		}
		if c < ' ' {
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteByte(c)
	}
	return b.String()
}

func sourceDigest(files []fileAnalysis) string {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.path))
		h.Write([]byte{0})
		fh := sha256.Sum256(f.src)
		h.Write(fh[:])
	}
	return hex.EncodeToString(h.Sum(nil))
}
