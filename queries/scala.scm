(comment) @keep
(block_comment) @keep

(package_clause) @keep
(import_declaration) @keep

(class_definition
  body: (template_body) @body) @signature
(class_definition !body) @keep
(trait_definition
  body: (template_body) @body) @signature
(trait_definition !body) @keep
(object_definition
  body: (template_body) @body) @signature
(object_definition !body) @keep
(enum_definition
  body: (enum_body) @body) @signature

(function_definition
  body: (block) @body) @signature
(function_definition) @keep
(function_declaration) @keep

(val_definition) @keep
(var_definition) @keep
(type_definition) @keep
(given_definition) @keep

(class_definition
  name: (identifier) @symbol.name) @symbol.class
(trait_definition
  name: (identifier) @symbol.name) @symbol.type
(object_definition
  name: (identifier) @symbol.name) @symbol.type
(enum_definition
  name: (identifier) @symbol.name) @symbol.type
(function_definition
  name: (identifier) @symbol.name) @symbol.func
(function_declaration
  name: (identifier) @symbol.name) @symbol.func
(val_definition
  pattern: (identifier) @symbol.name) @symbol.const
(var_definition
  pattern: (identifier) @symbol.name) @symbol.var
(type_definition
  name: (type_identifier) @symbol.name) @symbol.type
