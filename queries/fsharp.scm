(line_comment) @keep
(block_comment) @keep

(named_module) @keep
(module_defn) @keep
(import_decl) @keep

(type_definition) @keep

(declaration_expression) @keep

(named_module
  .
  (long_identifier
    (identifier) @symbol.name)) @symbol.type
(module_defn
  .
  (identifier) @symbol.name) @symbol.type
(type_definition
  .
  (_
    (type_name
      (identifier) @symbol.name))) @symbol.type
(declaration_expression
  (function_or_value_defn
    (value_declaration_left
      (identifier_pattern
        (long_identifier_or_op
          (identifier) @symbol.name))))) @symbol.var
