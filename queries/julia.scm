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
