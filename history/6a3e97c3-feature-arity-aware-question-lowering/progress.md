# Progress Log — Arity-Aware `?` Lowering

All 14 tasks complete. `task check` (vet + full suite) green.

### T001/T002 — in-file arity (analyze) — Complete
- `internal/analyze/analyze.go`: `FuncSig.Arity`; `countReturns`; `inFileArity` (slices the
  return clause from the true param close `MatchParen(toks, NameTok+1)`, dodging the
  `ParamsClose`-on-parenthesized-return bug); `analyzeSig` sets arity before the Result/Option
  guard and overrides per mode.
- `internal/analyze/analyze_test.go`: `TestFuncSigArity` (error→1, (int,error)→2, named-group→3,
  (error)→1, void→0, Result→2, Option→1). Passing.

### T003/T004 — `scan.CalleeKey` — Complete
- `internal/scan/scan.go`: `CalleeKey` (leading optionally-qualified ident; generics → base).
- `internal/scan/scan_test.go` (new): `TestCalleeKey`. Passing.

### T005/T006/T007/T008 — foreign func arity (analyze/foreign) — Complete
- Fixtures: added `Mkdir`/`Open`/`Triple` exported funcs + unexported `hidden` + a method
  `Close` to `internal/analyze/testdata/extpkg/types.go`; `Mkdir` to
  `internal/pipeline/testdata/extpkg/types.go`.
- `internal/analyze/foreign.go`: `foreignStructs`→`foreignDecls` (also returns `funcs`);
  `resultArity`; `questionCalleeAliases` unioned into `needed` before the empty short-circuit;
  funcs merged into `FuncSignatures` with `Mode` left zero.
- `internal/analyze/foreign_test.go`: `TestEnrichForeignRecordsFuncArity` — arities 1/2/3,
  `?`-only import still loaded, `Mode==ModeNone`, unexported + method excluded. Passing.

### T009 — arity-aware lowering (pass) — Complete
- `internal/pass/question.go`: `calleeArity`; `ModeResult` discard emits `Repeat("_, ", n-1)`
  (unknown→2, `n>=1` guarded); non-discard emits the FR-009 diagnostic for resolved arity≠2.

### T010/T011/T012/T013 — golden / compile / fallback / diagnostic — Complete
- T010: existing `qprop_discard`/`qprop_result` goldens unchanged (FR-006) — regression suite green.
- T011: new golden `features/05-question-prop/examples/qprop_erronly.{goal,go.expected}` (in-file
  `clean()?` → `if __goal_err := clean(); …`).
- T012: `internal/pipeline/question_arity_test.go` — `TestPackageForeignErrorOnlyQuestion`
  (foreign one-value guard via real driver) + `TestErrorOnlyQuestionCompiles` (`go build`, SC-001).
- T013: `TestSingleFileForeignQuestionFallback` (FR-005 two-value, no regression) +
  `TestBindingErrorOnlyCalleeDiagnostic` (FR-009).

### T014 — full suite — Complete
- `task check`: `go vet ./...` clean; all packages pass.

### Notes
- Pre-existing working-tree changes (the bare `expr?` statement form in
  check/closed.go, pass/closed.go, pass/question.go, scan.go) are the foundation this feature
  completes; they ship in the same commit.
- No blockers; no spec revisions needed during implementation (the plan-audit C1 fix held).
