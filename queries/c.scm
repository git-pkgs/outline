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

(preproc_def
  name: (identifier) @symbol.name) @symbol.const
(preproc_function_def
  name: (identifier) @symbol.name) @symbol.func
(type_definition
  (type_identifier) @symbol.name) @symbol.type
(struct_specifier
  name: (type_identifier) @symbol.name) @symbol.type
(union_specifier
  name: (type_identifier) @symbol.name) @symbol.type
(enum_specifier
  name: (type_identifier) @symbol.name) @symbol.type
(declaration
  declarator: (identifier) @symbol.name) @symbol.var
(declaration
  declarator: (init_declarator
    declarator: (identifier) @symbol.name)) @symbol.var
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @symbol.name)) @symbol.func
