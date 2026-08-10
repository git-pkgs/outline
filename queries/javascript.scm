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

(class_declaration
  name: (identifier) @symbol.name) @symbol.class
(function_declaration
  name: (identifier) @symbol.name) @symbol.func
(generator_function_declaration
  name: (identifier) @symbol.name) @symbol.func
(method_definition
  name: (property_identifier) @symbol.name) @symbol.func

(lexical_declaration
  "const"
  (variable_declarator
    name: (identifier) @symbol.name
    value: (arrow_function))) @symbol.const
(lexical_declaration
  "let"
  (variable_declarator
    name: (identifier) @symbol.name
    value: (arrow_function))) @symbol.var
