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

(namespace_definition
  name: (namespace_identifier) @symbol.name) @symbol.type
(alias_declaration
  name: (type_identifier) @symbol.name) @symbol.type
(type_definition
  (type_identifier) @symbol.name) @symbol.type
(class_specifier
  name: (type_identifier) @symbol.name) @symbol.class
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
