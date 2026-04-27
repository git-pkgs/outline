package outline

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}

func (e *errWriter) write(s string) {
	if e.err != nil {
		return
	}
	_, e.err = io.WriteString(e.w, s)
}

// Markdown writes the result as a single markdown document: directory tree
// followed by one fenced code block per file.
func (r *Result) Markdown(w io.Writer) error {
	ew := &errWriter{w: w}
	fence := r.fence()

	if r.Tree != "" {
		ew.printf("## Structure\n\n```\n%s```\n\n", r.Tree)
	}
	if r.Truncated {
		ew.write("_File listing truncated._\n\n")
	}
	ew.write("## Files\n\n")

	for _, f := range r.Files {
		ew.printf("### %s\n\n", f.Path)
		if f.Skipped != "" {
			ew.printf("_(skipped: %s, %d bytes)_\n\n", f.Skipped, f.Size)
			continue
		}
		lang := f.Language
		if lang == "" {
			lang = guessFence(f.Path)
		}
		ew.printf("%s%s\n", fence, lang)
		ew.write(strings.TrimRight(f.Content, "\n"))
		ew.printf("\n%s\n\n", fence)
	}
	return ew.err
}

// XML writes the result in repomix-compatible <file path="...">...</file> form.
func (r *Result) XML(w io.Writer) error {
	ew := &errWriter{w: w}
	if r.Tree != "" {
		ew.printf("<directory_structure>\n%s</directory_structure>\n\n", r.Tree)
	}
	ew.write("<files>\n")
	for _, f := range r.Files {
		if f.Skipped != "" {
			ew.printf("<file path=%q skipped=%q size=\"%d\"/>\n", f.Path, f.Skipped, f.Size)
			continue
		}
		ew.printf("<file path=%q>\n", f.Path)
		ew.write(strings.TrimRight(f.Content, "\n"))
		ew.write("\n</file>\n")
	}
	ew.write("</files>\n")
	return ew.err
}

// fence returns a backtick run longer than any run appearing in file contents,
// so fenced blocks never terminate early.
func (r *Result) fence() string {
	longest := 2
	for _, f := range r.Files {
		s := f.Content
		for {
			i := strings.IndexByte(s, '`')
			if i < 0 {
				break
			}
			j := i
			for j < len(s) && s[j] == '`' {
				j++
			}
			if j-i > longest {
				longest = j - i
			}
			s = s[j:]
		}
	}
	return strings.Repeat("`", longest+1)
}

var fenceByExt = map[string]string{
	".md":   "markdown",
	".json": "json",
	".yaml": "yaml",
	".yml":  "yaml",
	".toml": "toml",
	".sh":   "bash",
	".sql":  "sql",
	".html": "html",
	".css":  "css",
	".xml":  "xml",
	".c":    "c",
	".h":    "c",
	".cpp":  "cpp",
}

func guessFence(path string) string {
	return fenceByExt[filepath.Ext(path)]
}
