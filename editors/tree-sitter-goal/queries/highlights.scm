; Highlight queries for the Goal language (.goal).
; Standard capture names so the grammar renders under nvim-treesitter, Helix,
; Zed, and GitHub. Later patterns win, so specific cases follow general ones.

; ---- comments ----
(comment) @comment
(doc_comment) @comment.documentation

; ---- literals ----
(int_literal) @number
(float_literal) @number
(imaginary_literal) @number
(interpreted_string_literal) @string
(raw_string_literal) @string
(rune_literal) @string
(escape_sequence) @string.escape
(true) @constant.builtin
(false) @constant.builtin
(nil) @constant.builtin
(iota) @constant.builtin

; ---- keywords ----
[
  "package"
  "import"
  "func"
  "type"
  "var"
  "const"
  "struct"
  "interface"
  "map"
  "chan"
  "return"
  "if"
  "else"
  "for"
  "range"
  "switch"
  "case"
  "default"
  "go"
  "defer"
] @keyword

(fallthrough_statement) @keyword

; goal-specific keywords
[
  "enum"
  "sealed"
  "match"
  "assert"
  "from"
  "derive"
  "implements"
] @keyword

; ---- operators ----
[
  "=>"
  "?"
  "..."
  "="
  ":="
  "=="
  "!="
  "<"
  "<="
  ">"
  ">="
  "+"
  "-"
  "*"
  "/"
  "%"
  "&&"
  "||"
  "!"
  "&"
  "|"
  "^"
  "<<"
  ">>"
  "<-"
  "++"
  "--"
] @operator

; ---- types ----
((type_identifier) @type.builtin
  (#any-of? @type.builtin
    "bool" "byte" "complex64" "complex128" "error" "float32" "float64"
    "int" "int8" "int16" "int32" "int64" "rune" "string"
    "uint" "uint8" "uint16" "uint32" "uint64" "uintptr" "any" "comparable"))

; goal sum types
((type_identifier) @type.builtin
  (#any-of? @type.builtin "Result" "Option"))
((generic_type name: (type_identifier) @type.builtin)
  (#any-of? @type.builtin "Result" "Option"))

(type_identifier) @type
(qualified_type package: (identifier) @namespace)

; ---- declarations ----
(enum_declaration name: (identifier) @type)
(enum_variant name: (identifier) @constant)
(sealed_interface_declaration name: (identifier) @type)
(payload_field name: (identifier) @property)

(function_declaration name: (identifier) @function)
(method_declaration name: (identifier) @function.method)
(parameter name: (identifier) @variable.parameter)

; ---- expressions ----
(call_expression
  function: (identifier) @function.call)
(call_expression
  function: (selector_expression field: (identifier) @function.method.call))

(keyed_element key: (identifier) @property)
(labeled_argument label: (identifier) @property)

; Qualified construction `Enum.Variant` / `Result.Ok` — capitalized selector.
; Placed last so it wins over the method-call rule for constructor calls.
((selector_expression
   operand: (identifier) @type
   field: (identifier) @constant)
 (#match? @type "^[A-Z]")
 (#match? @constant "^[A-Z]"))

(identifier) @variable
