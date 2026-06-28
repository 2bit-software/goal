# Technical Requirements / Research — US-047

## Approach
Mirror the existing `internal/lsp/symbols.go` (US-046) pattern: parse the buffer
with `parser.ParseFile` and classify from the AST. Combine:
- `lexer.Tokens(src)` for token positions + lengths (the lexer already carries
  first-class `token.Pos`), and base classification of keywords / literals /
  comments / the `?` operator.
- An AST walk (`ast.Walk`) to build a byte-offset -> semantic-role override map
  for identifiers whose role is structurally known (enum name, enum variant,
  sealed interface / struct / alias names, func/method names, parameters,
  struct fields, variant construction/pattern enum + tag references).

## LSP protocol
- Advertise `semanticTokensProvider` with a legend (tokenTypes, tokenModifiers)
  and `full: true` in `ServerCapabilities`.
- Handle `textDocument/semanticTokens/full`, replying with
  `{ data: []uint }` delta-encoded as 5-tuples
  `[deltaLine, deltaStartChar, length, tokenType, tokenModifiers]`, 0-based.

## Notes
- Operators/delimiters carry empty `Lit`; length comes from `kind.String()` for
  those, `len(Lit)` otherwise.
- Only emit tokens we can classify confidently (keywords, comments, strings,
  numbers, `?`/`=>`/`...`, and AST-known identifiers) — skip unknown idents so a
  builtin like `int` is not miscolored.
