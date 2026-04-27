(comment) @keep

(variable_declaration
  (struct_declaration) @body) @signature
(variable_declaration
  (enum_declaration) @body) @signature
(variable_declaration
  (union_declaration) @body) @signature
(variable_declaration) @keep

(container_field) @keep

(function_declaration
  body: (block) @body) @signature
(function_declaration) @keep

(test_declaration
  body: (block) @body) @signature
