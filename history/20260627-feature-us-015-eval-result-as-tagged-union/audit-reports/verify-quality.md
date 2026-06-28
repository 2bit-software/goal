# Verify — Quality (US-015)

## Error handling
- Unknown Result constructor and wrong arity are located (`call.Pos()`),
  descriptive errors, asserted by tests. Consistent with the host-bridge refusal
  style (US-011).
- The non-exhaustive match default remains the loud `unreachable` panic (unchanged
  shared seam); a Result match over Ok/Err is exhaustive so it is not normally hit.

## Edge cases
- `payloadValue` returns ok=false for a variant with other than one field, so a
  hypothetical data-less variant (Option.None, US-016) will bind nothing rather
  than mis-unwrap — armScopeFor falls back to the whole variant safely.
- The Result unwrap is keyed strictly on `TypeID == "Result"`, so enum bindings
  (`Event.Login(l) => l.user`) are provably untouched — verified by the existing
  enum/match tests still passing.

## Tests assert real behavior
- Construction tests inspect the actual Variant TypeID/Tag/payload, not just
  non-nil. Match tests assert concrete output strings ("hello", "empty input",
  "empty", "q"), exercising the unwrap in both arms and at nested depth.

## Findings
No CRITICAL, MAJOR, or MINOR findings. Implementation matches the spec.
