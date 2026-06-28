# Implementation Plan — US-004

## Components & changes

### cmd/goal/main.go (modify)

1. Remove the `"goal/internal/check"` import.
2. `checkPackage`:
   - Replace `perFile, err := check.AnalyzePackageInDir(srcs, pkg.Dir)` with
     `perFile, err := sema.AnalyzePackageInDir(srcs, pkg.Dir)`.
   - `perFile` is now `[][]sema.Diagnostic`.
   - In the per-file loop, drop `p := check.OffsetToPosition(...)`. Use the
     diagnostic's own position: `d.Pos.Line`, `d.Pos.Col`.
   - Dedup key for a sema finding: `dedupKey(path, d.Pos.Line, d.Feature)`.
   - Build `checkDiag{path, d.Pos.Line, d.Pos.Col, d.Severity, d.Code, d.Message}`.
   - Depth-stage findings (unchanged loop) now convert severity:
     `sema.Severity(d.Severity)` when constructing the `checkDiag`.
3. `checkDiag.severity` field type: `check.Severity` -> `sema.Severity`.
4. `cmdCheck`: `d.severity == check.Error` -> `d.severity == sema.Error`.
5. Update the doc comment on `checkPackage` to say the lexical-equivalent stage
   is the AST/sema checker (was internal/check).

### Interface contracts

- `sema.AnalyzePackageInDir(srcs []string, dir string) ([][]sema.Diagnostic, error)`
- `sema.Diagnostic{ Pos token.Pos; Severity sema.Severity; Feature, Code, Message string }`
  where `token.Pos` has `.Line int`, `.Col int`.
- `checkDiag{ file string; line, col int; severity sema.Severity; code, message string }`
- `sema.Severity(typecheckDiag.Severity)` — numeric conversion, no check import.

## Integration points

- `cmd/goal/main.go` `checkPackage` <- `sema.AnalyzePackageInDir`
  (internal/sema/package.go).
- Depth stage unchanged: `runDepthChecks` -> `typecheck.GoTypesChecker.Check`
  still returns `[]typecheck.Diagnostic`.

## Testing strategy

- `cmd/goal/main_test.go` (modify): existing TestCheckDepthStageCatchesElidedLiteral,
  TestCheckCleanProgramPasses, TestCheckDepthNoteOmitsGeneratedDump must still
  pass (they exercise the rewired path).
- Add `TestCheckCorpusOutputUnchanged` (new): drive `goal check` over a corpus
  KindCheck case directory and assert the rendered findings match the expected
  set (sema behavior, accounting for any DECISIONS.md divergence). Pick a case
  with a stable, non-divergent finding so the assertion is exact, OR assert the
  presence of the expected `[code]` lines.
- verifyCommands: `task check`, `task build`.

## Dependency order

1. Edit main.go (import, struct field, checkPackage body, cmdCheck).
2. Add e2e test.
3. Run `task check` + `task build`.
