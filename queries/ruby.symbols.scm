(class
  name: [(constant) (scope_resolution)] @name) @def.type

(module
  name: [(constant) (scope_resolution)] @name) @def.module

(method
  name: (identifier) @name) @def.method

(singleton_method
  name: (identifier) @name) @def.method

(assignment
  left: (constant) @name) @def.const

(class
  superclass: (superclass
    [(constant) (scope_resolution)] @name)) @ref.inherit

(call
  method: (identifier) @_m
  arguments: (argument_list
    [(constant) (scope_resolution)] @name)
  (#match? @_m "^(include|extend|prepend)$")) @ref.inherit

(call
  method: (identifier) @_m
  arguments: (argument_list
    (string (string_content) @name))
  (#match? @_m "^(require|require_relative|load|autoload)$")) @ref.import

(call
  !receiver
  method: (identifier) @name
  (#not-match? @name "^(require|require_relative|load|autoload|include|extend|prepend|attr_reader|attr_writer|attr_accessor|private|public|protected|raise|puts|print|p|new|lambda|proc)$")) @ref.call

; bare identifier in statement position: Ruby method call without parens/args.
; over-captures local-var references but those won't resolve to a symbol.
(body_statement
  (identifier) @name @ref.call
  (#not-match? @name "^(nil|true|false|self|super)$"))

(call
  receiver: (_)
  method: (identifier) @name
  (#not-match? @name "^(new|each|map|select|reject|find|length|size|to_s|to_i|to_a|to_h|nil\\?|empty\\?|first|last|push|pop|<<|\\[\\])$")) @ref.call

(scope_resolution
  name: (constant) @name) @ref.type

(constant) @name @ref.type
