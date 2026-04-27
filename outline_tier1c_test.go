package outline

import "testing"

func TestOutlinePerl(t *testing.T) {
	langCase{
		filename: "Sample.pm",
		src: `package My::Mod;
use strict;
use warnings;

# Greet someone
sub hello {
    my ($name) = @_;
    print "hi $name\n";
}

1;
`,
		want: []string{"package My::Mod;", "use strict;", "# Greet someone", "sub hello {"},
		drop: []string{"my ($name)", `print "hi`},
	}.run(t)
}

func TestOutlineLua(t *testing.T) {
	langCase{
		filename: "sample.lua",
		src: `local M = {}

-- Greet someone
function M.hello(name)
    print("hi " .. name)
end

local function priv()
    return 1
end

return M
`,
		want: []string{"local M = {}", "-- Greet someone", "function M.hello(name)", "local function priv()", "return M"},
		drop: []string{`print("hi`, "return 1"},
	}.run(t)
}

func TestOutlineR(t *testing.T) {
	langCase{
		filename: "sample.R",
		src: `library(dplyr)

#' Greet someone
hello <- function(name) {
    print(paste("hi", name))
}

x <- 1
`,
		want: []string{"library(dplyr)", "#' Greet someone", "hello <- function(name) {", "x <- 1"},
		drop: []string{"print(paste"},
	}.run(t)
}

func TestOutlineJulia(t *testing.T) {
	langCase{
		filename: "sample.jl",
		src: `module M

using Pkg

struct User
    name::String
end

function greet(u::User)
    println("hi $(u.name)")
end

short(x) = x + 1

end
`,
		want: []string{"module M", "using Pkg", "struct User", "name::String", "function greet(u::User)", "short(x) = x + 1"},
		drop: []string{`println("hi`},
	}.run(t)
}

func TestOutlineOCaml(t *testing.T) {
	langCase{
		filename: "sample.ml",
		src: `open Printf

type user = { name: string }

let greet u =
  printf "hi %s\n" u.name

let x = 1
`,
		want: []string{"open Printf", "type user = { name: string }", "let greet u =", "let x = 1"},
		drop: []string{},
	}.run(t)
}

func TestOutlineFSharp(t *testing.T) {
	langCase{
		filename: "Sample.fs",
		src: `module M

open System

type User = { Name: string }

let greet u =
    printfn "hi %s" u.Name
`,
		want: []string{"module M", "open System", "type User = { Name: string }", "let greet u ="},
		drop: []string{},
	}.run(t)
}

func TestOutlineCrystal(t *testing.T) {
	langCase{
		filename: "sample.cr",
		src: `require "json"

class User
  getter name : String

  def initialize(@name)
  end

  def greet
    puts "hi #{@name}"
  end
end
`,
		want: []string{`require "json"`, "class User", "getter name : String", "def initialize(@name)", "def greet"},
		drop: []string{},
	}.run(t)
}

func TestOutlineNim(t *testing.T) {
	langCase{
		filename: "sample.nim",
		src: `import strutils

type
  User = object
    name: string

proc greet(u: User): string =
  result = "hi " & u.name
`,
		want: []string{"import strutils", "type", "User = object", "name: string", "proc greet(u: User): string ="},
		drop: []string{`result = "hi`},
	}.run(t)
}

func TestOutlineZig(t *testing.T) {
	langCase{
		filename: "sample.zig",
		src: `const std = @import("std");

pub const User = struct {
    name: []const u8,

    pub fn greet(self: User) void {
        std.debug.print("hi {s}\n", .{self.name});
    }
};

pub fn main() void {
    const u = User{ .name = "x" };
    u.greet();
}
`,
		want: []string{`const std = @import("std");`, "pub const User = struct {", "name: []const u8,", "pub fn greet(self: User) void {", "pub fn main() void {"},
		drop: []string{"std.debug.print(", "u.greet();"},
	}.run(t)
}

func TestOutlineD(t *testing.T) {
	langCase{
		filename: "sample.d",
		src: `module app;

import std.stdio;

struct User {
    string name;
}

void greet(User u) {
    writeln("hi ", u.name);
}
`,
		want: []string{"module app;", "import std.stdio;", "struct User {", "string name;", "void greet(User u) {"},
		drop: []string{"writeln("},
	}.run(t)
}

func TestOutlineGroovy(t *testing.T) {
	langCase{
		filename: "Sample.groovy",
		src: `package com.x

import java.io.*

class User {
    String name

    String greet() {
        return "hi $name"
    }
}

def main() {
    println(new User(name: "x").greet())
}
`,
		want: []string{"package com.x", "import java.io.*", "class User {", "String name", "String greet() {", "def main() {"},
		drop: []string{`return "hi`, "println(new"},
	}.run(t)
}
