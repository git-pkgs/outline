(line_comment) @keep
(block_comment) @keep

(package_declaration) @keep
(import_declaration) @keep

(class_declaration
  body: (class_body) @body) @signature

(interface_declaration
  body: (interface_body) @body) @signature

(enum_declaration
  body: (enum_body) @body) @signature

(record_declaration
  body: (class_body) @body) @signature

(annotation_type_declaration) @keep

(field_declaration) @keep
(constant_declaration) @keep
(enum_constant) @keep

(method_declaration
  body: (block) @body) @signature
(method_declaration !body) @keep

(constructor_declaration
  body: (constructor_body) @body) @signature
