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
