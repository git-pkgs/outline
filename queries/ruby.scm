(comment) @keep

(assignment
  left: (constant)) @keep

(class
  name: (_)
  body: (_) @body) @signature

(singleton_class
  body: (_) @body) @signature

(module
  name: (_)
  body: (_) @body) @signature

(method
  body: (body_statement) @body) @signature

(method
  !body) @keep

(singleton_method
  body: (body_statement) @body) @signature

(singleton_method
  !body) @keep

(call
  method: (identifier) @_m
  (#match? @_m "^(require|require_relative|load|autoload|attr_reader|attr_writer|attr_accessor|include|extend|prepend)$")) @keep

(assignment
  left: (constant) @symbol.name) @symbol.const
(class
  name: (_) @symbol.name) @symbol.class
(module
  name: (_) @symbol.name) @symbol.type
(method
  name: [(identifier) (constant)] @symbol.name) @symbol.func
(singleton_method
  name: [(identifier) (constant)] @symbol.name) @symbol.func
