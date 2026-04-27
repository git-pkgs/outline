package main

import (
	"fmt"
	"os"

	"github.com/git-pkgs/outline"
)

func main() {
	for _, path := range os.Args[1:] {
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
