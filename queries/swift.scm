(comment) @keep
(multiline_comment) @keep

(import_declaration) @keep

(class_declaration
  body: (class_body) @body) @signature
(class_declaration
  body: (enum_class_body) @body) @signature

(protocol_declaration
  body: (protocol_body) @body) @signature

(property_declaration) @keep
(typealias_declaration) @keep
(protocol_function_declaration) @keep
(protocol_property_declaration) @keep
(associatedtype_declaration) @keep

(function_declaration
  body: (function_body) @body) @signature
(function_declaration !body) @keep

(init_declaration
  body: (function_body) @body) @signature
(deinit_declaration
  body: (function_body) @body) @signature
