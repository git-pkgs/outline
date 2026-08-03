(function_declaration
  name: (identifier) @name) @def.func

(method_declaration
  receiver: (parameter_list
    (parameter_declaration
      type: [
        (type_identifier) @container
        (pointer_type (type_identifier) @container)
        (generic_type type: (type_identifier) @container)
        (pointer_type (generic_type type: (type_identifier) @container))
      ]))
  name: (field_identifier) @name) @def.method

(type_spec
  name: (type_identifier) @name) @def.type

(const_spec
  name: (identifier) @name) @def.const

(var_spec
  name: (identifier) @name) @def.var

(call_expression
  function: (identifier) @name) @ref.call

(call_expression
  function: (selector_expression
    field: (field_identifier) @name)) @ref.call

(import_spec
  path: (interpreted_string_literal) @name) @ref.import

(type_identifier) @name @ref.type
