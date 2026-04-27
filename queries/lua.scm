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
