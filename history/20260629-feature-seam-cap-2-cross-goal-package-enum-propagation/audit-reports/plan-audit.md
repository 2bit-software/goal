# Plan Audit

## Traceability
- FR-1 -> C1 (goalForeignDecls projects sibling-.goal enums into info.Enums).
- FR-2 -> existing matchQualifier + enumOf (SEAM-CAP) now fed by C1; proven by test.
- FR-3 -> C2 (enumRef in selectorExpr/variantLit/armBodyType).
- FR-4 -> C3 (selfhost mirror of C1+C2).
- All acceptance criteria map to the test fixture + gates.

## Findings
- **MINOR** — `qualifyForeignType` is best-effort; acceptable because the unblocked enums
  (FuncMod/ChanDir/Mode/Severity) and the test fixture are tag-only (no fields).
- **MINOR** — File paths verified: foreign.go/foreign.goal, lower.go/lower.goal,
  emit.go/emit.goal exist at matching line numbers; selfhost/sema already imports
  goal/selfhost/parser (package.goal), so no cycle.

No CRITICAL/MAJOR. Dependency order valid (enrichment + lowering are independent;
test depends on both). Plan is implementable.

## Assumptions
- `.go` path precedence preserved when both forms exist.
- Enum-only projection from .goal source (structs/funcs/methods deferred).
