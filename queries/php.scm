(comment) @keep
(php_tag) @keep

(namespace_definition) @keep
(namespace_use_declaration) @keep

(class_declaration
  body: (declaration_list) @body) @signature
(interface_declaration
  body: (declaration_list) @body) @signature
(trait_declaration
  body: (declaration_list) @body) @signature
(enum_declaration
  body: (enum_declaration_list) @body) @signature

(property_declaration) @keep
(const_declaration) @keep
(use_declaration) @keep

(method_declaration
  body: (compound_statement) @body) @signature
(method_declaration !body) @keep

(function_definition
  body: (compound_statement) @body) @signature
