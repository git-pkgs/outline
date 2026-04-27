package outline

import "testing"

func TestOutlineScala(t *testing.T) {
	langCase{
		filename: "Sample.scala",
		src: `package com.example

import java.io._

trait Greeter {
  def greet: String
}

class User(name: String) extends Greeter {
  val upper = name.toUpperCase
  def greet: String = {
    s"hi $name"
  }
}

object Main {
  def main(args: Array[String]): Unit = {
    println(new User("x").greet)
  }
}
`,
		want: []string{
			"package com.example",
			"import java.io._",
			"trait Greeter {",
			"def greet: String",
			"class User(name: String) extends Greeter {",
			"val upper = name.toUpperCase",
			"def greet: String = {",
			"object Main {",
			"def main(args: Array[String]): Unit = {",
		},
		drop: []string{`s"hi $name"`, "println(new User"},
	}.run(t)
}

func TestOutlineDart(t *testing.T) {
	langCase{
		filename: "sample.dart",
		src: `import 'dart:io';

class User {
  final String name;
  User(this.name);

  String greet() {
    return 'hi $name';
  }
}

void main() {
  print(User('x').greet());
}
`,
		want: []string{
			"import 'dart:io';",
			"class User {",
			"final String name;",
			"User(this.name);",
			"String greet() {",
			"void main() {",
		},
		drop: []string{"return 'hi", "print(User"},
	}.run(t)
}

func TestOutlineElixir(t *testing.T) {
	langCase{
		filename: "sample.ex",
		src: `defmodule MyApp.User do
  @moduledoc "A user."

  use GenServer
  alias MyApp.Repo

  defstruct [:name, :age]

  @spec greet(String.t()) :: String.t()
  def greet(name) do
    IO.puts("hi #{name}")
  end

  defp helper, do: :ok
end
`,
		want: []string{
			"defmodule MyApp.User do",
			`@moduledoc "A user."`,
			"use GenServer",
			"alias MyApp.Repo",
			"defstruct [:name, :age]",
			"@spec greet(String.t()) :: String.t()",
			"def greet(name) do",
			"defp helper, do: :ok",
		},
		drop: []string{"IO.puts("},
	}.run(t)
}

func TestOutlineErlang(t *testing.T) {
	langCase{
		filename: "sample.erl",
		src: `-module(user).
-export([greet/1]).

-spec greet(string()) -> ok.
greet(Name) ->
    io:format("hi ~s~n", [Name]),
    ok.
`,
		want: []string{
			"-module(user).",
			"-export([greet/1]).",
			"-spec greet(string()) -> ok.",
			"greet(Name) ->",
		},
		drop: []string{"io:format("},
	}.run(t)
}

func TestOutlineHaskell(t *testing.T) {
	langCase{
		filename: "Sample.hs",
		src: `module Sample where

import Data.List

data User = User { name :: String }

greet :: User -> String
greet u =
  "hi " ++ name u

main :: IO ()
main =
  putStrLn (greet (User "x"))
`,
		want: []string{
			"module Sample where",
			"import Data.List",
			"data User = User { name :: String }",
			"greet :: User -> String",
			"main :: IO ()",
		},
		drop: []string{`"hi " ++ name u`, "putStrLn (greet"},
	}.run(t)
}

func TestOutlineClojure(t *testing.T) {
	langCase{
		filename: "sample.clj",
		src: `(ns my.app
  (:require [clojure.string :as str]))

(def max-users 100)

(defn greet [name]
  (println "hi" name))

(defn- helper [x]
  (inc x))
`,
		want: []string{
			"(ns my.app",
			"(:require [clojure.string :as str]))",
			"(def max-users 100)",
			"(defn greet [name]",
			"(defn- helper [x]",
		},
		drop: []string{`(println "hi"`, "(inc x)"},
	}.run(t)
}
