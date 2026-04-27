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
