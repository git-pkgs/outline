package outline

import "testing"

func TestOutlineC(t *testing.T) {
	langCase{
		filename: "sample.c",
		src: `#include <stdio.h>

#define MAX 100

// A point.
struct Point {
    int x;
    int y;
};

typedef struct Point Point;

int add(int a, int b) {
    return a + b;
}

int main(void) {
    printf("%d\n", add(1, 2));
    return 0;
}
`,
		want: []string{
			"#include <stdio.h>",
			"#define MAX 100",
			"// A point.",
			"struct Point {",
			"int x;",
			"typedef struct Point Point;",
			"int add(int a, int b) {",
			"int main(void) {",
		},
		drop: []string{"return a + b", "printf("},
	}.run(t)
}

func TestOutlineCpp(t *testing.T) {
	langCase{
		filename: "sample.cpp",
		src: `#include <vector>

namespace app {

class Widget {
public:
    Widget(int id);
    int id() const { return id_; }
private:
    int id_;
};

template<typename T>
T max(T a, T b) {
    return a > b ? a : b;
}

}  // namespace app
`,
		want: []string{
			"#include <vector>",
			"namespace app {",
			"class Widget {",
			"public:",
			"Widget(int id);",
			"int id() const {",
			"private:",
			"int id_;",
			"template<typename T>",
			"T max(T a, T b) {",
		},
		drop: []string{"return a > b"},
	}.run(t)
}

func TestOutlineCSharp(t *testing.T) {
	langCase{
		filename: "Sample.cs",
		src: `using System;

namespace App;

public interface IGreeter {
    string Greet();
}

public class User : IGreeter {
    public string Name { get; set; }
    private int age;

    public User(string name) {
        Name = name;
    }

    public string Greet() {
        return $"hi {Name}";
    }
}
`,
		want: []string{
			"using System;",
			"namespace App;",
			"public interface IGreeter {",
			"string Greet();",
			"public class User : IGreeter {",
			"public string Name { get; set; }",
			"private int age;",
			"public User(string name) {",
			"public string Greet() {",
		},
		drop: []string{"Name = name;", "return $\"hi"},
	}.run(t)
}

func TestOutlinePHP(t *testing.T) {
	langCase{
		filename: "sample.php",
		src: `<?php

namespace App;

use Foo\Bar;

interface Greeter {
    public function greet(): string;
}

class User implements Greeter {
    public string $name;

    public function __construct(string $name) {
        $this->name = $name;
    }

    public function greet(): string {
        return "hi {$this->name}";
    }
}

function helper() {
    doThing();
}
`,
		want: []string{
			"<?php",
			"namespace App;",
			"use Foo\\Bar;",
			"interface Greeter {",
			"public function greet(): string;",
			"class User implements Greeter {",
			"public string $name;",
			"public function __construct(string $name) {",
			"public function greet(): string {",
			"function helper() {",
		},
		drop: []string{"$this->name = $name", `return "hi`, "doThing()"},
	}.run(t)
}

func TestOutlineKotlin(t *testing.T) {
	langCase{
		filename: "Sample.kt",
		src: `package com.example

import java.io.File

interface Greeter {
    fun greet(): String
}

class User(val name: String) : Greeter {
    override fun greet(): String {
        return "hi $name"
    }
}

fun main() {
    println(User("x").greet())
}
`,
		want: []string{
			"package com.example",
			"import java.io.File",
			"interface Greeter {",
			"fun greet(): String",
			"class User(val name: String) : Greeter {",
			"override fun greet(): String {",
			"fun main() {",
		},
		drop: []string{`return "hi`, "println("},
	}.run(t)
}

func TestOutlineSwift(t *testing.T) {
	langCase{
		filename: "Sample.swift",
		src: `import Foundation

protocol Greeter {
    func greet() -> String
}

struct User: Greeter {
    let name: String

    func greet() -> String {
        return "hi \(name)"
    }
}

func main() {
    print(User(name: "x").greet())
}
`,
		want: []string{
			"import Foundation",
			"protocol Greeter {",
			"func greet() -> String",
			"struct User: Greeter {",
			"let name: String",
			"func greet() -> String {",
			"func main() {",
		},
		drop: []string{`return "hi`, "print(User"},
	}.run(t)
}
