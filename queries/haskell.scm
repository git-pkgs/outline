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

(data_type
  .
  (name) @symbol.name) @symbol.type
(newtype
  .
  (name) @symbol.name) @symbol.type
(type_synomym
  .
  (name) @symbol.name) @symbol.type
(type_family
  .
  (name) @symbol.name) @symbol.type
(class
  .
  (name) @symbol.name) @symbol.type
(bind
  .
  (variable) @symbol.name) @symbol.var
(function
  .
  (variable) @symbol.name) @symbol.func
