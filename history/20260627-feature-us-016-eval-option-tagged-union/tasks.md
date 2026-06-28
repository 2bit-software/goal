# Implementation Tasks

## Task 1: Add Option constants
**Status**: completed
- In internal/interp/value.go add `optionTypeID="Option"`, `optionSomeTag="Some"`,
  `optionNoneTag="None"`, `optionSomeField="value"` next to the Result constants.
- Acceptance: package compiles.

## Task 2: Construct Option.Some and Option.None
**Status**: completed
- In internal/interp/eval.go add `evalOptionCtor` (mirror `evalResultCtor`; only
  the `Some` ctor + default refusal).
- Intercept `Option.Some(x)` in `evalCallMulti` (receiver-name guard, not shadowed).
- Intercept `Option.None` in `evalSelector` -> `VariantVal("Option","None",nil)`.
- Acceptance: `Option.Some(1)` -> Variant Option/Some payload 1; `Option.None` ->
  Variant Option/None no payload.

## Task 3: Unwrap Option payload in match arms
**Status**: completed
- In internal/interp/interp.go widen `armScopeFor` unwrap guard to include
  `optionTypeID`.
- Acceptance: a Some arm binding reads the unwrapped inner value.

## Task 4: Tests over a 04-option shape
**Status**: completed
- New internal/interp/option_test.go: Some/None construction assertions + a
  `match` over Option (exists-shaped) asserting Some/None arms. stdlib testing.
- Acceptance: `go test ./internal/interp` green.

## Task 5: Verify gates
**Status**: completed
- Run prd.json verifyCommands: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`. All green.
