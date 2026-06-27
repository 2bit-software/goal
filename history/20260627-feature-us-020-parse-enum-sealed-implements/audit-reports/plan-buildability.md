# Plan Buildability Audit — US-020

- Dependency order is valid: AST/token already exist; new parse methods depend
  only on existing helpers; the `parser.go` dispatch edit depends on the new
  method names; tests come last. No forward references.
- Interface contracts use real `ast` node types and real `*parser` helper methods
  (`ident`, `expect`, `parseType`) verified present in `parser.go`.
- File paths are real (`internal/parser/`), no placeholders.
- Integration points are specific (exact functions `parseDecl`, `parseStructType`
  and the exact insertion points).

One watch-item (MINOR): `parseStructType` is also reachable for an anonymous
inline `struct{...}` type; `parseImplementsClause` must be a no-op (return nil,
consume nothing) when `implements` is absent so inline struct types are
unaffected. The plan already specifies this. No CRITICAL/MAJOR.

## Assumptions

- The contextual-keyword check is `Kind == IDENT && Lit == "sealed"/"implements"`.
- `parseInterfaceType`-style method-list parsing is reused for the sealed
  interface body.
