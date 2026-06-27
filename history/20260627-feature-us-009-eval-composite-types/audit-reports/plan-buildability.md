# Plan Buildability Audit — US-009

Dependency order is a valid topological sort: value carriers (exist) → eval
cases → range/assignment (read eval) → tests. Interface contracts use concrete
Go signatures consistent with existing `(Value, error)` / `error` conventions.
File paths verified against the live tree (eval.go, interp.go, value.go all
exist; composite_test.go is new and non-conflicting). Integration points name
the exact switch statements (`evalExpr`, `execStmt`, `bindTargets`).

Each component compiles in order: the eval cases compile against existing value
constructors; execRange/assignTarget compile against the new eval helpers.

No CRITICAL/MAJOR findings.

## Assumptions

- `bindTargets` is refactored to delegate to `assignTarget` while preserving the
  existing `*ast.Ident` :=/=/compound behavior byte-for-byte.
- internal/interp stays dependency-clean (US-022 gate).
