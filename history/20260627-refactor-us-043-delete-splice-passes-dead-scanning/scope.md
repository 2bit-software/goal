# Scope — US-043 Delete splice passes and dead scanning

## What is being refactored and why

The token-splice front-end (`internal/pass` + the `internal/pipeline` functions
that orchestrate it) was superseded by the AST front-end (lexer → parser → ast →
sema → backend), which became the default in US-042. The splice engine is now
dead weight that implies two front-ends exist. This story removes it so there is
exactly one.

## Old code (to delete)

- `internal/pass/*` — the entire splice lowering pass package. Imported ONLY by
  `internal/pipeline/pipeline.go`.
- `internal/pipeline` splice functions: `Pass` type, `Passes` var, `Transpile`,
  `transpileWith`, `runPasses`, `doctestFile`, `TranspilePackage`,
  `preludeFile`, and any helper left dangling (`goName`/`testName`).
- `internal/scan` symbols used ONLY by `internal/pass` (dead after its removal).
  NOTE: `scan.Splice`/`scan.Replacement` are STILL used by `internal/fix`
  (US-044 not done) and `scan.Lex`/token helpers by lsp/check/project — those
  STAY.
- `internal/analyze` token-scan tables/functions used ONLY by `internal/pass`
  (subsumed by `internal/sema`). NOTE: analyze is still used by
  check/fix/lsp/typecheck/backend — only the genuinely-dead subset is removed.

## Call sites to re-point onto the AST backend

- `cmd/goalc/main.go`: `pipeline.Transpile` → `backend.Transpile`
- `cmd/build-playground/main.go`: `pipeline.Transpile` → `backend.Transpile`
- `cmd/goal-wasm/main_wasm.go`: `pipeline.Transpile` → `backend.Transpile`
- `internal/guide/guide.go`: `pipeline.Transpile` → `backend.Transpile` (2 sites)
- `internal/typecheck/typecheck.go`: `pipeline.TranspilePackage` →
  `backend.TranspilePackage`
- `internal/corpus` behavioral tests (`behavior_runner_test.go`,
  `doctest_behavior_runner_test.go`, `package_runner_test.go`):
  `pipeline.Transpile`/`TranspilePackage` → `backend.*`
- `cmd/goal/main.go`: remove the `--engine=splice|ast` flag entirely; the driver
  always uses `backend.TranspilePackage`. Update usage text + `parseFlags` +
  `transpileAll` + `main_test.go`.

## What must NOT change (preserve)

- The shared output types `pipeline.Output`, `pipeline.GoFile`,
  `pipeline.PackageOutput`, and `pipeline.AddLineDirectives` (backend depends on
  them).
- The AST backend (`internal/backend`), `internal/sema`, `internal/parser`,
  `internal/lexer`, `internal/ast`.
- The behavioral conformance corpus (manifest, goldens, fixtures) and the
  whole-corpus behavioral gate.
- `internal/scan` and `internal/analyze` parts still consumed by
  fix/lsp/check/typecheck/project/backend.

## Acceptance

- `grep` finds no references to the deleted symbols.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.
