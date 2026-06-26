# Technical Spec — tree-sitter-goal (M1)

## Node inventory (named nodes the tree exposes)

- `source_file`, `package_clause`, `import_declaration` / `import_spec`
- `function_declaration` (receiver?, type_parameters?, parameters, result?), `method` via receiver
- `type_declaration` → `struct_type` (with optional `implements_clause`), `interface_type`, `type_alias`
- `enum_declaration` → `enum_variant` (name, optional `variant_payload` of `field: Type`)
- `from_declaration` (`from func ...`), `derive_declaration` (`derive func ...`)
- `const_declaration`, `var_declaration`
- Types: `pointer_type`, `slice_type`, `array_type`, `map_type`, `qualified_type`,
  `generic_type` (`Name[T, ...]`), `type_identifier`; `Result`/`Option` are generic_type
  whose name is captured specially in queries
- Statements: `block`, `return_statement`, `if_statement`, `for_statement`, `go_statement`,
  `defer_statement`, `short_var_declaration` (`:=`), `assignment_statement`,
  `expression_statement`, `assert_statement`, `match_statement`, `unwrap_binding` (`x := e?`)
- `match_expression` → `match_arm` (`pattern => body`), pattern = expression | `_`
- Expressions: `identifier`, `selector_expression`, `call_expression`, `index_expression`,
  `composite_literal` with `keyed_element` (`field: expr`) and `spread_element`
  (`...defaults` | `...derive(expr)`), `unary_expression`, `binary_expression`,
  `unwrap_expression` (postfix `?`), `parenthesized_expression`
- Literals: `interpreted_string_literal`, `raw_string_literal`, `rune_literal`,
  `int_literal`, `float_literal`, `imaginary_literal`, `true`/`false`/`nil`/`iota`
- Comments: `comment` (`//`, `/* */`), `doc_comment` (`///` line) containing `doctest_marker` (`>>>`)

## externals (scanner.c)

`externals: [$._automatic_semicolon]` — emitted at a newline, before `}`, or at EOF when the
previous significant token can terminate a statement (identifier, literal, `)`, `]`, `}`,
`?`, `return`-less expression end, etc.), per Go's ASI rules. Statement rules end with an
optional `_automatic_semicolon`. `extras`: `/\s/` (incl. spaces/tabs; newlines consumed by
scanner) + `comment` + `doc_comment`.

## highlights.scm capture map

| Construct | Capture |
|---|---|
| `enum sealed match assert from derive implements` + Go keywords | `@keyword` |
| `Result` `Option` | `@type.builtin` |
| builtin types (`int string bool ...`) | `@type.builtin` |
| type identifiers | `@type` |
| `Ok Err Some None` and `Enum.Variant` member | `@constant` / `@variable.member` |
| `true false nil iota` | `@constant.builtin` |
| function name in decl/call | `@function` / `@function.call` |
| `=> ? ... + - * / ...` | `@operator` |
| `// /* */` | `@comment` ; `///` | `@comment.documentation` ; `>>>` | `@keyword` |
| strings / runes | `@string` ; numbers | `@number` |

## package.json

`{ "name": "tree-sitter-goal", "scripts": { "generate": "tree-sitter generate",
"test": "tree-sitter test" }, "tree-sitter": [{ "scope": "source.goal",
"file-types": ["goal"], "highlights": "queries/highlights.scm" }] }` (devDep on
`tree-sitter-cli` optional; brew CLI is used here).
