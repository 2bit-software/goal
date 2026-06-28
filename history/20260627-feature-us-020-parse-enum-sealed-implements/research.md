# Research — US-020

This story is self-contained within the repo; no external research was needed.

## Findings (from the codebase)

- `internal/ast/goal_decl.go` already defines every target node:
  `EnumDecl{Enum, Name, Lbrace, Variants, Rbrace}`,
  `Variant{Name, Lbrace, Payload, Rbrace}`, `PayloadField{Name, Type}`,
  `SealedInterfaceDecl{Sealed, Interface, Name, Methods}`,
  `ImplementsClause{Implements, Type}`. `StructType.Implements` already holds
  the clause, and `ast.Walk` already descends into all of these (US-015).
- `internal/token/token.go`: `enum`->`token.ENUM`, `interface`->`token.INTERFACE`
  are reserved keywords; `sealed`/`implements` are contextual keywords that lex
  as `token.IDENT` (verified by token tests).
- `internal/parser/parser.go` currently dispatches top-level decls in
  `parseDecl` on PACKAGE/IMPORT/CONST/VAR/TYPE/FUNC; goal-specific decls are not
  yet handled. `parseStructType` parses `struct { fields }` but ignores any
  `implements` clause. `parseInterfaceType`/`parseMethodSpec` parse a method
  list and can be reused for the sealed interface body.

## Surface syntax to support (from the example inputs)

- Enum: `enum Status { Pending; Active { since: Time }; Cancelled { reason: string, at: Time } }`
  (variants separated only by whitespace — no commas/semicolons between them).
- Sealed interface: `sealed interface Shape {}` (often empty body).
- Implements: `type Circle struct implements Shape { Radius float64 }` and the
  qualified form `type Discard struct implements io.Writer {}`.

## Confidence

High. The node set and tokens exist; this is purely parser wiring plus a test.
