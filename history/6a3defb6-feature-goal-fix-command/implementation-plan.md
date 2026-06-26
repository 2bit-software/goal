# Implementation Plan: `goal fix`

Ordered by dependency. Each phase is independently shippable and testable. P1/P2 deliver the
user's primary ask (plain-Go `(T,error)`+propagation → `Result`+`?`); P3 is a stretch.

## Phase A — Enabling refactor + CLI skeleton (vertical slice)

1. **A1 (refactor)**: Relocate `funcSpans`/`sigAt` → `analyze.FuncSpans`/`SigAt` and
   `zeroLit` → `analyze.ZeroLit` (exported); repoint `internal/pass`. Behavior unchanged;
   existing `internal/pass` tests must stay green. *(reuse-audit refactor)*
2. **A2**: Create `internal/fix` with types (`Change`, `Report`, `Level`) and
   `func File(src string, t *analyze.Tables) (out string, changes []Change, reports []Report, err error)`
   that initially returns `src` unchanged. Unit test: no-op file in → identical out.
3. **A3**: Wire CLI — add `fix` to `guideCommands`, `case "fix"` in `run()`, `parseFixFlags`
   (`-inplace`, file-or-dir path, default `.`), and `cmdFix` (Discover → `fix.File` →
   stdout default / `-inplace` write-changed-only / reports to stderr).
4. **A4 (CLI tests)**: `goalModule`+`run`: `fix` on an already-idiomatic file prints it
   unchanged and writes nothing; `-inplace` on unchanged file does not rewrite; bad path →
   non-zero exit. **Milestone: `goal fix` exists end-to-end and is a safe no-op.**

## Phase B — Propagation collapse engine (`propagate.go`) — body-local, ripple-free

5. **B1**: Implement the Result keep/discard collapse (DR-1/DR-2): inside `ModeResult`
   functions, `v, err := g(); if err != nil { return <zeros>, err | Result.Err(err) }` →
   `v := g()?` (and `_ := g()?`). Strict matching; skip+Report on any deviation.
6. **B2**: Implement the Option collapse + `*o` rewrite (DR-4) inside `ModeOption` functions;
   skip+Report if `o` escapes.
7. **B3**: Safety guards: DR-5 (comment in span → skip), DR-6 (multi-line RHS → skip).
8. **B4 (golden)**: `testdata/fix/propagate/{before,after}.goal` for keep/discard/option;
   negative fixtures (wrapped, decorated, comment, multi-line, escaping pointer); idempotence
   + oracle (`lower(after)` valid). **Milestone: the `?`-collapse the user named, for
   already-`Result` functions.**

## Phase C — Signature conversion (`resultsig.go`) — couples with B for plain-Go input

9. **C1**: Detect convertible `(T, error)` functions (DR-3 whole-function check using
   `analyze.ZeroLit` to confirm every early return is a bare-propagation zero and the success
   return is single-valued). Non-conforming → Skip+Report `file:line`.
10. **C2**: Rewrite signature `(T,error)`→`Result[T,error]` and returns
    (`return v, nil`→`return Result.Ok(v)`); set `signatureChanged` so the orchestrator
    rebuilds tables; bare props then collapse via Phase B next iteration.
11. **C3**: Fixed-point loop in `fix.File` (re-lex + rebuild tables on signature change,
    bounded iterations) — gives FR-011 and orders B-after-C automatically.
12. **C4 (golden + CLI)**: plain-Go `before.goal` (the spike fixture) → idiomatic
    `after.goal`, asserting `lower(after) == lower(before)`; exported function → stderr
    warning (DR-9/FR-008). **Milestone: plain-Go `(T,error)`+propagation → `Result`+`?`
    end-to-end — the user's headline example.**

## Phase D — Call-site reporting (`callsite.go`)

13. **D1**: After conversions, enumerate call sites of converted functions; legal in-package
    Result/Option callers collapse via Phase B (no extra code); illegal sites
    (non-Result caller, closure/`defer`/`go`) → Suggest report "manual follow-up" (FR-007).
14. **D2 (tests)**: same-package caller inside a Result fn collapses; caller not in a
    Result fn is left unchanged + reported.

## Phase E — switch→match (`match.go`) [P3, stretch]

15. **E1**: `switch` over in-file enum (`t.Enums` lookup via `scan.LeadIdent`/case qualifier)
    → `match`, preserving arm bodies, dropping `break` (DR-8). Body-local.
16. **E2 (golden)**: switch-over-enum → match; non-enum switch untouched.

## Phase F — Polish & docs

17. **F1**: Idempotence/fixed-point test across all rules on a kitchen-sink fixture; verify
    SC-002/SC-005 (no disk writes in default mode; only changed files written by `-inplace`).
18. **F2**: Docs — README subcommand entry + a line in the `goal ai` guide; `goal fix --help`
    usage text.
19. **F3**: Run full `go test ./...` + `goal check` on fixtures; ensure zero-dependency and
    no regression in `internal/pass`.

## Dependencies & traceability

- A → B → C → D → (E) → F. C depends on B (collapse), B/C depend on A (exported helpers).
- Spec coverage: FR-001/002/003/004/013/014→A; FR-005/010/011→B; FR-006/008/011→C;
  FR-007→D; FR-009→E; FR-012 across B/C/E. SC-001→B, SC-002/SC-005→F, SC-003→C/oracle,
  SC-004→all skip+report paths.

## Remaining uncertainties (flagged, non-blocking)

- Exact lexical extraction of the success type `T` from `(T, error)` when `T` itself
  contains commas/brackets (e.g. `(map[string]int, error)`) — reuse the bracket-aware
  splitting `result.go`’s signature logic already does; covered by a fixture.
- Option `*o` escape analysis (DR-4) is conservative-by-design; acceptable to skip+report
  rather than over-engineer in MVP.
