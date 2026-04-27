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
