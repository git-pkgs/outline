(comment) @keep

(open_module) @keep
(include_module) @keep
(module_definition
  body: (_) @body) @signature
(module_definition) @keep
(module_type_definition) @keep

(type_definition) @keep
(exception_definition) @keep
(external) @keep

(value_definition) @keep
(class_definition) @keep

(module_definition
  (module_binding
    (module_name) @symbol.name)) @symbol.type
(module_type_definition
  .
  (module_type_name) @symbol.name) @symbol.type
(type_definition
  (type_binding
    (type_constructor) @symbol.name)) @symbol.type
(external
  .
  (value_name) @symbol.name) @symbol.func
(value_definition
  (let_binding
    (value_name) @symbol.name
    (parameter))) @symbol.func
(value_definition
  (let_binding
    (value_name) @symbol.name)) @symbol.var
(class_definition
  (class_binding
    (class_name) @symbol.name)) @symbol.class
