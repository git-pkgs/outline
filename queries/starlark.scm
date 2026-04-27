(comment) @keep

(module
  (expression_statement
    (call
      function: (identifier) @_f
      (#eq? @_f "load"))) @keep)

(module
  (expression_statement
    (assignment)) @keep)

(module
  (expression_statement
    (call)) @signature)

(function_definition
  body: (block) @body) @signature
