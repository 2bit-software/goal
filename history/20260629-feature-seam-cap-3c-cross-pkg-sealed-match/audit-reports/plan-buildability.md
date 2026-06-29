# Plan Buildability Audit

## Findings
- Dependency order valid: capability (foreign.go) → mirror (foreign.goal) → fixtures →
  tests. No forward references.
- Interface contract self-consistent: 6th return `sealed map[string][]string`; single call
  site updated in each file. `EnrichForeign` merge uses existing nil-init pattern.
- File paths verified against the tree: foreign.go/foreign.goal exist; testdata dirs follow
  the established `goalenum`/`extpkg` precedent; test files follow `crosspkg_goal_enum_test.go`.
- Integration point is specific (EnrichForeign ~line 95; downstream sealedInterfaceOf
  unchanged). Reuses qualifyForeignType/isExportedName/ResolvePackage — all present.
- No CRITICAL/MAJOR.

## Assumptions
- selfhost/sema/foreign.goal compiles as Go-superset; the multi-value return with 6 values
  is valid goal (matches the existing 5-value return form already in the file).
- The PostToolUse `task check` hook will flag any mirror drift immediately during edits.
