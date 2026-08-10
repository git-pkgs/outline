(line_comment) @keep
(multiline_comment) @keep

(package_header) @keep
(import_header) @keep

(class_declaration
  (class_body) @body) @signature
(class_declaration
  (enum_class_body) @body) @signature

(object_declaration
  (class_body) @body) @signature

(property_declaration) @keep
(type_alias) @keep
(companion_object
  (class_body) @body) @signature

(function_declaration
  (function_body) @body) @signature
(function_declaration) @keep

(secondary_constructor) @keep

(class_declaration
  .
  (type_identifier) @symbol.name) @symbol.class
(object_declaration
  .
  (type_identifier) @symbol.name) @symbol.type
(type_alias
  .
  (type_identifier) @symbol.name) @symbol.type
(property_declaration
  (variable_declaration
    (simple_identifier) @symbol.name)) @symbol.var
(function_declaration
  .
  (simple_identifier) @symbol.name) @symbol.func
