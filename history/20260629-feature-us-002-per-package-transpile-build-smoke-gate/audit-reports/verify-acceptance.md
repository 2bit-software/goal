# Verify — Acceptance Audit — US-002

Acceptance criteria (from prd.json US-002) vs. evidence:

- [x] A test transpiles each in-scope package and runs `go build` on the generated
      Go. `TestInScopePackagesTranspileAndBuild` does exactly this via
      `selfhost.BuildTranspiled` (temp module, `go build ./...`). PASS.
- [x] Covers token, lexer, ast, parser, sema, project, pipeline, backend.
      `selfhost.InScope` lists all eight; the test builds the layout for each. PASS.
- [x] Green after US-001. `go test ./internal/selfhost/...` -> ok; `task check` and
      `task build` both green. PASS.
- [x] Fails if any covered package transpiles to non-compiling Go.
      `TestGateFailsOnNonCompilingTranspile` feeds a package that transpiles to a
      bare-`return` int function and asserts `BuildTranspiled` returns an error.
      PASS. (Also empirically demonstrated during development: the gate caught
      `enum`-as-identifier transpile failures in sema and backend.)

No CRITICAL or MAJOR findings.

## Assumptions
- The gate proves *compilation* only; running each package's existing unit tests
  against the transpiled output is deferred to the per-package port stories
  (US-005+), per the spec's "where practical" and Out of Scope.
