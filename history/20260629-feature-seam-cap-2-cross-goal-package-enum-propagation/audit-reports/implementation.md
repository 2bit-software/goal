# Implementation Audit — SEAM-CAP-2

## Acceptance criteria (source of truth)
- [x] Enum defined in a sibling `.goal` package resolved by consumers in other `.goal`
      packages — `goalForeignDecls` (parse + ResolvePackage + project to `info.Enums`).
- [x] 2-package fixture (sibling `.goal` enum) transpiles a cross-package `match` and
      behaves identically to the switch — `TestCrossPackageGoalEnumMatchLowers` +
      `TestCrossPackageGoalEnumBehavesLikeSwitch` (build+run vs reference switch), PASS.
- [x] Bare cross-package construction lowers to `pkg.Enum(pkg.Enum_Variant{})` — asserted
      on emitted Go (`mood.Mood(mood.Mood_Happy{})`).
- [x] `task check`, `task build`, `task fixpoint` (FIXPOINT OK) all green; corpus
      behavioral tier unchanged (additive fixture).
- [x] Applied in BOTH internal/ and selfhost/.

## Findings
No CRITICAL/MAJOR. 

- MINOR: `qualifyForeignType` is best-effort for payload variant fields; not exercised
  (tested enums and the unblocked SEAM-002/003 enums are tag-only). Documented.
- MINOR: struct/func/method foreign facts from `.goal` source intentionally not projected
  (enum keystone scope); strictly additive, no regression.

## Verification evidence
- `go test ./internal/backend/ -run CrossPackageGoal` — PASS (both tests).
- `task check` — full `go test ./...` + `go vet` green, including selfhost port gates.
- `task build` — both binaries.
- `task fixpoint` — FIXPOINT OK (stage1 == stage2, byte-identical).

## Assumptions
- `.go` enrichment path takes precedence when a dir holds both `.go` and `.goal`.
- Tag-only enums are the relevant/exercised case; payload requalification best-effort.
- Fixture under `internal/backend/testdata/goalenum/` so module-relative imports resolve
  via `moduleResolve`.
