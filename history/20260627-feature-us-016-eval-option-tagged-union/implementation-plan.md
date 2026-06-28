# Implementation Plan — US-016 Eval Option as tagged union

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| internal/interp/option_test.go | Unit tests over a 04-option shape: Some/None construction + match arms. |

### Modified Files
| File | Change |
|------|--------|
| internal/interp/value.go | Add Option constants: `optionTypeID="Option"`, `optionSomeTag="Some"`, `optionNoneTag="None"`, `optionSomeField="value"`. |
| internal/interp/eval.go | Add `evalOptionCtor`; intercept `Option.Some(x)` in `evalCallMulti` (receiver-name guard + not-shadowed); intercept `Option.None` in `evalSelector`. |
| internal/interp/interp.go | Extend `armScopeFor` unwrap guard to fire for `optionTypeID` as well as `resultTypeID`. |

## Implementation Steps

1. value.go: add the four Option constants next to the Result constants. Update
   the `payloadValue` doc comment (already mentions Option.Some) if needed.
2. eval.go `evalOptionCtor`: mirror `evalResultCtor` — `Some` requires exactly 1
   arg -> `VariantVal("Option","Some",{value:x})`; an unknown ctor or wrong arity
   is a located refusal. (Note: `None` is data-less and handled in evalSelector,
   so evalOptionCtor only needs the `Some` case + default refusal.)
3. eval.go `evalCallMulti`: add an interception block (after the Result block,
   before host) for a selector call whose receiver Ident is `optionTypeID` and
   not shadowed -> `evalOptionCtor`.
4. eval.go `evalSelector`: after the enum data-less guard, add a guard for a
   receiver Ident `optionTypeID` (not shadowed) with Sel `None` ->
   `VariantVal("Option","None",nil)`.
5. interp.go `armScopeFor`: change the unwrap condition to
   `TypeID == resultTypeID || TypeID == optionTypeID`.
6. option_test.go: construct Some/None via parsed program + `newInterp`; assert
   tag/payload; drive a `match` (04-option `exists`-shaped) asserting Some/None
   arms. Reuse existing helpers (`newInterp`, `call`, `intLit`). stdlib testing.

## Reuse

- `VariantVal`, `payloadValue` (value.go) — unchanged, reused.
- `evalResultCtor` shape (eval.go) — template for `evalOptionCtor`.
- `selectMatchArm` (interp.go) — unchanged; dispatches Some/None by tag for free.
- Test seam: `newInterp` + `evalExpr(call(...))` (call_test.go / result_test.go).

## Risks

- Low. Mechanical mirror of the shipped US-015 Result code. Must keep
  `internal/interp` free of go/types deps (US-022 gate) — no new imports needed.
