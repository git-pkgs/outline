(comment) @keep

(variable_assignment) @keep
(include_directive) @keep
(define_directive) @keep
(conditional) @signature

(rule
  (recipe) @body) @signature
(rule) @keep

(variable_assignment
  name: (word) @symbol.name) @symbol.var
(rule
  (targets
    (word) @symbol.name)) @symbol.func
