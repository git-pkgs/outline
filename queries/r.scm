(comment) @keep

(call
  function: (identifier) @_f
  (#match? @_f "^(library|require|source|loadNamespace)$")) @keep

(binary_operator
  rhs: (function_definition
    body: (braced_expression) @body)) @signature

(binary_operator
  lhs: (identifier)
  rhs: (_)) @keep
