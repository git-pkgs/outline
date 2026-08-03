package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/git-pkgs/outline"
)

var (
	graphDir = flag.String("graph", "", "build a symbol graph for the given directory and print stats")
	callers  = flag.String("callers", "", "with -graph, list callers of the named symbol")
)

func main() {
	if *graphDir != "" {
		runGraph(*graphDir)
		return
	}
	for _, path := range flag.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		out, ok := outline.Outline(data, path)
		if !ok {
			fmt.Fprintf(os.Stderr, "unsupported: %s\n", path)
			continue
		}
		fmt.Printf("==== %s ====\n%s\n\n", path, out)
	}
}

func runGraph(dir string) {
	g, err := outline.BuildGraph(dir, outline.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	st := g.Stats()
	fmt.Printf("root:     %s\n", g.Root)
	fmt.Printf("files:    %d\n", st.Files)
	fmt.Printf("symbols:  %d\n", st.Symbols)

	kinds := []outline.RefKind{outline.RefCall, outline.RefType, outline.RefInherit, outline.RefImport}
	fmt.Printf("\n%-10s %8s %8s %8s %6s\n", "ref kind", "total", "in-repo", "unique", "hit%")
	for _, k := range kinds {
		total := st.Refs[k]
		if total == 0 {
			continue
		}
		res := st.Resolved[k]
		uniq := st.Unambiguous[k]
		fmt.Printf("%-10s %8d %8d %8d %5.1f%%\n", k, total, res, uniq, 100*float64(uniq)/float64(total))
	}

	if *callers != "" {
		fmt.Printf("\ncallers of %s:\n", *callers)
		cs := g.Callers(*callers)
		sort.Slice(cs, func(i, j int) bool { return cs[i].File < cs[j].File })
		for _, c := range cs {
			fmt.Printf("  %s:%d %s\n", c.File, c.StartLine, c.Name)
		}
		if len(cs) == 0 {
			fmt.Println("  (none)")
		}
	}
}
