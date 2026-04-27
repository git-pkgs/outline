package outline

import (
	"strings"
	"testing"
)

const goFixture = `// Package main is the entry point
package main

import (
	"fmt"
	"os"
)

// User represents a person
type User struct {
	Name string
	Age  int
}

// Greeter is something that can greet
type Greeter interface {
	Greet() string
}

const (
	MaxUsers = 100
	Version  = "1.0.0"
)

var (
	debugMode = false
	logLevel  = "info"
)

// SayHello prints a greeting message
func SayHello(name string) {
	fmt.Printf("Hello, %s!\n", name)
}

// Greet implements the Greeter interface
func (u User) Greet() string {
	return fmt.Sprintf("Hello, I'm %s!", u.Name)
}

func main() {
	user := User{Name: "John", Age: 30}
	fmt.Println(user.Greet())
	SayHello(os.Args[1])
}
`

func TestOutlineGo(t *testing.T) {
	got, ok := Outline([]byte(goFixture), "sample.go")
	if !ok {
		t.Fatal("go not supported")
	}
	t.Logf("output:\n%s", got)

	want := []string{
		"// Package main is the entry point",
		"package main",
		"import (",
		`"fmt"`,
		`"os"`,
		"type User struct {",
		"Name string",
		"Age  int",
		"type Greeter interface {",
		"Greet() string",
		"const (",
		"MaxUsers = 100",
		"var (",
		"debugMode = false",
		"// SayHello prints a greeting message",
		"func SayHello(name string) {",
		"// Greet implements the Greeter interface",
		"func (u User) Greet() string {",
		"func main() {",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing: %q", w)
		}
	}

	bodies := []string{
		"fmt.Printf",
		"fmt.Sprintf",
		"user := User{",
		"SayHello(os.Args",
	}
	for _, b := range bodies {
		if strings.Contains(got, b) {
			t.Errorf("body leaked: %q", b)
		}
	}

	if !strings.Contains(got, Separator) {
		t.Error("no separator emitted")
	}
}

const rubyFixture = `# User module for handling user-related functionality
module User
  ACTIVE = 1

  # Person class represents a human user
  class Person
    attr_accessor :name, :age

    # Initialize a new person
    def initialize(name, age)
      @name = name
      @age = age
    end

    def greet
      "Hello, I'm #{@name}"
    end

    def self.create(name, age)
      new(name, age)
    end
  end
end

require 'json'
require_relative './helpers'
`

func TestOutlineRuby(t *testing.T) {
	got, ok := Outline([]byte(rubyFixture), "sample.rb")
	if !ok {
		t.Fatal("ruby not supported")
	}
	t.Logf("output:\n%s", got)

	want := []string{
		"# User module",
		"module User",
		"ACTIVE = 1",
		"# Person class",
		"class Person",
		"attr_accessor :name, :age",
		"def initialize(name, age)",
		"def greet",
		"def self.create(name, age)",
		"require 'json'",
		"require_relative './helpers'",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing: %q", w)
		}
	}

	bodies := []string{
		"@name = name",
		"Hello, I'm",
		"new(name, age)\n",
	}
	for _, b := range bodies {
		if strings.Contains(got, b) {
			t.Errorf("body leaked: %q", b)
		}
	}
}

func TestUnsupported(t *testing.T) {
	if _, ok := Outline([]byte("hello"), "README.md"); ok {
		t.Error("markdown should not be supported")
	}
	if Supported("foo.txt") {
		t.Error("txt should not be supported")
	}
	if !Supported("foo.go") {
		t.Error("go should be supported")
	}
}

func TestIndexLines(t *testing.T) {
	src := []byte("a\nbb\n\nccc")
	starts, ends := indexLines(src)
	lines := []string{"a", "bb", "", "ccc"}
	if len(starts) != len(lines) || len(ends) != len(lines) {
		t.Fatalf("got %d/%d lines, want %d", len(starts), len(ends), len(lines))
	}
	for i, want := range lines {
		got := string(src[starts[i]:ends[i]])
		if got != want {
			t.Errorf("line %d: got %q want %q", i, got, want)
		}
	}
}
