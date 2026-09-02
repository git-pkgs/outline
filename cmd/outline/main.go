package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/git-pkgs/outline"
)

const (
	exitUsage     = 2
	defaultBudget = 2000
	pathArgs      = 2
)

const usage = `outline graph [-json] [-o file] [dir]
outline def [-g file] [-dir path] <name>
outline callers [-g file] [-dir path] [-inferred] <name>
outline callees [-g file] [-dir path] [-inferred] <name>
outline affected [-g file] [-dir path] [-depth N] [-relations r,...] [-inferred] <name>...
outline path [-g file] [-dir path] [-relations r,...] [-inferred] <from> <to>

Flags must precede positional arguments.
`

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(exitUsage)
	}
	var err error
	switch os.Args[1] {
	case "graph":
		err = cmdGraph(os.Args[2:])
	case "def":
		err = cmdDef(os.Args[2:])
	case "callers":
		err = cmdNeighbours(os.Args[2:], true)
	case "callees":
		err = cmdNeighbours(os.Args[2:], false)
	case "affected":
		err = cmdAffected(os.Args[2:])
	case "path":
		err = cmdPath(os.Args[2:])
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(exitUsage)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "outline:", err)
		os.Exit(1)
	}
}

type common struct {
	graph    string
	dir      string
	depth    int
	inferred bool
	rels     string
	budget   int
}

func (c *common) register(fs *flag.FlagSet) {
	fs.StringVar(&c.graph, "g", "", "load graph from file instead of building")
	fs.StringVar(&c.dir, "dir", ".", "repository root when building")
	fs.IntVar(&c.depth, "depth", 0, "hop limit (0 = unbounded)")
	fs.BoolVar(&c.inferred, "inferred", false, "follow inferred edges")
	fs.StringVar(&c.rels, "relations", "", "comma-separated relations to follow")
	fs.IntVar(&c.budget, "budget", defaultBudget, "output token budget")
}

func (c *common) traverse() outline.TraverseOptions {
	opts := outline.TraverseOptions{Depth: c.depth, IncludeInferred: c.inferred}
	if c.rels != "" {
		opts.Relations = strings.Split(c.rels, ",")
	}
	return opts
}

func (c *common) load() (*outline.Graph, error) {
	if c.graph != "" {
		return loadGraph(c.graph)
	}
	return outline.Build(c.dir, outline.Options{})
}

func loadGraph(path string) (*outline.Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g outline.Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if g.SchemaVersion != outline.SchemaVersion {
		return nil, fmt.Errorf("%s: schema version %d, this build supports %d",
			path, g.SchemaVersion, outline.SchemaVersion)
	}
	return &g, nil
}

func cmdGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "write JSON to stdout")
	out := fs.String("o", "", "write JSON to file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	g, err := outline.Build(dir, outline.Options{})
	if err != nil {
		return err
	}
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		if err := g.JSON(f); err != nil {
			_ = f.Close()
			return err
		}
		return f.Close()
	}
	if *jsonOut {
		return g.JSON(os.Stdout)
	}
	fmt.Printf("%d nodes, %d edges, complete=%t\n", len(g.Nodes), len(g.Edges), g.Complete)
	for _, w := range g.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
	return nil
}

func cmdDef(args []string) error {
	fs := flag.NewFlagSet("def", flag.ExitOnError)
	var c common
	c.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("def requires a name")
	}
	g, err := c.load()
	if err != nil {
		return err
	}
	nodes := g.Def(fs.Arg(0))
	if len(nodes) == 0 {
		return fmt.Errorf("no node matches %q", fs.Arg(0))
	}
	ids := make([]string, len(nodes))
	for i := range nodes {
		ids[i] = nodes[i].ID
	}
	return g.Text(os.Stdout, ids, c.budget)
}

func cmdNeighbours(args []string, callers bool) error {
	fs := flag.NewFlagSet("neighbours", flag.ExitOnError)
	var c common
	c.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("requires a name")
	}
	g, err := c.load()
	if err != nil {
		return err
	}
	seed, err := resolveOne(g, fs.Arg(0))
	if err != nil {
		return err
	}
	var edges []outline.Edge
	if callers {
		edges = g.Callers(seed)
	} else {
		edges = g.Callees(seed)
	}
	ids := []string{seed}
	for _, e := range edges {
		if callers {
			ids = append(ids, e.From)
		} else {
			ids = append(ids, e.To)
		}
	}
	return g.Text(os.Stdout, ids, c.budget)
}

func cmdAffected(args []string) error {
	fs := flag.NewFlagSet("affected", flag.ExitOnError)
	var c common
	c.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("affected requires at least one seed")
	}
	g, err := c.load()
	if err != nil {
		return err
	}
	var seeds []string
	for _, name := range fs.Args() {
		id, err := resolveOne(g, name)
		if err != nil {
			return err
		}
		seeds = append(seeds, id)
	}
	paths := g.Affected(seeds, c.traverse())
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) < len(paths[j]) })
	return writePaths(g, paths, c.budget)
}

func cmdPath(args []string) error {
	fs := flag.NewFlagSet("path", flag.ExitOnError)
	var c common
	c.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < pathArgs {
		return fmt.Errorf("path requires <from> <to>")
	}
	g, err := c.load()
	if err != nil {
		return err
	}
	from, err := resolveOne(g, fs.Arg(0))
	if err != nil {
		return err
	}
	to, err := resolveOne(g, fs.Arg(1))
	if err != nil {
		return err
	}
	p := g.Path(from, to, c.traverse())
	if p == nil {
		return fmt.Errorf("no path from %s to %s", from, to)
	}
	return writePaths(g, []outline.Path{p}, c.budget)
}

// resolveOne turns a user-supplied seed into exactly one node ID, returning
// an error listing candidates if the name is ambiguous.
func resolveOne(g *outline.Graph, name string) (string, error) {
	nodes := g.Def(name)
	switch len(nodes) {
	case 0:
		return "", fmt.Errorf("no node matches %q", name)
	case 1:
		return nodes[0].ID, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d nodes match %q; use one of:\n", len(nodes), name)
	for _, n := range nodes {
		fmt.Fprintf(&b, "  %s  (%s %s:%d)\n", n.ID, n.Kind, n.File, n.Line)
	}
	return "", fmt.Errorf("%s", b.String())
}

func writePaths(g *outline.Graph, paths []outline.Path, budget int) error {
	seen := make(map[string]bool)
	var ids []string
	for _, p := range paths {
		for _, e := range p {
			if !seen[e.From] {
				seen[e.From] = true
				ids = append(ids, e.From)
			}
			if !seen[e.To] {
				seen[e.To] = true
				ids = append(ids, e.To)
			}
		}
	}
	fmt.Printf("%d paths, %d nodes\n", len(paths), len(ids))
	return g.Text(os.Stdout, ids, budget)
}
