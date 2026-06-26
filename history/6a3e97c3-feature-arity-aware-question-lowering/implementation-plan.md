# Implementation Plan: Arity-Aware `?` Lowering

Companion to `technical-spec.md`. Steps are ordered bottom-up (data → producers → consumer →
tests) so each layer compiles against the one below it.

## Step order

### S1 — `FuncSig.Arity` + in-file arity (`internal/analyze/analyze.go`)
- Add `Arity int` to `FuncSig`.
- Add `countReturns(ret string) int` (reuses `splitTopLevel`).
- Restructure `analyzeSig` to compute `Arity` *before* the Result/Option guard, slicing the
  return clause from the **true param-list close** `scan.MatchParen(toks, f.NameTok+1)` — NOT
  `f.ParamsClose` (which points at a parenthesized return type's own `)`; see technical-spec C1).
  Override to 2 for `ModeResult`, 1 for `ModeOption`/`ModeResultClosed`.
- **Write this unit test FIRST** (it catches the slice bug) — `analyze_test.go`:
  `func f() error` → 1, `func f() (int, error)` → 2,
  `func f() (a, b int, err error)` → 3, `func f()` → 0, `Result[T,error]` → 2,
  `Option[T]` → 1. (FR-002)

### S2 — `scan.CalleeKey` (`internal/scan/scan.go`)
- Add `CalleeKey(expr string) string` (leading ident, optional `.ident`, stop at `(`; "" for
  non-simple-call / deeper chains / method-on-value).
- **Unit test** (`scan_test.go`): `"os.MkdirAll(p)"` → `"os.MkdirAll"`, `"doThing(x)"` →
  `"doThing"`, `"f.Close()"` → `"f.Close"`, `"xs[0]"` → `""`, `"(a+b)"` → `""`.

### S3 — Foreign func arity (`internal/analyze/foreign.go`)
- Refactor `foreignStructs` → `foreignDecls` returning `funcs map[string]int` too (single parse
  pass); update its caller.
- Add `resultArity(*ast.FuncType) int`; collect exported, receiver-less funcs keyed
  `alias.Func`.
- Add `questionCalleeAliases(srcs)` and union into `needed` in `EnrichForeign`; merge funcs into
  `t.FuncSignatures` with `Mode` left zero (FR-010).
- **Unit test** (`foreign_test.go`): via the injected `DirResolver` against a fixture dir,
  assert `t.FuncSignatures["ext.Mkdir"].Arity == 1` and a `(T, error)` func → 2; assert an
  import referenced *only* by `?` is still loaded; assert `Mode == ModeNone`. (FR-003, FR-007,
  FR-010)
- **Fixture**: add exported funcs to `internal/analyze/testdata/extpkg/…` and
  `internal/pipeline/testdata/extpkg/types.go` (e.g. `func Mkdir(p string) error`,
  `func Open(p string) (int, error)`).

### S4 — Arity-aware lowering (`internal/pass/question.go`)
- Add `calleeArity(t, rhs)`; in the `ModeResult` branch emit `arity−1` blanks for discard, with
  the unknown→2 default (FR-005); emit the FR-009 diagnostic for a resolved non-discard arity≠2.
- Guard `strings.Repeat` with `n >= 1`.

### S5 — Golden + compile + regression tests
- **Goldens** under `features/05-question-prop/examples/`: a new error-only discard example
  (in-file `func clean() error` + `clean()?`) with its 1-value `.go.expected`; a multi-return
  discard if practical. Confirm `qprop_discard`/`qprop_result`/binding goldens are unchanged
  (FR-006).
- **Compile proof** (`internal/pipeline/pipeline_package_test.go` style): a package-mode case
  importing the fixture pkg and calling `ext.Mkdir(p)?`, run through `go build ./...` (SC-001).
- **Fallback** (FR-005): single-file unresolved foreign discard → two-value form unchanged.
- **Diagnostic** (FR-009): `x := clean()?` over an error-only callee → expected error.

### S6 — Full suite + manual sanity
- `task` test entry (or `go test -count=1 ./...`); confirm zero regressions across analyze /
  pass / pipeline / check suites.

## Reuse Audit (inline)

Searched the codebase for existing implementations before planning new code:

| Planned item | Finding | Decision |
|---|---|---|
| Count top-level return entries | `analyze.splitTopLevel` already splits comma lists at delimiter depth 0 | **Reuse** inside `countReturns` |
| Arity reasoning ("sole output is the error") | `internal/fix/propagate.go` already does this in the Go→goal direction | **Reference** for correctness; no shared code (different direction/representation) |
| Foreign Go parsing (`go/ast` walk over package dir) | `foreignStructs` already parses the dir's `.go` files | **Extend** — add func collection to the same parse pass (refactor to `foreignDecls`) |
| Import path / dir resolution | `DefaultResolver`, `ParseImports`, `lastSegment` exist | **Reuse** as-is |
| Paren matching | `scan.MatchPair`/`MatchParen` forward-only; no back-match | **Create** `CalleeKey` (string-based) instead of a back-matching token helper |
| `?` callee name extraction | `scan.LeadIdent` exists but drops the qualifier | **Create** `scan.CalleeKey` (superset; LeadIdent unchanged) |
| Name constants `__goal_ok`/`__goal_err` | `internal/pass/pass.go:23` | **Reuse** |
| `?` regression lock | golden `qprop_*.go.expected` + `go build` package test | **Reuse** harnesses; add new goldens |

No DUPLICATE creations. One OVERLAP resolved by **Extend** (`foreignStructs`→`foreignDecls`).
Two **Create new** (`CalleeKey`, `countReturns`) justified above (no back-match helper; no
qualifier-aware extractor).

## Risks & mitigations

- **`analyzeSig` early-return drops single-return arity** → compute arity before the guard (S1).
- **`f.ParamsClose` is the return-type close for parenthesized returns** → slice the return
  clause from `scan.MatchParen(toks, f.NameTok+1)` instead; S1 test (written first) is the
  guard against this (verified CRITICAL in plan audit).
- **`neededAliases` empty short-circuit skips `?`-only imports** → union `questionCalleeAliases`
  (S3, FR-007).
- **Negative `strings.Repeat`** on a void/arity-0 callee → `n >= 1` guard falls back to 2 (S4).
- **Foreign key collision with in-file func** → impossible (foreign keys contain `.`).
- **Goldens silently rewritten** → verify the three existing `qprop_*` expecteds are untouched
  in S5 before adding new ones.

## Out of scope (per spec)

- Method-call / non-call `?` callees (unresolved → two-value default).
- Error-only foreign `?` in single-file mode (foreign-blind; package-mode-only limitation).
- Type-checking that an arity-1 callee's sole return is actually `error`.
