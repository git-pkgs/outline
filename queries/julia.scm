(line_comment) @keep
(block_comment) @keep

(module_definition
  (block) @body) @signature
(using_statement) @keep
(import_statement) @keep
(export_statement) @keep

(struct_definition) @keep
(abstract_definition) @keep
(primitive_definition) @keep
(const_statement) @keep

(function_definition
  (block) @body) @signature
(function_definition) @keep
(macro_definition
  (block) @body) @signature
(assignment
  (call_expression)
  (_)) @keep

(module_definition
  (identifier) @symbol.name) @symbol.type
(struct_definition
  (type_head
    (identifier) @symbol.name)) @symbol.type
(abstract_definition
  (type_head
    (identifier) @symbol.name)) @symbol.type
(primitive_definition
  (type_head
    (identifier) @symbol.name)) @symbol.type
(const_statement
  (assignment
    (identifier) @symbol.name)) @symbol.const
(function_definition
  (signature
    (call_expression
      .
      (identifier) @symbol.name))) @symbol.func
(macro_definition
  (signature
    (call_expression
      .
      (identifier) @symbol.name))) @symbol.func
(assignment
  .
  (call_expression
    .
    (identifier) @symbol.name)) @symbol.func
