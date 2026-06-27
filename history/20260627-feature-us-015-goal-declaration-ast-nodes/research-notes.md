# Research Notes — US-015

## Summary

No external research required. The design is fully constrained by in-repo
sources of truth:

- **internal/ast (US-014)** establishes the node conventions to mirror: the
  `Node` interface (`Pos()`/`End()`), closed category markers via unexported
  methods (`declNode`/`stmtNode`/`exprNode`/`specNode`), support nodes that
  implement only `Node` (e.g. `Field`, `FieldList`), and `Walk(Visitor, Node)`
  pre-order traversal whose per-type switch is explicitly designed to grow with
  new goal nodes.
- **REWRITE-ARCHITECTURE.md §1.3** enumerates exactly the goal declaration
  surface to add: `FuncDecl` with `from`/`derive` modifiers,
  `EnumDecl`/`Variant`/`PayloadField`, struct type with `ImplementsClause`, and
  `SealedInterfaceDecl`. It also fixes the contextual-keyword decision
  (implements/sealed/from/derive lexed as identifiers, decided positionally) —
  so `from`/`derive` are a modifier enum on `FuncDecl`, never token kinds.
- **Feature examples** confirm the concrete shapes:
  - `enum Decision { Admit; Reject { reason: Rejection } }` → data-less and
    payload variants (`PayloadField` = `name: Type`).
  - `type Circle struct implements Shape { … }` and
    `type Discard struct implements io.Writer {}` → ImplementsClause type is a
    named type (`*Ident` or `*SelectorExpr`).
  - `sealed interface Shape {}` → SealedInterfaceDecl with a (possibly empty)
    method set.
  - `from func parseUUID(...)` and bodyless `derive func fromStorage(...)` →
    FuncMod on FuncDecl, body optional.

## Confidence

High — the work is additive, mechanical, and mirrors an existing in-package
pattern with a direct test analogue (`ast_test.go`'s Walk collector).

## Open Questions

None blocking. ImplementsClause is hosted on `StructType` (optional
`Implements` field) rather than introducing a separate `StructDecl`, keeping the
existing GenDecl/TypeSpec type model intact; a dedicated StructDecl can come with
the parser stories if needed.
