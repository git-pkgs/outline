package outline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/git-pkgs/outline"
)

func TestBuildRubyShebangInterpreter(t *testing.T) {
	for _, tc := range []struct {
		name, line string
		ruby       bool
	}{
		{"direct", "#!/usr/bin/ruby", true},
		{"direct argument", "#!/usr/bin/ruby -w", true},
		{"numeric suffix", "#!/usr/bin/ruby3", true},
		{"dotted suffix", "#!/usr/bin/ruby3.3", true},
		{"env", "#!/usr/bin/env ruby", true},
		{"env split", "#!/usr/bin/env -S ruby -w", true},
		{"env attached split", "#!/usr/bin/env -Sruby -w", true},
		{"env long split", "#!/usr/bin/env --split-string=ruby -w", true},
		{"env assignment", "#!/usr/bin/env -S RUBYOPT=-w ruby", true},
		{"env unset", "#!/usr/bin/env -S -u RUBYOPT ruby", true},
		{"env chdir", "#!/usr/bin/env -S -C /tmp ruby", true},
		{"env flags", "#!/usr/bin/env -S -i -- ruby", true},
		{"env long options", "#!/usr/bin/env -S --ignore-environment --unset=RUBYOPT --chdir=/tmp ruby", true},
		{"shell argument", "#!/bin/sh ruby", false},
		{"env shell argument", "#!/usr/bin/env -S sh ruby", false},
		{"env unset operand", "#!/usr/bin/env -S -u ruby sh", false},
		{"env chdir operand", "#!/usr/bin/env -S -C ruby sh", false},
		{"env assignment value", "#!/usr/bin/env -S LANGUAGE=ruby sh", false},
		{"env ended options", "#!/usr/bin/env -S -- -u ruby", false},
		{"env missing operand", "#!/usr/bin/env -S -u ruby", false},
		{"env unknown option", "#!/usr/bin/env --unknown ruby", false},
		{"env help", "#!/usr/bin/env --help ruby", false},
		{"env quoted operand", "#!/usr/bin/env -S -u 'x ruby y' sh", false},
		{"env expansion", "#!/usr/bin/env -S ${INTERPRETER} ruby", false},
		{"env comment operand", "#!/usr/bin/env -S -u # ruby", false},
		{"env directory", "#!/usr/bin/env/ ruby", false},
		{"ruby directory", "#!/usr/bin/ruby/", false},
		{"lookalike", "#!/usr/bin/ruby-not-an-interpreter", false},
		{"suffix letters", "#!/usr/bin/ruby3helper", false},
		{"suffix trailing dot", "#!/usr/bin/ruby3.", false},
		{"suffix double dot", "#!/usr/bin/ruby3..3", false},
		{"empty", "#!", false},
		{"not first line", "\n#!/usr/bin/ruby", false},
		{"tabs and CRLF", "#!\t/usr/bin/env\t-S\truby\r", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			src := []byte(tc.line + "\nrequire 'json'\n")
			if err := os.WriteFile(filepath.Join(root, "tool"), src, 0700); err != nil {
				t.Fatal(err)
			}
			if got := outline.SupportedSource(src, "tool"); got != tc.ruby {
				t.Errorf("SupportedSource = %v, want %v", got, tc.ruby)
			}
			graph, err := outline.Build(root, outline.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if graph.Node(outline.FileID("tool")) == nil {
				t.Fatal("missing file node")
			}
			var loaded bool
			for _, edge := range graph.Edges {
				if edge.From == outline.FileID("tool") && edge.To == outline.ModID("ruby", "json") && edge.Rel == outline.RelLoads {
					loaded = true
				}
			}
			if loaded != tc.ruby {
				t.Errorf("Ruby json load = %v, want %v", loaded, tc.ruby)
			}
		})
	}
}
