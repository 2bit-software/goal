---
status: complete
updated: 2026-06-26
---

# Research: Arity-Aware `?` Lowering

## Executive Summary

The postfix `?` operator's **discard** lowering (a bare `expr?` statement, or `_ := expr?`)
always emits the two-value guard `if _, __goal_err := rhs; __goal_err != nil { … }`, which
hard-codes the assumption that the callee returns `(value, error)`. For an **error-only**
callee such as `os.MkdirAll(path)` (returns just `error`) this produces non-compiling Go
(`assignment mismatch: 2 variables but 1 value`). The fix is to make the lowering emit
`arity − 1` blank identifiers, where `arity` is the callee's actual return count, resolved
from `analyze.Tables` (in-file) and a new foreign-function table (imported Go, package mode).

## Findings

### Codebase Context

- **Open-E `?` lowering** lives in `internal/pass/question.go`. It keys behavior off the
  *enclosing* function's `Mode` (recovered by name from `analyze.Tables.FuncSignatures`),
  never the callee. The `ModeResult` discard branch hard-codes the `_, ` prefix
  (`question.go:52`). Non-discard `name := f()?` (`question.go:54`) binds exactly one value.
- **Closed-E `?`** (`internal/pass/closed.go`, `lowerClosedQuestions`) is a separate pass
  (pipeline step 6) over a single sum value — arity 1 by construction, unaffected.
- **Option `?`** lowers to a nil-check over `*T` (arity 1) — already a one-value form,
  unaffected.
- **Working-tree baseline**: uncommitted edits already added the bare `expr?` statement
  form (`scan.IsBareQuestionStmt`) and route both bare and `_ :=` through the same
  `discard := !ok || name == "_"` path — which is exactly the two-value path that breaks on
  error-only callees. This feature completes that work.
- **`analyze.FuncSig`** (`internal/analyze/analyze.go:36`) records `Mode`, `T`, `E` — **no
  return arity**. `analyzeSig` (`analyze.go:635`) reads only the Result/Option shape; a
  `ModeNone` function such as `func f() error` carries no arity today.
- **Foreign enrichment** (`internal/analyze/foreign.go`, `EnrichForeign`) already resolves
  imports, parses imported Go with `go/parser`, and records **struct field sets** keyed
  `alias.Type`. It parses **no `func` declarations** today. It runs only in package mode:
  the package driver `pipeline.TranspilePackage` (`internal/pipeline/pipeline.go:169`) calls
  it; single-file `Transpile` (`pipeline.go:78`) uses bare `analyze.Build` and is
  "foreign-blind".
- **No open-E `?` checker** exists in `internal/check/`; validation is inline in the
  Question pass. The must-use checker (`internal/check/mustuse.go`) only verifies a Result
  is consumed by `?` and is independent of arity.
- **`scan.LeadIdent`** returns the first identifier only; there is **no helper** to extract a
  qualified `pkg.Func` selector from an rhs string. One must be added.
- **Pass ordering** (`pipeline.go:28`): `result` (3) lowers `Result[T,error]` signatures to
  named `(T, error)` returns **before** `question` (5) runs, so at `?`-lowering time an
  in-file Result callee already presents as a 2-value Go call.

### Foreign-arity feasibility

- `foreign.go` already has the directory resolver (`DefaultResolver`: same-module walk →
  `go list` fallback) and a `go/parser` pass over the package's `.go` files. Adding a
  `foreignFuncs(dir)` sibling to `foreignStructs` that records each exported top-level
  `func`'s result count is a localized extension.
- `neededAliases` (`foreign.go:173`) currently collects aliases only from `derive`/`from`
  type positions; it must also collect aliases that appear as `?` callees (`alias.Func(` at
  the head of a `?` rhs) so the right imports are parsed.
- Methods (`f.Close()?`) require receiver-type inference the toolchain does not do — out of
  scope; they fall back to the default.

## Decision Points

- [x] **D1 — Where to store foreign arity**: add `Arity int` to `FuncSig` and store foreign
  functions in `FuncSignatures` keyed `alias.Func` (distinct from bare in-file keys, so no
  collision). One lookup path in the pass. *(Recommended; final shape settled in plan.)*
- [x] **D2 — In-file arity source**: extend `analyzeSig` to set `Arity` for every function:
  `ModeResult` → 2, `ModeOption`/`ModeResultClosed` → 1, `ModeNone` → raw return-value count.
- [x] **D3 — Single-file / unresolved-foreign fallback**: when callee arity is unknown
  (foreign callee in single-file mode, method call, or non-call rhs), keep **today's two-value
  form** unchanged. A bare-vs-`_:=` heuristic was rejected: bare `expr?` and `_ := expr?` are
  semantically identical discards (`question.go:45`), so inferring arity from which was written
  would *regress* a currently-compiling `bare os.Open(p)?` (value discard) to non-compiling Go.
  In-file callees are always resolved (tables built from the file), so only foreign single-file
  calls hit this default — error-only foreign `?` in single-file therefore stays unsupported,
  exactly the accepted "package-mode-only" limitation. The clean `os.MkdirAll` win lands in
  package mode where foreign arity is resolved.
- [x] **D4 — Non-discard `name := f()?`**: keeps its single-value bind (arity 2). When the
  callee **resolves** to arity ≠ 2, emit a goal-level diagnostic instead of silent malformed
  Go (FR-009 — required for MVP so no resolvable case ships broken output).

### Precedent & test harnesses (confirmed)

- `internal/fix/propagate.go` already encodes this arity distinction in the Go→goal direction
  ("the call's sole output is the error … a bare `CALL?` discards nothing"); reuse its model.
- `?` output is regression-locked by golden `features/05-question-prop/examples/qprop_*.go.
  expected` files, NOT inline tests — the byte-for-byte lock for FR-006.
- SC-001 ("compiles") is provable via the `go build ./...` package harness
  (`internal/pipeline/pipeline_package_test.go`, `TestTranspilePackageCrossFile`).
- Foreign-arity fixtures live at `internal/{analyze,pipeline}/testdata/extpkg/` (struct-only
  today — must gain a `func` decl).

## Recommendations

1. Implement resolution-based arity (in-file always; foreign in package mode) as the core,
   with the bare-vs-`_:=` heuristic as the unresolved fallback (D3). This delivers clean
   `os.MkdirAll(p)?` in package mode and never regresses existing two-value call sites.
2. Add a `scan` helper to extract a `pkg.Func` callee key from an rhs; reuse it in both the
   pass and the foreign alias collector.
3. Test foreign arity through the existing `DirResolver`/`testdata` fixture pattern
   (`internal/analyze/foreign_test.go`) — no dependency on the real stdlib.

## Sources

- `internal/pass/question.go`, `internal/pass/closed.go`
- `internal/analyze/analyze.go`, `internal/analyze/foreign.go`, `internal/analyze/foreign_test.go`
- `internal/pipeline/pipeline.go`
- `internal/check/mustuse.go`, `internal/scan/scan.go`
- Working-tree diff adding `IsBareQuestionStmt` (bare `expr?` statement form)
