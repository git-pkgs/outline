(comment) @keep

(module_declaration) @keep
(import_declaration) @keep

(struct_declaration
  (aggregate_body) @body) @signature
(class_declaration
  (aggregate_body) @body) @signature
(interface_declaration
  (aggregate_body) @body) @signature
(union_declaration
  (aggregate_body) @body) @signature
(enum_declaration) @keep
(template_declaration
  (aggregate_body) @body) @signature

(variable_declaration) @keep
(alias_declaration) @keep

(function_declaration
  (function_body
    (block_statement) @body)) @signature
(function_declaration) @keep

(struct_declaration
  (identifier) @symbol.name) @symbol.type
(class_declaration
  (identifier) @symbol.name) @symbol.class
(interface_declaration
  (identifier) @symbol.name) @symbol.type
(union_declaration
  (identifier) @symbol.name) @symbol.type
(enum_declaration
  (identifier) @symbol.name) @symbol.type
(alias_declaration
  (alias_initializer
    (identifier) @symbol.name)) @symbol.type
(function_declaration
  (identifier) @symbol.name) @symbol.func
