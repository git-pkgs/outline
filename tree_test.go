package outline

import "testing"

func TestTree(t *testing.T) {
	got := Tree([]string{
		"README.md",
		"src/main.go",
		"src/util/helper.go",
		"src/util/helper_test.go",
		"cmd/app/main.go",
		"go.mod",
	})
	want := `├── cmd/
│   └── app/
│       └── main.go
├── src/
│   ├── util/
│   │   ├── helper.go
│   │   └── helper_test.go
│   └── main.go
├── README.md
└── go.mod
`
	if got != want {
		t.Errorf("tree mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTreeEmpty(t *testing.T) {
	if Tree(nil) != "" {
		t.Error("empty paths should produce empty tree")
	}
}

func TestTreeSingle(t *testing.T) {
	got := Tree([]string{"main.go"})
	want := "└── main.go\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
