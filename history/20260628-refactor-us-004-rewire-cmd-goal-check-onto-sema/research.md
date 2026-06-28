# Research — US-004

This is an internal behavior-preserving refactor; research is codebase-local
(no external/library investigation needed).

## Findings

- `cmd/goal/main.go` `checkPackage` (lines ~554-595) calls
  `check.AnalyzePackageInDir(srcs, pkg.Dir)` then maps each diagnostic's byte
  offset via `check.OffsetToPosition(src, d.Pos)`. The depth stage
  (`runDepthChecks` -> `typecheck.GoTypesChecker.Check`) produces
  `typecheck.Diagnostic` with a resolved `token.Position`. The dedup builds a
  `suppress` map keyed by `dedupKey(basename, line, feature)` from depth findings
  and drops the lexical finding for any suppressed key.

- `sema.AnalyzePackageInDir(srcs []string, dir string) ([][]Diagnostic, error)`
  (internal/sema/package.go, US-002) is a drop-in shape match. `sema.Diagnostic`
  has `Pos token.Pos` (with `.Line`/`.Col`), `Severity sema.Severity`, `Feature`,
  `Code`, `Message` — so `check.OffsetToPosition` is not needed.

- `sema.Severity` and `check.Severity` are both `int`-based with `Error = iota`
  (0) and `Warning` (1). So `sema.Severity(d.Severity)` converts a depth
  finding's severity without importing `internal/check`.

- `checkDiag.severity` is currently `check.Severity`; change to `sema.Severity`.
  `cmdCheck`'s tally `d.severity == check.Error` -> `d.severity == sema.Error`.

- Safety net: `internal/corpus/parity_test.go` (US-003) already proves sema vs
  legacy agree field-for-field over the corpus modulo DECISIONS.md divergences.

## Confidence

High — all symbols verified directly in the tree.

## Open questions

None. The conversion path is mechanical and the parity gate guards correctness.
