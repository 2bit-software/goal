# Technical Requirements / Research — US-008

## Existing seam

- `internal/interp/interp.go` `execStmt` is the statement-dispatch switch; it
  already handles ExprStmt, DeclStmt, AssignStmt, ReturnStmt, IfStmt, EmptyStmt.
- Non-local control flow already uses an error-sentinel pattern: `returnSignal`
  is an `error` threaded up through execBlock/execStmt and recovered at the call
  boundary (callFunc). break/continue follow the SAME pattern with their own
  sentinel signal types, caught at the loop/switch boundary.
- `execIf` already opens child scopes via `scope.NewChild()`; for/switch bodies
  must do likewise for nested block scoping.

## AST nodes (internal/ast/ast.go)

- `ForStmt{Init, Cond, Post, Body}` — all of Init/Cond/Post optional.
- `SwitchStmt{Init, Tag, Body}`; `CaseClause{List, Body}` (List nil => default).
- `BranchStmt{Tok}` — token.BREAK / token.CONTINUE (goto/fallthrough out of scope).
- `BlockStmt` — a bare nested block; runs its List in a child scope.
- `IncDecStmt{X, Tok}` — `i++` / `i--`, needed for for-loop post clauses.

## Scope notes

- range-for (RangeStmt) is US-009 (composite types) — NOT this story.
- goto/fallthrough/labelled break are out of scope; remain descriptive refusals.
