# Verification — US-017 Eval question-mark unwinding

## Gates (prd.json verifyCommands)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages, incl. internal/interp)
- US-022 envelope: `go list -deps goal/internal/interp | grep -E
  'go/types|internal/backend|internal/typecheck'` — empty (clean).

## Acceptance criteria -> evidence

| Criterion | Evidence (internal/interp/question_test.go) |
|-----------|---------------------------------------------|
| Ok yields unwrapped value, continues | `TestQuestionResultOkContinues` (Raw=="cfg!" proves the post-`?` statement consumed the unwrapped value) |
| Some yields unwrapped value, continues | `TestQuestionOptionSomeContinues` (Name=="ann-gp") |
| Err early-returns enclosing Result.Err | `TestQuestionResultErrEarlyReturns` (result is Err with the propagated message; the post-`?` Ok did not run) |
| None early-returns enclosing Option.None | `TestQuestionOptionNoneEarlyReturns` |
| Closed-E `from` conversion applied | `TestQuestionClosedEAppliesFromConversion` (ParseError.Empty -> AppError.Wrapped via `from func toApp`); `TestQuestionClosedEOkContinues` (Ok path, no conversion) |
| Closed-E same-E: no conversion | `TestQuestionClosedESameENoConversion` (Err payload is the unchanged ParseError.Empty) |
| `?` outside a Result/Option function refuses | `TestQuestionOutsidePropagatingFunctionRefused` |
| `?` on a non-Result/Option operand refuses | `TestQuestionOnNonVariantRefused` |

All acceptance criteria from prd.json US-017 are covered by an asserting test.

## Quality notes
- Stack balance under unwind: `fnStack` push/pop via `defer` in callFunc/callMethod
  so returnSignal / panicSignal / error all keep the stack balanced.
- FR-5 loudness: the enclosing-shape check runs UP FRONT in evalUnwrap (keyed on
  the operand TypeID), so a misplaced `?` refuses on the success path too.
- No silent failure: an unresolvable closed-E `from` conversion is a located
  refusal; a non-variant operand is a located refusal.

## Verdict
Implementation satisfies the spec. No CRITICAL/MAJOR findings.
