(comment) @keep

(function_definition
  body: (compound_statement) @body) @signature
(function_definition) @keep

(program
  (variable_assignment) @keep)
(program
  (command
    name: (command_name
      (word) @_c)
    (#match? @_c "^(source|export|set|shopt|alias|declare|readonly|local)$")) @keep)
(program
  (declaration_command) @keep)

(function_definition
  name: (word) @symbol.name) @symbol.func
(program
  (variable_assignment
    name: (variable_name) @symbol.name) @symbol.var)
