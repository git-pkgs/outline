(comment) @keep

(package_clause) @keep

(import_declaration) @keep

(type_declaration) @keep

(var_declaration) @keep

(const_declaration) @keep

(function_declaration
  body: (block) @body) @signature

(function_declaration
  !body) @keep

(method_declaration
  body: (block) @body) @signature

(method_declaration
  !body) @keep

(type_spec
  name: (type_identifier) @symbol.name) @symbol.type
(type_alias
  name: (type_identifier) @symbol.name) @symbol.type

(var_spec
  (identifier) @symbol.name) @symbol.var
(const_spec
  (identifier) @symbol.name) @symbol.const

(function_declaration
  name: (identifier) @symbol.name) @symbol.func
(method_declaration
  name: (field_identifier) @symbol.name) @symbol.func
