# Plan Buildability Audit

- Dependency order valid: ast field -> walk/parser/backend -> tests.
- Interface contract concrete (FuncType struct shown with new field).
- File paths verified against codebase (all four files exist and lines read).
- Integration point specific: emit type params at start of funcSig, between
  printed name and params.

No CRITICAL/MAJOR findings.

## Assumptions

- `fieldList(tp, "[", "]")` produces a valid Go type-param list, mirroring the
  existing TypeSpec emission at emit.go:347-348.
- Generic methods excluded by gating on `fd.Recv == nil`.
