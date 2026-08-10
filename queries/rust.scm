(line_comment) @keep
(block_comment) @keep

(use_declaration) @keep
(extern_crate_declaration) @keep
(attribute_item) @keep
(inner_attribute_item) @keep

(mod_item
  body: (declaration_list) @body) @signature
(mod_item !body) @keep

(struct_item) @keep
(enum_item) @keep
(union_item) @keep
(type_item) @keep
(const_item) @keep
(static_item) @keep
(macro_definition
  (macro_rule) @body) @signature

(trait_item
  body: (declaration_list) @body) @signature

(impl_item
  body: (declaration_list) @body) @signature

(function_item
  body: (block) @body) @signature
(function_item !body) @keep
(function_signature_item) @keep

(mod_item
  name: (identifier) @symbol.name) @symbol.type
(struct_item
  name: (type_identifier) @symbol.name) @symbol.type
(enum_item
  name: (type_identifier) @symbol.name) @symbol.type
(union_item
  name: (type_identifier) @symbol.name) @symbol.type
(type_item
  name: (type_identifier) @symbol.name) @symbol.type
(trait_item
  name: (type_identifier) @symbol.name) @symbol.type
(const_item
  name: (identifier) @symbol.name) @symbol.const
(static_item
  name: (identifier) @symbol.name) @symbol.var
(function_item
  name: (identifier) @symbol.name) @symbol.func
(function_signature_item
  name: (identifier) @symbol.name) @symbol.func
