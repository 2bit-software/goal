# Plan Buildability Audit — US-008

- Dependency order valid: new function + new test, both in `internal/corpus`,
  reusing existing exported symbols (`Transpiler`, `Case`, `KindDoctest`,
  `Load`, `manifestPath`, `repoRoot`).
- Signatures agree: `RunDoctestExec(root string, c Case, tp Transpiler) error`
  matches the established runner shape (`RunCompile`, `RunDoctest`).
- File paths verified: `internal/corpus/doctest_behavior_runner.go` and its
  `_test.go` do not exist yet (no conflict).
- Temp-module recipe is copied from a compiling precedent (`RunCompile`).

No CRITICAL/MAJOR findings.

## Assumptions
- `repoRoot` (`../..`) and `manifestPath` (`../../corpus/manifest.json`) consts
  already declared in the package are reused, not redeclared.
