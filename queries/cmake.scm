(line_comment) @keep
(bracket_comment) @keep

(function_def
  (function_command) @keep
  (body) @body) @signature

(macro_def
  (macro_command) @keep
  (body) @body) @signature

(source_file
  (normal_command) @keep)
(source_file
  (if_condition) @signature)
(source_file
  (foreach_loop) @signature)

(function_def
  (function_command
    (argument_list
      .
      (argument) @symbol.name))) @symbol.func
(macro_def
  (macro_command
    (argument_list
      .
      (argument) @symbol.name))) @symbol.func
