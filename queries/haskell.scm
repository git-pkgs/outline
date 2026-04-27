(comment) @keep
(haddock) @keep

(header) @keep
(import) @keep

(data_type) @keep
(newtype) @keep
(type_synomym) @keep
(type_family) @keep
(class
  (class_declarations) @body) @signature
(class) @keep
(instance
  (instance_declarations) @body) @signature
(instance) @keep

(signature) @keep

(bind
  (match) @body) @signature
(function
  (match) @body) @signature
