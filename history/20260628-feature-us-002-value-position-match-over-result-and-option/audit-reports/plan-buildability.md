# Plan Buildability Audit — US-002

- Dependency order valid: signature generalization (1) before matchStmt update (2)
  before dispatch wiring (3) before test (4); no forward references.
- Signature contract: `resultMatch/closedResultMatch/optionMatch(m, pos, name)`
  consistent with existing `enumMatch(m, pos, name)` and `armWrap(body, pos, name)`.
- `armBodyRenamedWrap(body, binding, target, pos, name)` mirrors existing
  `armBodyRenamed` + `armWrap`; types match (ast.Node, string, matchPos, string).
- File paths verified to exist (internal/backend/emit.go, backend_test.go).
- Each step compiles independently: step 1 keeps callers compiling by updating
  matchStmt in the same change (step 2 folded in); steps 3-4 additive.

No CRITICAL/MAJOR findings. Plan is buildable.
