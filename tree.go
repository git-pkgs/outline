package outline

import (
	"sort"
	"strings"
)

type treeNode struct {
	name     string
	children map[string]*treeNode
	isDir    bool
}

func (n *treeNode) child(name string, isDir bool) *treeNode {
	if n.children == nil {
		n.children = make(map[string]*treeNode)
	}
	c, ok := n.children[name]
	if !ok {
		c = &treeNode{name: name, isDir: isDir}
		n.children[name] = c
	}
	if isDir {
		c.isDir = true
	}
	return c
}

func (n *treeNode) sorted() []*treeNode {
	out := make([]*treeNode, 0, len(n.children))
	for _, c := range n.children {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].isDir != out[j].isDir {
			return out[i].isDir
		}
		return out[i].name < out[j].name
	})
	return out
}

// Tree renders a sorted list of slash-separated relative paths as a
// box-drawing directory tree. Directories are inferred from path segments
// and listed before files.
func Tree(paths []string) string {
	root := &treeNode{isDir: true}
	for _, p := range paths {
		p = strings.TrimPrefix(p, "./")
		parts := strings.Split(p, "/")
		cur := root
		for i, part := range parts {
			if part == "" {
				continue
			}
			cur = cur.child(part, i < len(parts)-1)
		}
	}

	var b strings.Builder
	writeTree(&b, root, "")
	return b.String()
}

const (
	treeBranch = "├── "
	treeLast   = "└── "
	treePipe   = "│   "
	treeSpace  = "    "
)

func writeTree(b *strings.Builder, n *treeNode, prefix string) {
	children := n.sorted()
	for i, c := range children {
		last := i == len(children)-1
		b.WriteString(prefix)
		if last {
			b.WriteString(treeLast)
		} else {
			b.WriteString(treeBranch)
		}
		b.WriteString(c.name)
		if c.isDir {
			b.WriteByte('/')
		}
		b.WriteByte('\n')
		if c.isDir {
			next := prefix + treePipe
			if last {
				next = prefix + treeSpace
			}
			writeTree(b, c, next)
		}
	}
}
