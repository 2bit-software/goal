# Technical Requirements / Research — US-001

## Established port pattern (from progress.txt Codebase Patterns + prior ports)

- Copy `internal/backend/*.go` (non-test) verbatim to `selfhost/backend/*.goal`
  (Go superset = valid goal). Grep for bare `match`/`enum`/`assert` identifier
  collisions first; backend has NONE (all occurrences are string-literal error
  messages or `token.MATCH/ENUM` constants).
- Add `TestPortedBackendPackage` to `internal/selfhost/port_test.go`:
  - COMPILE gate: `selfhost.BuildTranspiled(layout)` over the full dep closure
    keyed by module-relative dir.
  - BEHAVIORAL gate: `selfhost.BuildAndTest("internal/backend", pkg, testFiles, deps)`
    copying the EXISTING self-contained backend test files beside the generated Go.
- backend imports (in-module): ast, parser, pipeline, sema, project, token; plus
  transitive lexer. Foreign: fmt, go/format, go/importer, go/token, go/types,
  strconv, strings, unicode (pass through).
- Reuse the `discoverPorted(t, name)` helper.

## Test-subset problem

`internal/backend/backend_test.go` is a single monolithic file mixing 12
self-contained tests with many that depend on `goal/internal/corpus`, `repoRoot`,
and repo-relative `../../features`/`testdata` fixtures. The corpus import alone
prevents copying the whole file into the harness's throwaway temp module. Split the
self-contained subset into its own real test file so the behavioral gate can copy
a faithful, fixture-free suite (mirrors how sema's already-separate self-contained
suites were selected in US-009).

Self-contained tests (no corpus/repoRoot/fixture/mustRead deps):
TestInterfacesExist, TestGoFormatterFormats, TestASTEngineEmitsSwitch,
TestASTEngineEmitsGoGrammarForms, TestASTEngineLowersNestedOptionInResult,
TestASTEngineLowersOptionInValuePositions, TestASTEngineResolvesErrorOnlyArityByImport,
TestASTEngineUnwrapsErrorOnlyStdlibCall, TestPackageDoctestPreludeNotRedeclared,
TestASTEngineUnwrapsMethodCallResult, TestASTEngineEmitsIotaConstBlock,
TestASTEngineEmitsGenericFuncDecls.

## Fixpoint

`selfhost/` is auto-discovered by `task fixpoint` (project.Discover walks the
tree), so no harness change is needed for the new package to be covered.
