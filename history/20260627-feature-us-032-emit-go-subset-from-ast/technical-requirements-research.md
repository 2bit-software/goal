# Technical Requirements & Research — US-032

## Current state (from US-026)

- `internal/backend/emit.go` is the seed emitter. It already covers:
  package/import/func/var/const/type decls; the statements Block, Expr, Assign,
  IncDec, Return, If, For, Range, Decl, Defer, Go, Branch, Empty; the expressions
  Ident, BasicLit, Paren, Unary, Binary, Selector, Star, Index, IndexList, Slice,
  Call, KeyValue, CompositeLit, FuncLit; and the type forms Array, Map, Struct,
  Interface, Func, Chan, Ellipsis.
- `internal/backend/backend.go` wires the Backend/Formatter seam and the
  `Transpile` engine (parse -> sema -> Emit -> Format), satisfying
  `corpus.Transpiler`.

## Gap analysis (full Go subset that the parser produces)

- `*ast.SwitchStmt` and `*ast.CaseClause` are produced by the parser (US-018) but
  the emitter's `stmt` switch does NOT handle them — emitting any plain-Go file
  containing a `switch` currently fails with `unsupported statement
  *ast.SwitchStmt`. This is the primary missing plain-Go form.
- No `LabeledStmt`/`SendStmt`/`SelectStmt`/`TypeSwitchStmt` nodes exist in the AST
  or parser, so they are out of scope (not part of goal's Go subset).

## Plan

1. Add `*ast.SwitchStmt` + `*ast.CaseClause` emission to emit.go (expression
   switch with optional init, optional tag, case/default clauses).
2. Enrich the behavioral fixture (or add one) to exercise the full Go subset
   including a switch, struct type + composite literal, map, defer, etc., and
   prove it compiles + vets via `corpus.RunCompile`.

## Verify gates

- `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
- AC2 behavioral tier: `corpus.RunCompile` over a plain-Go fixture through
  `corpus.TranspilerFunc(backend.Transpile)`.
