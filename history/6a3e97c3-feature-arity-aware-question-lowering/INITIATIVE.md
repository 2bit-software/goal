# Initiative: arity-aware-question-lowering

**Type**: feature
**Status**: in_progress
**Created**: 2026-06-26
**ID**: 6a3e97c3-feature-arity-aware-question-lowering

## Completion

**Completed**: 2026-06-26 09:05
**Branching**: none — committed directly to `main` per request (no push, no PR).

### Outcomes
- Feature: arity-aware `?` lowering — **Complete**. The discard `?` form now emits `arity − 1`
  blank identifiers, resolving callee arity from in-file signatures (`FuncSig.Arity`) and
  imported package-level functions (foreign enrichment, package mode). Error-only callees
  (`os.MkdirAll`, in-file `func clean() error`) lower to the one-value error guard and compile.

### Verification
- `task check` (vet + full suite) green; new unit/golden/compile/fallback/diagnostic tests pass.
- Existing `qprop_*` goldens unchanged (no regression).

### Notes
- The pre-existing bare `expr?` statement work in the working tree is the foundation this
  feature completes; both ship in the same commit.

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-26 08:16 |
| plan | plan | complete | 2026-06-26 08:40 |
| tasks | tasks | complete | 2026-06-26 08:45 |
| implement | implement | complete | 2026-06-26 09:05 |

## Description

Make the postfix `?` operator arity-aware. The discard form (bare `expr?` / `_ := expr?`)
currently always emits the two-value guard `if _, __goal_err := rhs; …`, which fails to
compile for an error-only callee like `os.MkdirAll`. Resolve the callee's actual return
arity — in-file via `analyze.Tables.FuncSignatures`, foreign (package mode) by extending
`internal/analyze/foreign.go` to record imported-function return arity — and emit `arity − 1`
blank identifiers. Worked directly on `main` (no branching, per request).

## Goals

- Clean `?` on error-only callees (`os.MkdirAll`, `func clean() error`) — emitted Go compiles.
- No regression to existing `?` output (binding form, `_ :=` discard, closed-E, Option).
- Foreign arity resolved in package mode; sensible syntactic fallback when foreign-blind.

## Progress

- 2026-06-26: spec + research complete. Two parallel audits (completeness + AI-consumer)
  run; CRITICAL fallback-regression and MAJOR test-strategy/non-discard findings resolved in
  `spec.md` (FR-005/006/009/010, US3) and `research.md` (D3). See `spec.md`, `research.md`.
- 2026-06-26: plan + technical-spec complete (`implementation-plan.md`, `technical-spec.md`),
  with inline reuse audit. Plan audit found 1 CRITICAL (C1: `f.ParamsClose` is the parenthesized
  return type's `)`, not the param list's — fixed to slice from `MatchParen(toks, NameTok+1)`)
  plus minor `lastSegment`/`CalleeKey`-edge fixes; all resolved. Foreign-merge safety,
  golden-stability, and call-site completeness verified against code.
- 2026-06-26: tasks complete (`tasks.md`) — 14 atomic tasks, TDD-first on arity slice + CalleeKey.
- 2026-06-26: implement complete (`progress.md`) — all 14 tasks done; `task check` (vet + full
  suite) green. New: `analyze.FuncSig.Arity`, `scan.CalleeKey`, foreign func-arity enrichment,
  arity-aware `?` discard lowering + FR-009 diagnostic, error-only golden, foreign-resolution +
  `go build` compile tests. Pre-existing bare-`?` working-tree changes ship in the same commit.
