# Plan Coverage Audit — US-017

## Requirement -> plan trace

| Spec item | Plan element |
|-----------|--------------|
| FR-1 Ok/Some unwrap-and-continue | `evalUnwrap` Ok/Some -> `payloadValue`; tests `TestQuestionResultOkContinues`, `TestQuestionOptionSomeContinues` |
| FR-2 Err early return | `propagateErr` -> `returnSignal{[Result.Err]}`; `TestQuestionResultErrEarlyReturns` |
| FR-3 None early return | `propagateNone` -> `returnSignal{[Option.None]}`; `TestQuestionOptionNoneEarlyReturns` |
| FR-4 closed-E `from` conversion | `propagateErr` ModeResultClosed branch + `calleeErrType` + FromRegistry + callFunc on conversion; `TestQuestionClosedEAppliesFromConversion` |
| FR-5 refusal outside Result/Option | `propagateErr`/`propagateNone` none-sig branch; `TestQuestionOutsidePropagatingFunctionRefused` |
| AC same-E no conversion | ModeResultClosed `calleeE == sig.E` skip; `TestQuestionClosedESameENoConversion` |
| EH non-variant operand | `evalUnwrap` default refusal; `TestQuestionOnNonVariantRefused` |

No plan element lacks a requirement (no scope creep). Every acceptance criterion
has a named test.

## Findings

None CRITICAL/MAJOR. Coverage complete.

## Assumptions

- Test functions are parameterless so they reuse the existing `evalFn` helper
  (which calls a function by name with no args and asserts a single return value).
