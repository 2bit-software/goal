# Research — US-032 Emit the Go subset from AST

This is an internal-codebase task (no external/web research required). Findings
come from reading the existing AST, parser, and backend packages.

## Reference model

The emitter is modeled on `go/printer`/`go/ast` but trimmed to goal's Go subset
and follows a deliberate "emit token-correct Go, format once" discipline (see
emit.go header): the emitter need not pretty-print, only produce parseable Go;
`backend.GoFormatter` (go/format.Source) normalizes layout afterward.

## What the parser actually produces (the subset to cover)

Enumerated from `internal/ast/ast.go` statement/expression node definitions and
the parser's `parseStmt`/`parseDecl`/`parseExpr` dispatch:

- Statements: Block, Expr, Assign, IncDec, Return, If, For, Range, **Switch**,
  **CaseClause**, Defer, Go, Branch, Decl, Empty (+ goal-only AssertStmt).
- Expressions/types: Ident, BasicLit, Paren, Unary, Binary, Selector, Star,
  Index, IndexList, Slice, Call, KeyValue, CompositeLit, FuncLit, Array, Map,
  Struct, Interface, Func, Chan, Ellipsis.

## Gap (confirmed)

The US-026 emitter handles every node above EXCEPT `*ast.SwitchStmt` and
`*ast.CaseClause`. A plain-Go file containing a `switch` therefore fails today
with `backend: unsupported statement *ast.SwitchStmt`. This is the only missing
ordinary-Go statement form.

Confirmed NOT in goal's subset (no AST node, no parser production): LabeledStmt,
SendStmt, SelectStmt/CommClause, TypeSwitchStmt, and trailing variadic call
spread `f(xs...)` (CallExpr has no Ellipsis field). These are out of scope.

## Approach (chosen)

1. Add SwitchStmt + CaseClause emission to emit.go, mirroring the existing
   `ifStmt`/`forStmt` structure (optional init `;`, optional tag, `{` body with
   `case <exprs>:` / `default:` clauses each holding a statement list).
2. Enrich a plain-Go behavioral fixture to exercise the full subset (switch,
   struct type + composite literal, map, slice, defer, multi-return) and gate it
   through `corpus.RunCompile(... corpus.TranspilerFunc(backend.Transpile))`,
   the established behavioral-tier seam.

**Confidence**: High — the gap is a single, well-isolated missing case in a
switch statement; the surrounding emitter patterns are directly reusable.
