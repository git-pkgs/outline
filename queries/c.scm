(comment) @keep

(preproc_include) @keep
(preproc_def) @keep
(preproc_function_def) @keep
(preproc_ifdef) @keep

(type_definition) @keep
(struct_specifier
  body: (field_declaration_list)) @keep
(union_specifier
  body: (field_declaration_list)) @keep
(enum_specifier
  body: (enumerator_list)) @keep

(declaration) @keep

(function_definition
  body: (compound_statement) @body) @signature
