(comment) @keep

(import_statement) @keep
(export_statement
  !declaration
  !source) @keep

(interface_declaration) @keep
(type_alias_declaration) @keep
(enum_declaration) @keep
(function_signature) @keep
(ambient_declaration) @keep

(class_declaration
  body: (class_body) @body) @signature

(abstract_class_declaration
  body: (class_body) @body) @signature

(function_declaration
  body: (statement_block) @body) @signature

(generator_function_declaration
  body: (statement_block) @body) @signature

(method_definition
  body: (statement_block) @body) @signature

(method_signature) @keep
(abstract_method_signature) @keep
(public_field_definition) @keep

(lexical_declaration
  (variable_declarator
    value: (arrow_function
      body: (statement_block) @body))) @signature

(interface_declaration
  (type_identifier) @symbol.name) @symbol.type
(type_alias_declaration
  (type_identifier) @symbol.name) @symbol.type
(enum_declaration
  (identifier) @symbol.name) @symbol.type
(function_signature
  (identifier) @symbol.name) @symbol.func

(class_declaration
  name: (type_identifier) @symbol.name) @symbol.class
(abstract_class_declaration
  name: (type_identifier) @symbol.name) @symbol.class
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
