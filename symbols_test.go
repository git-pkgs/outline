package outline

import "testing"

const goSymFixture = `package main

import "fmt"

type Server struct{ addr string }

func (s *Server) Run() error {
	return announce(s.addr)
}

func announce(addr string) error {
	fmt.Println(addr)
	return nil
}

func main() {
	s := &Server{addr: ":8080"}
	s.Run()
}
`

func TestExtractGo(t *testing.T) {
	syms, refs, ok := Extract([]byte(goSymFixture), "main.go")
	if !ok {
		t.Fatal("go should be supported for symbol extraction")
	}

	wantSyms := map[string]SymbolKind{
		"Server":   KindType,
		"Run":      KindMethod,
		"announce": KindFunc,
		"main":     KindFunc,
	}
	got := make(map[string]SymbolKind)
	for _, s := range syms {
		got[s.Name] = s.Kind
		if s.Name == "Run" && s.Container != "Server" {
			t.Errorf("Run container = %q, want Server", s.Container)
		}
	}
	for name, kind := range wantSyms {
		if got[name] != kind {
			t.Errorf("symbol %s: got kind %q, want %q", name, got[name], kind)
		}
	}

	wantCalls := map[string]bool{"announce": false, "Run": false, "Println": false}
	for _, r := range refs {
		if r.Kind == RefCall {
			if _, ok := wantCalls[r.Target]; ok {
				wantCalls[r.Target] = true
			}
		}
	}
	for name, seen := range wantCalls {
		if !seen {
			t.Errorf("missing call ref to %s", name)
		}
	}
}

const rubySymFixture = `require "json"

module Shop
  class Cart < Base
    include Pricing

    def total
      sum_items
    end

    def sum_items
      0
    end
  end
end
`

func TestExtractRuby(t *testing.T) {
	syms, refs, ok := Extract([]byte(rubySymFixture), "cart.rb")
	if !ok {
		t.Fatal("ruby should be supported for symbol extraction")
	}

	got := make(map[string]SymbolKind)
	for _, s := range syms {
		got[s.Name] = s.Kind
	}
	if got["Shop"] != KindModule {
		t.Errorf("Shop kind = %q, want module", got["Shop"])
	}
	if got["Cart"] != KindType {
		t.Errorf("Cart kind = %q, want type", got["Cart"])
	}
	if got["total"] != KindMethod {
		t.Errorf("total kind = %q, want method", got["total"])
	}

	for _, s := range syms {
		if s.Name == "total" && s.Container != "Cart" {
			t.Errorf("total container = %q, want Cart", s.Container)
		}
	}

	var sawInherit, sawImport, sawCall bool
	for _, r := range refs {
		switch {
		case r.Kind == RefInherit && (r.Target == "Base" || r.Target == "Pricing"):
			sawInherit = true
		case r.Kind == RefImport && r.Target == "json":
			sawImport = true
		case r.Kind == RefCall && r.Target == "sum_items":
			sawCall = true
		}
	}
	if !sawInherit {
		t.Error("missing inherit ref")
	}
	if !sawImport {
		t.Error("missing import ref")
	}
	if !sawCall {
		t.Error("missing call ref to sum_items")
	}
}

func TestExtractUnsupported(t *testing.T) {
	if _, _, ok := Extract([]byte("x = 1"), "x.py"); ok {
		t.Error("python has no symbol query yet, want ok=false")
	}
}

func TestBuildGraphResolves(t *testing.T) {
	g, err := BuildGraph(".", Options{Ignore: []string{"claude/"}})
	if err != nil {
		t.Fatal(err)
	}
	st := g.Stats()
	if st.Symbols == 0 {
		t.Fatal("expected symbols from this repo")
	}
	if st.Refs[RefCall] == 0 {
		t.Fatal("expected call refs from this repo")
	}
	callers := g.Callers("detect")
	if len(callers) == 0 {
		t.Error("expected at least one caller of detect()")
	}
}
