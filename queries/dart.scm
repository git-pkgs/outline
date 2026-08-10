(comment) @keep
(documentation_comment) @keep

(import_or_export) @keep
(part_directive) @keep
(part_of_directive) @keep
(library_name) @keep

(class_definition
  body: (class_body) @body) @signature
(mixin_declaration
  (class_body) @body) @signature
(extension_declaration
  (extension_body) @body) @signature
(enum_declaration) @keep

(declaration) @keep
(method_signature) @keep
(function_signature) @keep
(getter_signature) @keep
(setter_signature) @keep
(type_alias) @keep

(class_definition
  name: (identifier) @symbol.name) @symbol.class
(mixin_declaration
  .
  (mixin)
  .
  (identifier) @symbol.name) @symbol.type
(extension_declaration
  name: (identifier) @symbol.name) @symbol.type
(enum_declaration
  name: (identifier) @symbol.name) @symbol.type
(type_alias
  .
  (type_identifier) @symbol.name) @symbol.type
(function_signature
  name: (identifier) @symbol.name) @symbol.func
