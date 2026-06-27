# Verification — US-038

## Acceptance criteria

- **AC1 — lower+backend expand `...defaults` to explicit zero values and assert to
  an if-panic guard.** Met. `internal/backend/{lower.go,emit.go}` add the
  `...defaults` expansion (compositeLit/defaultEntries + zeroLit/zeroSafety) and the
  `assert` lowering (assertStmt + fmt-import injection). Shapes pinned by
  `TestASTEngineDefaultsAssertEncoding` and `TestASTEngineDefaultsUnsafeZeroRejected`.
- **AC2 — the 08-no-zero-value and 10-assert cases pass the behavioral tier through
  the new backend.** Met. `TestASTEngineDefaultsAssertBehavioralTier` runs all 6
  cases (3 + 3) through `backend.Transpile` + `corpus.RunCompile` (temp-module
  `go build` + `go vet`); all pass.

## Project gates (prd.json verifyCommands)

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages ok)

## Findings

None (CRITICAL/MAJOR/MINOR). Output matches the goldens modulo dropped `//`
comments and the `type Name = string` alias gap (US-042); both build+vet clean.
