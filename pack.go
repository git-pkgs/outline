package outline

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"unicode/utf8"

	"github.com/git-pkgs/gitignore"
)

const (
	defaultMaxFileSize = 1 << 20
	defaultMaxFiles    = 10000
	binarySniffLen     = 8192
)

// Options configures Pack.
type Options struct {
	// MaxFileSize is the per-file byte limit. Files larger than this are
	// listed in the tree but their content is omitted. Zero means 1MB.
	MaxFileSize int64
	// MaxFiles caps the number of files read. Zero means 10000.
	MaxFiles int
	// Compress applies tree-sitter outlining to supported source files.
	// When false, full file contents are kept.
	Compress bool
	// Ignore adds gitignore-syntax patterns on top of .gitignore and the
	// built-in defaults.
	Ignore []string
	// Concurrency is the number of files processed in parallel.
	// Zero means runtime.NumCPU().
	Concurrency int
}

// File is one packed file.
type File struct {
	Path     string
	Content  string
	Language string
	Outlined bool
	Size     int64
	Skipped  string // reason content was omitted, e.g. "binary", "too-large"
}

// Result is the output of Pack.
type Result struct {
	Root      string
	Files     []File
	Tree      string
	Truncated bool // MaxFiles was hit
}

// Pack walks root, reads text files that survive gitignore and default
// filtering, optionally outlines them, and returns the collected result.
func Pack(root string, opts Options) (*Result, error) {
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

	files := process(abs, paths, opts)

	return &Result{
		Root:      abs,
		Files:     files,
		Tree:      Tree(paths),
		Truncated: truncated,
	}, nil
}

type collector struct {
	root      string
	m         *gitignore.Matcher
	maxFiles  int
	paths     []string
	truncated bool
}

func collect(root string, m *gitignore.Matcher, maxFiles int) ([]string, bool, error) {
	c := &collector{root: root, m: m, maxFiles: maxFiles}
	if err := c.walk(""); err != nil {
		return nil, false, err
	}
	return c.paths, c.truncated, nil
}

func (c *collector) walk(rel string) error {
	dir := c.root
	if rel != "" {
		dir = filepath.Join(c.root, rel)
		c.m.AddFromFile(filepath.Join(dir, ".gitignore"), filepath.ToSlash(rel))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if c.truncated {
			return nil
		}
		if err := c.visit(rel, e); err != nil {
			return err
		}
	}
	return nil
}

func (c *collector) visit(rel string, e fs.DirEntry) error {
	name := e.Name()
	if name == ".git" {
		return nil
	}
	entryRel := name
	if rel != "" {
		entryRel = filepath.Join(rel, name)
	}
	slash := filepath.ToSlash(entryRel)

	if e.IsDir() {
		if c.m.MatchPath(slash, true) {
			return nil
		}
		return c.walk(entryRel)
	}
	if e.Type()&fs.ModeSymlink != 0 {
		return nil
	}
	if c.m.MatchPath(slash, false) {
		return nil
	}
	if len(c.paths) >= c.maxFiles {
		c.truncated = true
		return nil
	}
	c.paths = append(c.paths, slash)
	return nil
}

func process(root string, paths []string, opts Options) []File {
	files := make([]File, len(paths))
	work := make(chan int)

	var wg sync.WaitGroup
	for range opts.Concurrency {
		wg.Go(func() {
			for i := range work {
				files[i] = readFile(root, paths[i], opts)
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

func readFile(root, path string, opts Options) File {
	f := File{Path: path}
	full := filepath.Join(root, filepath.FromSlash(path))

	info, err := os.Lstat(full)
	if err != nil {
		f.Skipped = "unreadable"
		return f
	}
	f.Size = info.Size()
	if f.Size > opts.MaxFileSize {
		f.Skipped = "too-large"
		return f
	}

	data, err := os.ReadFile(full)
	if err != nil {
		f.Skipped = "unreadable"
		return f
	}
	if isBinary(data) {
		f.Skipped = "binary"
		return f
	}

	if opts.Compress {
		if l, ok := detect(path); ok {
			if out, ok := Outline(data, path); ok {
				f.Content = out
				f.Outlined = true
				f.Language = l.name
				return f
			}
		}
	}
	f.Content = string(data)
	return f
}

func isBinary(data []byte) bool {
	n := min(len(data), binarySniffLen)
	if bytes.IndexByte(data[:n], 0) >= 0 {
		return true
	}
	return !utf8.Valid(data[:n])
}
