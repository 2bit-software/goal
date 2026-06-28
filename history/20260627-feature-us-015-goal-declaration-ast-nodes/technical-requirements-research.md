# Technical Requirements & Research — US-015

## Source of truth

- REWRITE-ARCHITECTURE.md §1.3: the ast must cover goal decls including
  `FuncDecl` with `from`/`derive` modifiers, `EnumDecl`/`Variant`/`PayloadField`,
  `StructDecl`/struct type with `ImplementsClause`, and `SealedInterfaceDecl`.
- Existing internal/ast skeleton (US-014): `Node` interface (Pos/End), closed
  category markers (declNode/stmtNode/exprNode/specNode), `Walk(Visitor, Node)`
  pre-order traversal with a per-type switch. New nodes ADD switch cases.

## Design decisions

- `EnumDecl` and `SealedInterfaceDecl` are `Decl` (carry `declNode()`), so they
  sit directly in `File.Decls`.
- `Variant` and `PayloadField` are support nodes (implement `Node` only, like
  the existing `Field`/`FieldList`).
- `ImplementsClause` is a support node attached to `StructType` via an optional
  `Implements *ImplementsClause` field (goal's `type T struct implements I {…}`).
  This keeps the existing GenDecl/TypeSpec/StructType type model intact rather
  than introducing a separate StructDecl in this story.
- `from`/`derive` are contextual keywords (lexed as IDENT), so they are recorded
  on `FuncDecl` as a small `FuncMod` enum (`FuncPlain`/`FuncFrom`/`FuncDerive`)
  plus `ModPos`, not as token kinds. When set, `FuncDecl.Pos()` returns `ModPos`.

## Verify

- `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
- New test `TestWalkGoalDeclChildren` in internal/ast asserts Walk descends into
  each new node's children.

No external technical requirements beyond the above were provided.
