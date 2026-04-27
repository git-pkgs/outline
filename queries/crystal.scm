(comment) @keep

(require) @keep

(class_def
  (constant)) @signature
(module_def
  (constant)) @signature
(struct_def
  (constant)) @signature
(enum_def) @keep
(annotation_def) @keep

(method_def) @keep
(abstract_method_def) @keep

(const_assign) @keep
(alias) @keep
(type_declaration) @keep
(call
  method: (identifier) @_m
  (#match? @_m "^(getter|setter|property|include|extend)$")) @keep
