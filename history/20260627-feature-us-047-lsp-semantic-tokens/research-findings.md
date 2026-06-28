# Research Findings — US-047 LSP semantic tokens

## LSP `textDocument/semanticTokens/full` (protocol 3.16+)

**Summary**: The server declares a *legend* up front — an ordered list of
`tokenTypes` (e.g. `keyword`, `enum`, `enumMember`, `function`, `type`,
`parameter`, `property`, `variable`, `string`, `number`, `comment`, `operator`,
`interface`, `struct`, `method`) and `tokenModifiers`. Each token is then
referenced by its *index* into those lists.

The `/full` response is `{ "data": [uint, ...] }`: a flat array of 5-tuples,
one per token, **delta-encoded relative to the previous token**:

```
[ deltaLine, deltaStartChar, length, tokenType, tokenModifiers ]
```

- `deltaLine`: line delta from the previous token (0-based lines).
- `deltaStartChar`: char delta from the previous token's start *if on the same
  line*, else the absolute 0-based char.
- `length`: token length in characters.
- `tokenType`: index into the legend's `tokenTypes`.
- `tokenModifiers`: bitset over the legend's `tokenModifiers` (0 = none).

Tokens MUST be emitted in document order (sorted by start position).

**Capability**: advertise under `ServerCapabilities.semanticTokensProvider`:
```json
{ "legend": { "tokenTypes": [...], "tokenModifiers": [...] }, "full": true }
```

Method routed: `textDocument/semanticTokens/full`. Confidence: High (this is the
stable, widely-implemented shape used by gopls and rust-analyzer).

## Fit for goal
- `internal/lexer.Tokens` already yields ordered `token.Token{Kind,Lit,Pos}`
  with 0-based byte `Offset` and 1-based `Line`/`Col`. Source is ASCII in the
  corpus, so byte length == character length.
- The AST (US-046 already parses for symbols) provides the role refinement that
  makes classification "from the AST": enum/variant/func/type/param/field
  identities.

## Open questions / decisions
- Emit only confidently-classified tokens (skip unknown identifiers) so builtin
  types like `int` are not miscolored. Decided: yes.
- Range requests (`/range`) and delta (`/full/delta`) are optional and out of
  scope for this story; advertise `full: true` only.
