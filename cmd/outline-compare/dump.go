package main

import (
	"flag"
	"fmt"
	"os"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

var dumpLang = flag.String("dump", "", "dump SExpr for stdin in given language")

func init() {
	flag.Parse()
	if *dumpLang == "" {
		return
	}
	entry := grammars.DetectLanguageByName(*dumpLang)
	if entry == nil {
		fmt.Fprintf(os.Stderr, "unknown language: %s\n", *dumpLang)
		os.Exit(1)
	}
	lang := entry.Language()
	src, _ := os.ReadFile("/dev/stdin")
	tree, err := ts.NewParser(lang).Parse(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(tree.RootNode().SExpr(lang))
	os.Exit(0)
}
