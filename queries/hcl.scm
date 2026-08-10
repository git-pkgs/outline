(comment) @keep

(attribute
  (identifier)
  (expression
    (function_call))) @keep

(block
  (body) @body) @signature

(block
  .
  (identifier) @_kind
  .
  (string_lit
    (template_literal) @symbol.name)
  (#match? @_kind "^(variable|output|locals)$")) @symbol.var

(block
  .
  (identifier) @_kind
  .
  (string_lit)
  .
  (string_lit
    (template_literal) @symbol.name)
  (#match? @_kind "^(resource|data)$")) @symbol.type

(block
  .
  (identifier) @_kind
  .
  (string_lit
    (template_literal) @symbol.name)
  (#match? @_kind "^(module|provider)$")) @symbol.type
