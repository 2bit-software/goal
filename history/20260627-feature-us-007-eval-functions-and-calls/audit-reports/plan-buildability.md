# Plan Audit — Buildability

## Findings

No CRITICAL or MAJOR findings. The dependency order (value -> registration ->
evalCallMulti -> stmt forms/Run -> tests) is correct: each layer only uses what
is built before it. All referenced AST nodes exist and were verified against
internal/ast/ast.go: `FuncDecl{Recv,Name,Type,Body}`, `FuncType{Params,Results}`,
`FieldList.List[].Names`, `CallExpr{Fun,Args}`, `ReturnStmt{Results}`,
`IfStmt{Init,Cond,Body,Else}`. The Env API (NewChild/Define/Lookup/Assign) and
`applyBinary`/`zeroValue` reuse points all exist.

- MINOR: `value.go` will import `goal/internal/ast` for the first time; that adds
  no new external dependency and keeps the US-022 allowed-import set
  (errors/fmt/strconv + ast/sema/token).
- MINOR: Parameter `Field`s may group names (`a, b int`); the binder must iterate
  `Field.Names` and flatten across fields to get the positional parameter list.

## Assumptions
- A multi-value call is only legal as the sole RHS of a multi-assign or a return;
  elsewhere a call yields exactly one value. This mirrors Go semantics and is the
  spec's stated contract.
- The defining scope for a top-level function is the interpreter root (no nested
  function literals), so the closure env is always `root`.
