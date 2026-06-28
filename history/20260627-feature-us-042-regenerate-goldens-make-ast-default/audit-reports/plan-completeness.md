# Plan Audit: Coverage — US-042

Every FR/AC maps to a plan element:
- FR-1/AC default-engine -> cmd/goal/main.go + main_test.go.
- FR-2/AC goldens=AST output -> update_goldens_test.go regeneration step.
- FR-3/AC exact tier green -> 4 exact-tier tests switched to backend.Transpile.
- FR-4/AC splice available -> --engine=splice branch + behavioral-tier coverage
  retained (no deletion).
- FR-5/AC docs -> usage text + AI-KNOWLEDGE-BOOTSTRAP.md regen.
- Behavioral gate -> US-041 ast_gate_test.go unchanged, re-run in verify.

No scope creep: no plan element is unmapped. No CRITICAL/MAJOR.

## Assumptions
- Exact-tier tests move to backend.Transpile (not dual-engine).
