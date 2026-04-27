(comment) @keep

(using_directive) @keep
(file_scoped_namespace_declaration) @keep

(namespace_declaration
  body: (declaration_list) @body) @signature

(class_declaration
  body: (declaration_list) @body) @signature
(struct_declaration
  body: (declaration_list) @body) @signature
(record_declaration
  body: (declaration_list) @body) @signature
(record_declaration !body) @keep
(interface_declaration
  body: (declaration_list) @body) @signature
(enum_declaration) @keep

(field_declaration) @keep
(property_declaration) @keep
(event_declaration) @keep
(event_field_declaration) @keep
(delegate_declaration) @keep

(method_declaration
  body: (block) @body) @signature
(method_declaration !body) @keep
(constructor_declaration
  body: (block) @body) @signature
(operator_declaration
  body: (block) @body) @signature
(local_function_statement
  body: (block) @body) @signature
