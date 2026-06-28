# AI-Consumer Readiness Audit — US-004

## Findings

No CRITICAL or MAJOR findings. An implementer has everything needed:

- Input types are named and exist: `*ast.File` (internal/ast), `*sema.Info`
  (internal/sema), produced by `parser.ParseFile` + `sema.Resolve`.
- The entry point is unambiguous: top-level `*ast.FuncDecl` with `Name.Name ==
  "main"` and `Recv == nil`; body is `*ast.BlockStmt`.
- Acceptance criteria are directly assertable: run returns nil for empty main;
  run returns a descriptive error when no main exists.

### MINOR-1: Error message text not pinned
The spec requires a "descriptive, named error" but does not pin exact text. A
test can assert the error is non-nil and mentions "main" without coupling to
exact wording. Acceptable.

## Assumptions

- Statement evaluation is a no-op stub for an empty body; richer eval is US-005+.
  The Run signature returns `error` so later stories extend it without an API
  break.
- The interpreter package already provides Env/Value; this story adds only the
  entry (`interp.go`).
