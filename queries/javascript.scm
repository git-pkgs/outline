(comment) @keep

(import_statement) @keep
(export_statement
  !declaration
  !source) @keep

(class_declaration
  body: (class_body) @body) @signature

(function_declaration
  body: (statement_block) @body) @signature

(generator_function_declaration
  body: (statement_block) @body) @signature

(method_definition
  body: (statement_block) @body) @signature

(lexical_declaration
  (variable_declarator
    value: (arrow_function
      body: (statement_block) @body))) @signature
