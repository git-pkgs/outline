(comment) @keep

(list_lit
  .
  (sym_lit) @_head
  (#match? @_head "^(ns|in-ns)$")) @keep

(list_lit
  .
  (sym_lit) @_head
  (#match? @_head "^(def|defonce|declare)$")) @keep

(list_lit
  .
  (sym_lit) @_head
  .
  (sym_lit)
  .
  (vec_lit)
  .
  (_) @body
  (#match? @_head "^(defn|defn-|defmacro|defmethod)$")) @signature

(list_lit
  .
  (sym_lit) @_head
  (#match? @_head "^(defn|defn-|defmacro|defmulti|defmethod|defprotocol|defrecord|deftype|definterface)$")) @keep

(list_lit
  .
  (sym_lit) @_head
  .
  (sym_lit) @symbol.name
  (#match? @_head "^(def|defonce|declare)$")) @symbol.var

(list_lit
  .
  (sym_lit) @_head
  .
  (sym_lit) @symbol.name
  (#match? @_head "^(defn|defn-|defmacro|defmulti|defmethod)$")) @symbol.func

(list_lit
  .
  (sym_lit) @_head
  .
  (sym_lit) @symbol.name
  (#match? @_head "^(defprotocol|defrecord|deftype|definterface)$")) @symbol.type
