# Plan Buildability Audit — US-004

## Findings

No CRITICAL or MAJOR findings. The plan is buildable in dependency order:

- Dependency order is valid: front-end (ast/parser/sema) and interp value/env
  already exist; interp.go depends only on them; the test depends on interp.go +
  parser + sema. No forward references.
- Interface contracts are concrete actual Go signatures.
- File paths are real and verified: `internal/interp/` exists with value.go +
  env.go; interp.go and interp_test.go are new and do not collide.
- Integration point is specific: `Run` reads `file.Decls`, matches
  `*ast.FuncDecl` with `Recv == nil` && `Name.Name == "main"`.

### MINOR-1: Body walk is a stub
`Run` walks an empty body as a no-op. A non-empty body is out of scope (US-005+);
the plan should not over-build statement evaluation here. Noted, not blocking.

## Assumptions

- `sema.Resolve` (not `sema.New`) is used in the test to exercise the real
  front-end.
- The `Run` signature returns `error` so US-005+ extend it without an API break.
