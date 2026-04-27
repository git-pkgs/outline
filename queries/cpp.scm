(comment) @keep

(preproc_include) @keep
(preproc_def) @keep
(preproc_function_def) @keep

(using_declaration) @keep
(alias_declaration) @keep
(type_definition) @keep

(namespace_definition
  body: (declaration_list) @body) @signature

(struct_specifier
  body: (field_declaration_list) @body) @signature
(class_specifier
  body: (field_declaration_list) @body) @signature
(union_specifier
  body: (field_declaration_list)) @keep
(enum_specifier
  body: (enumerator_list)) @keep

(field_declaration) @keep
(access_specifier) @keep
(declaration) @keep

(template_declaration
  (function_definition
    body: (compound_statement) @body)) @signature
(template_declaration
  (class_specifier
    body: (field_declaration_list) @body)) @signature

(function_definition
  body: (compound_statement) @body) @signature
