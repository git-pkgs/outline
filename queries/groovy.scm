(comment) @keep
(groovy_doc) @keep

(groovy_package) @keep
(groovy_import) @keep

(class_definition
  (closure) @body) @signature

(declaration) @keep

(function_definition
  (closure) @body) @signature
(function_declaration) @keep

(class_definition
  name: (identifier) @symbol.name) @symbol.class
(function_definition
  function: (identifier) @symbol.name) @symbol.func
(function_declaration
  function: (identifier) @symbol.name) @symbol.func
