(comment) @keep

(unary_operator
  operand: (call
    target: (identifier) @_attr)
  (#match? @_attr "^(moduledoc|doc|typedoc|spec|type|typep|callback|behaviour|impl)$")) @keep

(call
  target: (identifier) @_def
  (do_block) @body
  (#match? @_def "^(defmodule|defprotocol|defimpl)$")) @signature

(call
  target: (identifier) @_def
  (do_block) @body
  (#match? @_def "^(def|defp|defmacro|defmacrop|defguard|defguardp|defdelegate)$")) @signature

(call
  target: (identifier) @_def
  (#match? @_def "^(def|defp|defmacro|defmacrop|defstruct|defexception|use|import|alias|require)$")) @keep

(call
  target: (identifier) @_def
  (arguments
    .
    (alias) @symbol.name)
  (#match? @_def "^(defmodule|defprotocol|defimpl)$")) @symbol.type

(call
  target: (identifier) @_def
  (arguments
    .
    (call
      target: (identifier) @symbol.name))
  (#match? @_def "^(def|defp|defmacro|defmacrop|defguard|defguardp|defdelegate)$")) @symbol.func
