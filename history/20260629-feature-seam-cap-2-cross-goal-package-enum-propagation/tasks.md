# Tasks — SEAM-CAP-2

## T1 — Foreign enrichment reads sibling .goal enums (internal)
- File: `internal/sema/foreign.go`.
- Import `goal/internal/parser`. In `foreignDecls`, split entries into non-test `.go` and
  `.goal`; when no `.go` and some `.goal`, delegate to new `goalForeignDecls`.
- Add `goalForeignDecls`, `isExportedName`, `qualifyForeignType`.
- Verify: `go test ./internal/sema/...`.

## T2 — Backend cross-package construction lowering (internal)
- Files: `internal/backend/lower.go` (add `enumRef`), `internal/backend/emit.go`
  (use it in `selectorExpr`, `variantLit`, `armBodyType`).
- Verify: `go test ./internal/backend/...`.

## T3 — Proof fixture + tests
- Fixtures: `internal/backend/testdata/goalenum/mood/mood.goal`,
  `internal/backend/testdata/goalenum/use/use.goal`.
- Test: `internal/backend/crosspkg_goal_enum_test.go` (transpile-shape + behavioral).
- Verify: `go test ./internal/backend/ -run CrossPackageGoal`.

## T4 — Mirror into selfhost
- Files: `selfhost/sema/foreign.goal`, `selfhost/backend/lower.goal`,
  `selfhost/backend/emit.goal` (same edits as T1+T2, `goal/selfhost/parser`).
- Verify: `task build`, `task fixpoint`.

## T5 — Gates + docs
- `task check`, `task build`, `task fixpoint` green; corpus behavioral tier unchanged.
- Record in DECISIONS.md and progress.txt.

Order: T1, T2 (independent) -> T3 (depends on T1+T2) -> T4 -> T5. Each task <= 5 files.

## Status: ALL TASKS COMPLETED (T1-T5)
