# Technical Requirements / Research — US-004

## Implementation hints (from the story)

- `cmd/goal/main.go` `checkPackage` must call the US-002 sema driver
  `sema.AnalyzePackageInDir(srcs, dir) ([][]Diagnostic, error)` instead of
  `check.AnalyzePackageInDir`.
- `cmd/goal/main.go` must no longer import `goal/internal/check`.
- Drop `check.OffsetToPosition`: `sema.Diagnostic.Pos` is a `token.Pos` that
  already carries `Line`/`Col`.
- Keep the `checkDiag` struct and its `render()`. `checkDiag.severity` becomes
  `sema.Severity`. Depth diagnostics carry `typecheck.Diagnostic.Severity`
  (underlying `check.Severity`, an `int` with Error=0/Warning=1); convert via
  `sema.Severity(d.Severity)` — a numeric type conversion that does NOT require
  importing `internal/check` (both severities share the int underlying type and
  Error=0/Warning=1 ordering).
- `cmdCheck`'s error tally `d.severity == check.Error` becomes
  `d.severity == sema.Error`.
- Preserve the dedup: build the `suppress` map keyed by
  `dedupKey(basename, line, feature)` from depth findings; skip a sema finding
  whose `(path, Pos.Line, Feature)` key is suppressed.

## Parity safety net

- US-003 added `internal/corpus/parity_test.go`, proving sema and legacy agree
  field-for-field on the corpus (minus DECISIONS.md-documented divergences). This
  is the corpus-wide guarantee the rewire is safe.
- US-002 added `sema.AnalyzePackageInDir` / `AnalyzePackageInDirWith` mirroring
  the legacy `check.AnalyzePackageInDir` shape.

## End-to-end test

- Add a cmd/goal `_test.go` that runs `goal check` over a corpus check case and
  asserts the rendered output matches expectations (unchanged from before the
  rewire). Existing e2e tests TestCheckDepthStageCatchesElidedLiteral,
  TestCheckCleanProgramPasses, TestCheckDepthNoteOmitsGeneratedDump already pin
  much of the behavior.

## verifyCommands (from prd.json)

- `task check`
- `task build`
