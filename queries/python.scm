(comment) @keep

(import_statement) @keep
(import_from_statement) @keep
(future_import_statement) @keep

(decorator) @keep

(class_definition
  body: (block) @body) @signature

(function_definition
  body: (block) @body) @signature

(module (assignment) @keep)
(module (string) @keep)

(class_definition
  body: (block
    (assignment) @keep))
(class_definition
  body: (block
    (string) @keep))
