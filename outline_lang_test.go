package outline

import (
	"strings"
	"testing"
)

type langCase struct {
	filename string
	src      string
	want     []string
	drop     []string
}

func (tc langCase) run(t *testing.T) {
	t.Helper()
	got, ok := Outline([]byte(tc.src), tc.filename)
	if !ok {
		t.Fatalf("%s: not supported", tc.filename)
	}
	t.Logf("output:\n%s", got)
	for _, w := range tc.want {
		if !strings.Contains(got, w) {
			t.Errorf("%s missing: %q", tc.filename, w)
		}
	}
	for _, d := range tc.drop {
		if strings.Contains(got, d) {
			t.Errorf("%s leaked body: %q", tc.filename, d)
		}
	}
}

func TestOutlinePython(t *testing.T) {
	langCase{
		filename: "sample.py",
		src: `"""Module docstring."""
import os
from typing import Optional

MAX = 100

@dataclass
class User:
    """A user."""
    name: str

    def greet(self) -> str:
        return f"hi {self.name}"

def main():
    u = User("x")
    print(u.greet())
`,
		want: []string{
			"import os",
			"from typing import Optional",
			"MAX = 100",
			"@dataclass",
			"class User:",
			"def greet(self) -> str:",
			"def main():",
		},
		drop: []string{
			`return f"hi`,
			`u = User("x")`,
			"print(u.greet())",
		},
	}.run(t)
}

func TestOutlineJavascript(t *testing.T) {
	langCase{
		filename: "sample.js",
		src: `// header
import { thing } from './lib';

class Widget {
  constructor(name) {
    this.name = name;
  }
  render() {
    return this.name;
  }
}

function helper(x) {
  return x + 1;
}

const handler = (e) => {
  console.log(e);
};
`,
		want: []string{
			"// header",
			"import { thing } from './lib';",
			"class Widget {",
			"constructor(name) {",
			"render() {",
			"function helper(x) {",
			"const handler = (e) => {",
		},
		drop: []string{
			"this.name = name",
			"return this.name",
			"return x + 1",
			"console.log(e)",
		},
	}.run(t)
}

func TestOutlineTypescript(t *testing.T) {
	langCase{
		filename: "sample.ts",
		src: `import type { Foo } from './types';

export interface User {
  id: number;
  name: string;
}

export type ID = string | number;

enum Color { Red, Green }

abstract class Base {
  abstract run(): void;
  protected value: number;
}

export class Impl extends Base {
  run(): void {
    doThing();
  }
}

export const make = (id: ID): User => {
  return { id, name: '' };
};
`,
		want: []string{
			"import type { Foo }",
			"export interface User {",
			"id: number;",
			"export type ID = string | number;",
			"enum Color { Red, Green }",
			"abstract class Base {",
			"abstract run(): void;",
			"protected value: number;",
			"export class Impl extends Base {",
			"run(): void {",
			"export const make = (id: ID): User => {",
		},
		drop: []string{
			"doThing()",
			"return { id, name:",
		},
	}.run(t)
}

func TestOutlineRust(t *testing.T) {
	langCase{
		filename: "sample.rs",
		src: `//! Crate docs.
use std::fmt;

#[derive(Debug)]
pub struct User {
    pub name: String,
}

pub trait Greet {
    fn greet(&self) -> String;
}

impl Greet for User {
    fn greet(&self) -> String {
        format!("hi {}", self.name)
    }
}

pub const MAX: u32 = 100;

pub fn run() {
    let u = User { name: "x".into() };
    println!("{}", u.greet());
}
`,
		want: []string{
			"//! Crate docs.",
			"use std::fmt;",
			"#[derive(Debug)]",
			"pub struct User {",
			"pub name: String,",
			"pub trait Greet {",
			"fn greet(&self) -> String;",
			"impl Greet for User {",
			"fn greet(&self) -> String {",
			"pub const MAX: u32 = 100;",
			"pub fn run() {",
		},
		drop: []string{
			"format!(",
			"let u = User",
			"println!(",
		},
	}.run(t)
}

func TestOutlineJava(t *testing.T) {
	langCase{
		filename: "Sample.java",
		src: `package com.example;

import java.util.List;

/** A user. */
public class User {
    private final String name;

    public User(String name) {
        this.name = name;
    }

    public String greet() {
        return "hi " + name;
    }
}
`,
		want: []string{
			"package com.example;",
			"import java.util.List;",
			"/** A user. */",
			"public class User {",
			"private final String name;",
			"public User(String name) {",
			"public String greet() {",
		},
		drop: []string{
			"this.name = name",
			`return "hi " + name`,
		},
	}.run(t)
}

func TestAllQueriesCompile(t *testing.T) {
	for name, l := range langs {
		l.init()
		if l.err != nil {
			t.Errorf("%s: %v", name, l.err)
		}
	}
}
