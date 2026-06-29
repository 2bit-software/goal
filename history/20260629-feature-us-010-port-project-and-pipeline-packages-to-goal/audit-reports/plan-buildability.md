# Plan Audit: Buildability — US-010

## Findings

No CRITICAL or MAJOR findings.

- Dependency order is valid: token/lexer/ast/parser already ported (US-005..008);
  project and pipeline only add leaves above them.
- Interface contracts unchanged (verbatim port); harness signatures verified
  against internal/selfhost/port_test.go and selfhost.go.
- File paths verified real (selfhost/ exists; internal/project + internal/pipeline
  sources and tests confirmed present).
- Integration point is concrete: port_test functions mirror TestPortedSemaPackage.

### MINOR-1: lexer in deps despite no direct import
project/pipeline don't import lexer directly, but the transpiled parser does, so
the layout and deps maps must include internal/lexer. Captured in the plan.

## Assumptions
- No edit to selfhost.go is needed (existing deps-aware harness suffices) — true
  since the sema port used the identical multi-dep shape.
