(comment) @keep

(function_declaration
  body: (block) @body) @signature
(function_declaration) @keep

(variable_declaration
  (assignment_statement
    (expression_list
      (function_definition
        body: (block) @body)))) @signature

(chunk (return_statement) @keep)
(chunk (variable_declaration) @keep)

(function_declaration
  name: (identifier) @symbol.name) @symbol.func
(function_declaration
  name: (dot_index_expression) @symbol.name) @symbol.func
(function_declaration
  name: (method_index_expression) @symbol.name) @symbol.func

(chunk
  (variable_declaration
    (assignment_statement
      (variable_list
        (identifier) @symbol.name))) @symbol.var)
