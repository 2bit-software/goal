# Tasks: Arity-Aware `?` Lowering

Traces to `implementation-plan.md` (S1–S6) and `technical-spec.md`. Complexity: **Medium**
(4 source files + 2 fixtures + tests). Single list (< 20 tasks). TDD where a test pins a bug
(T001, T004).

## Dependency order & parallelism

```
T001 ─┬─ T002 ───────────────┐
T003 ─┘                       ├─ T011 ─ T012 ─ T013
T004 ─ T005 ─ T006 ─ T007 ────┘   (golden) (compile) (suite)
T008 ─ T009 (scan, parallel to analyze/foreign)
```
Critical path: T004 → T005 → T007 → T011 → T012 → T013.
`[P]` tasks touch disjoint files and may run concurrently.

## Layer 1 — in-file arity (analyze)

- [ ] **T001** [P] [US1/US2] Write FIRST (red): in `internal/analyze/analyze_test.go` add a
  table test for the forthcoming `Arity`: `func f() error`→1, `func f() (int, error)`→2,
  `func f() (a, b int, err error)`→3, `func f() (error)`→1, `func f()`→0,
  `func f() Result[int, error]`→2, `func f() Option[int]`→1. (FR-002, SC-003)
- [ ] **T002** [US1/US2] In `internal/analyze/analyze.go`: add `Arity int` to `FuncSig`; add
  `countReturns(ret string) int` (reuse `splitTopLevel`); restructure `analyzeSig` to slice the
  return clause from `scan.MatchParen(toks, f.NameTok+1)` (NOT `f.ParamsClose`), set arity
  before the Result/Option guard, override per mode (Result→2, Option/Closed→1). Make T001 pass.

## Layer 2 — callee-key helper (scan) — parallel to Layer 1/3

- [ ] **T003** [P] Write FIRST: in `internal/scan/scan_test.go` add `CalleeKey` cases from the
  technical-spec S2 table (`os.MkdirAll(p)`→`os.MkdirAll`, `doThing(x)`→`doThing`,
  `f.Close()`→`f.Close`, `f[T](x)`→`f`, `pkg.Sub.Func(x)`→`pkg.Sub`, `xs[0]`→``, `(a+b)`→``,
  leading whitespace). 
- [ ] **T004** Add `scan.CalleeKey(expr string) string` to `internal/scan/scan.go` (leading
  ident; optional `.ident`; stop at first `[`/`(`). Make T003 pass. Leave `LeadIdent` untouched.

## Layer 3 — foreign func arity (analyze/foreign)

- [ ] **T005** [P] [US1] Add an exported func to each fixture: `func Mkdir(p string) error` and
  `func Open(p string) (int, error)` (and a 3-return `func Triple() (int, int, error)`) to
  `internal/analyze/testdata/extpkg/…` and `internal/pipeline/testdata/extpkg/types.go`.
- [ ] **T006** [US1] In `internal/analyze/foreign.go`: refactor `foreignStructs`→`foreignDecls`
  returning `funcs map[string]int` (single parse pass; update caller); add
  `resultArity(*ast.FuncType) int`; collect exported, receiver-less funcs keyed `alias.Func`.
- [ ] **T007** [US1/FR-007] In `EnrichForeign`: add `questionCalleeAliases(srcs)` (reuses
  `CalleeKey` on each `?`'s rhs) and union into `needed` **before** the `len(needed)==0`
  short-circuit; merge funcs into `t.FuncSignatures` as `FuncSig{Arity: n}` (Mode left zero).
  Depends on T004, T006.
- [ ] **T008** [US1/FR-003/FR-007/FR-010] In `internal/analyze/foreign_test.go`: via injected
  `DirResolver`/fixture, assert `FuncSignatures["ext.Mkdir"].Arity==1`, `…["ext.Open"].Arity==2`,
  an import referenced **only** by `?` is still loaded, and foreign entries have
  `Mode==ModeNone`. Depends on T005, T007.

## Layer 4 — arity-aware lowering (pass)

- [ ] **T009** [US1/US2/US3/FR-001/FR-005/FR-009] In `internal/pass/question.go` `ModeResult`
  branch: add `calleeArity(t, rhs)`; discard form emits `strings.Repeat("_, ", n-1)` with
  unknown→2 default and `n>=1` guard; non-discard form emits the FR-009 diagnostic when a
  resolved callee has arity≠2. Depends on T002, T004.

## Layer 5 — golden / compile / fallback / diagnostic tests

- [ ] **T010** [P/FR-006] Confirm `features/05-question-prop/examples/qprop_discard.go.expected`
  (and `qprop_result`, binding form) are **byte-for-byte unchanged** after T009. Depends on T009.
- [ ] **T011** [US1] Add a golden under `features/05-question-prop/examples/`: in-file
  `func clean() error` + `clean()?` → 1-value `if __goal_err := clean(); …` `.go.expected`;
  optionally a multi-return discard. Depends on T009.
- [ ] **T012** [US1/SC-001] Add a package-mode test in
  `internal/pipeline/pipeline_package_test.go` style importing the fixture pkg and calling
  `ext.Mkdir(p)?`; run the emitted package through `go build ./...`. Depends on T005, T007, T009.
- [ ] **T013** [US3/FR-005] Single-file unresolved foreign discard (bare and `_ :=`) → two-value
  form unchanged. [FR-009] `x := clean()?` over error-only callee → expected diagnostic.
  Depends on T009.

## Layer 6 — full suite

- [ ] **T014** Run the full suite (`task` test entry or `go test -count=1 ./...`); confirm zero
  regressions across analyze / scan / pass / pipeline / check. Depends on all above.

## Requirement → task coverage

| Req | Tasks |
|---|---|
| FR-001 | T009, T011 |
| FR-002 | T001, T002 |
| FR-003 | T006, T008 |
| FR-004 | T004, T009 |
| FR-005 | T009, T013 |
| FR-006 | T010 |
| FR-007 | T007, T008 |
| FR-008 | T012 (single-file path untouched), T013 |
| FR-009 | T009, T013 |
| FR-010 | T007, T008 |
| SC-001 | T012 |
| SC-003 | T001, T011 |

No orphan tasks; every task traces to an S-step.
