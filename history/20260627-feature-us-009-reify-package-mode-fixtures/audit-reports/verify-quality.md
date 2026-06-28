# Verify — Quality

- Error handling: `RunPackage` returns descriptive, case-identified errors for
  read/transpile/invalid-Go/build failures; `TestPackageRunner` has a loud
  zero-case guard (`t.Fatalf`) matching the other corpus runners.
- No regressions: the existing file-mode transpile/compile loops (runner_test.go,
  behavior_runner_test.go, pipeline_test.go) were scoped to `Mode=file` so they
  do not misread a package fixture's directory `Input` as a file.
- Tests assert real behavior: the foreign-derive case actually compiles its
  generated Go against the wired-in `extpkg` — strictly stronger than the old
  inline test, which only string-matched the output.
- Generated manifest is deterministic (`TestGenerateDeterministic` still green)
  and the fixtures are non-destructive additions under `testdata/package/`.

No CRITICAL or MAJOR findings.

## Assumptions
- Temp build module is named after the repo's module (`goal`) so the in-module
  foreign import path resolves; foreign packages are copied flat under their
  import-path tail.
- The inline Go tests in `internal/pipeline/{foreign,pipeline_package}_test.go`
  are left in place (the fixtures are additive); removing them is optional
  cleanup outside this story's scope.
