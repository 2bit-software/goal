# Technical Requirements / Research — US-020

## Existing scaffolding (already in place)

- AST nodes exist (US-015): `ast.EnumDecl`, `ast.Variant`, `ast.PayloadField`,
  `ast.SealedInterfaceDecl`, `ast.ImplementsClause`; `StructType.Implements`
  holds the implements clause.
- Token kinds exist (US-011): `token.ENUM`, `token.INTERFACE` are reserved
  keywords; `sealed`, `implements` lex as `token.IDENT` (contextual keywords).

## Approach

- `parseDecl` dispatches: `token.ENUM` -> `parseEnumDecl`; an `IDENT` whose Lit
  is `"sealed"` -> `parseSealedInterfaceDecl`.
- `parseEnumDecl`: after `enum Name {`, loop over variants until `}`. Each
  variant is an IDENT tag; an optional `{ name: Type, ... }` payload block makes
  it a payload variant (`ast.PayloadField` list).
- `parseSealedInterfaceDecl`: `sealed interface Name { methods }`, reusing the
  existing interface method-list parsing.
- `parseStructType`: after `struct`, if the current token is the contextual
  `implements` identifier, parse `implements <TypeName>` into
  `StructType.Implements` before the field-list brace.

## Verify

- prd.json verifyCommands: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`.
- New parser test over the 01-enums and 07-implements example `.goal` inputs.

No other technical requirements were specified by the user.
