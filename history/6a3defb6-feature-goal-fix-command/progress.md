# Progress Log: `goal fix`

## Summary

`goal fix` is implemented, tested, and working end-to-end. It rewrites plain-Go patterns in
`.goal` files into idiomatic goal — the keystone being `(T, error)` + manual `if err != nil`
propagation → `Result[T, error]` + `?` — with stdout (default) and `-inplace` modes.

## Tasks

### T001 — export shared helpers — Complete
- Files: `internal/analyze/spans.go` (new: `FuncSpan`/`FuncSpans`/`SigAt`/`ZeroLit`),
  `internal/pass/pass.go` (thin aliases), `internal/pass/defaults.go` (call `analyze.ZeroLit`).
- Behavior-preserving relocation; `internal/pass` + `internal/analyze` tests stay green.

### T002 — fix package skeleton + orchestrator — Complete
- Files: `internal/fix/fix.go`. `File(src) (out, changes, reports)` runs all fixers to a
  fixed point; mutating-rule reports accumulate (deduped by rule+message, line-independent),
  the post-hoc call-site analysis runs once at convergence.

### T003 — CLI wiring — Complete
- Files: `cmd/goal/main.go` (`fix` in `guideCommands`, dispatch case, `parseFixFlags`,
  `cmdFix`). Accepts a `.goal` file or directory; default `.`; stdout default; `-inplace`
  writes changed files only; reports → stderr; operational errors only → non-zero exit.

### T005/T006/T007 — propagation collapse — Complete
- Files: `internal/fix/propagate.go`. Result keep/discard + Option (`*o`→`o`) collapse to
  `?`; DR-5 comment guard and DR-6 multi-line guard; strict matching with skip+report.

### T009/T010/T011 — signature conversion + fixed point — Complete
- Files: `internal/fix/resultsig.go`. `(T, error)` → `Result[T, error]`, all-or-nothing per
  DR-3; `return v, nil` → `Result.Ok(v)`; bare propagations left for the next pass to
  collapse; multi-value / bare-error / decorated returns reported; exported-change warning.
  Local `paramsClose` works around `scan.ParamsClose` mishandling parenthesized returns.

### T013 — call-site reporting — Complete
- Files: `internal/fix/callsite.go`. At convergence, flags manual error handling left in
  functions that are not Result-returning (so `?` is not legal there).

### T015 — switch→match — Complete (as detection)
- Files: `internal/fix/match.go`. A `switch` over an in-file enum is **reported** as a
  `match` candidate. Decision: goal `match` arms are single expressions, so a faithful
  mechanical rewrite of statement-bodied Go `switch` clauses is not expressible — detection
  is the safe, useful step. Auto-rewrite deferred (noted in spec).

### T004/T008/T012/T016/T017 — tests — Complete
- Files: `internal/fix/fix_test.go` (conversion, idempotence, collapse, Option, decorated
  skip, comment guard, multi-value skip, switch detection), `cmd/goal/fix_test.go` (stdout
  no-write, `-inplace`, no-op unchanged, exported warning, bad path, directory).

### T018/T019 — docs + full suite — Complete
- README `goal fix` quick-start + prose; regenerated `AI-KNOWLEDGE-BOOTSTRAP.md`.
- `go build ./...`, `go vet ./...`, `gofmt -l`, and `go test ./...` all clean.

## Decisions / notes

- **switch→match is detection-only** for the MVP (language has no statement-block match
  arms). Recorded as a deliberate scope choice, not a gap.
- **Option pointer→Option signature conversion** stays suggestion territory (US3); the
  Option *collapse* (inside an already-Option function) is implemented.
- **No branching** per the user's request: work committed directly on `main`, no PR.
- Correctness backbone: the transpiler's `?`/Result passes are the inverse of fix, so a
  converted file lowers to the same Go the original did (verified by the spike + oracle).
