(comment) @keep
(documentation_comment) @keep

(import_statement) @keep
(import_from_statement) @keep
(include_statement) @keep
(export_statement) @keep

(type_section) @keep
(const_section) @keep
(var_section) @keep
(let_section) @keep

(proc_declaration
  body: (statement_list) @body) @signature
(proc_declaration) @keep
(func_declaration
  body: (statement_list) @body) @signature
(func_declaration) @keep
(method_declaration
  body: (statement_list) @body) @signature
(method_declaration) @keep
(iterator_declaration
  body: (statement_list) @body) @signature
(template_declaration
  body: (statement_list) @body) @signature
(macro_declaration
  body: (statement_list) @body) @signature

(type_section
  (type_declaration
    (type_symbol_declaration
      (identifier) @symbol.name))) @symbol.type
(type_section
  (type_declaration
    (type_symbol_declaration
      (exported_symbol
        (identifier) @symbol.name) @symbol.exported))) @symbol.type

(const_section
  (variable_declaration
    (symbol_declaration_list
      (symbol_declaration
        (identifier) @symbol.name)))) @symbol.const
(const_section
  (variable_declaration
    (symbol_declaration_list
      (symbol_declaration
        (exported_symbol
          (identifier) @symbol.name) @symbol.exported)))) @symbol.const
(var_section
  (variable_declaration
    (symbol_declaration_list
      (symbol_declaration
        (identifier) @symbol.name)))) @symbol.var
(var_section
  (variable_declaration
    (symbol_declaration_list
      (symbol_declaration
        (exported_symbol
          (identifier) @symbol.name) @symbol.exported)))) @symbol.var
(let_section
  (variable_declaration
    (symbol_declaration_list
      (symbol_declaration
        (identifier) @symbol.name)))) @symbol.const
(let_section
  (variable_declaration
    (symbol_declaration_list
      (symbol_declaration
        (exported_symbol
          (identifier) @symbol.name) @symbol.exported)))) @symbol.const

(proc_declaration
  name: (identifier) @symbol.name) @symbol.func
(proc_declaration
  name: (exported_symbol
    (identifier) @symbol.name) @symbol.exported) @symbol.func
(func_declaration
  name: (identifier) @symbol.name) @symbol.func
(func_declaration
  name: (exported_symbol
    (identifier) @symbol.name) @symbol.exported) @symbol.func
(method_declaration
  name: (identifier) @symbol.name) @symbol.func
(method_declaration
  name: (exported_symbol
    (identifier) @symbol.name) @symbol.exported) @symbol.func
(iterator_declaration
  name: (identifier) @symbol.name) @symbol.func
(iterator_declaration
  name: (exported_symbol
    (identifier) @symbol.name) @symbol.exported) @symbol.func
(template_declaration
  name: (identifier) @symbol.name) @symbol.func
(template_declaration
  name: (exported_symbol
    (identifier) @symbol.name) @symbol.exported) @symbol.func
(macro_declaration
  name: (identifier) @symbol.name) @symbol.func
(macro_declaration
  name: (exported_symbol
    (identifier) @symbol.name) @symbol.exported) @symbol.func
