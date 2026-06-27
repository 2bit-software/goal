# Plan Audit — Coverage

Every spec FR/AC maps to a plan element:

- FR-1 (on-disk fixtures) → `testdata/package/{cross-file-demo,foreign-derive}/*.goal`.
- FR-2 (declared import map) → `pkg.json` `imports` field per fixture.
- FR-3 (Mode=package indexing, counts unchanged) → `generate.go` discovery +
  regenerated manifest + count test (51 file-mode / 2 package / 50 / 4).
- FR-4 (package runner) → `package_runner.go` `RunPackage` + test.
- Error handling (descriptive failure, zero-case guard) → `RunPackage` errors +
  test `t.Fatalf` guard.

No scope creep: no plan element lacks a requirement. No CRITICAL/MAJOR.

## Assumptions
- Two package fixtures (cross-file + foreign) are the complete set to reify; no
  other inline package sources exist (`internal/analyze/foreign_test.go` and
  `internal/check/foreign_test.go` are checker/analyze-level, out of scope).
