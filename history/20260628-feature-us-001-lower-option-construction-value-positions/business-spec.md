# Lower Option construction in value positions — Business Specification

## Overview

goal's `Option.Some(v)` and `Option.None` are sugar that the Go backend lowers to
the `*T` pointer encoding. Today the lowering only fires at a direct `return` or as
a `Result.Ok(...)` payload; in any other position the construction is emitted
verbatim, leaving an undefined `Option.` reference and Go that fails to compile.
This feature generalizes the lowering so an Option value composes wherever it is
produced.

## Functional Requirements

### FR-1: Option construction lowers in var/assignment position
`x := Option.Some(v)` (and `var x = Option.Some(v)`) SHALL bind `x` to the `*T`
pointer form, with no literal `Option.` token in the generated Go.

### FR-2: Option construction lowers in every value position
`Option.Some(x)` and `Option.None` used as a call argument, a struct-literal field
value, a slice-literal element, and a map-literal element SHALL each yield valid Go
(parsing under go/format) with no literal `Option.` token.

### FR-3: Stable pointer encoding
`Option.None` SHALL lower to `nil`. `Option.Some(x)` SHALL lower to the `*T`
pointer encoding: `&x` for an addressable argument, otherwise a boxed temporary
that remains a valid Go expression in its position.

### FR-4: No regression of existing Option lowering
The existing direct-return and Result.Ok-payload Option lowering SHALL keep passing
its current tests unchanged.

## Acceptance Criteria

- [ ] Transpiling `x := Option.Some(v)` (v a local identifier) yields valid Go in
      which x is bound to the *T pointer form, with no literal `Option.` token.
- [ ] Transpiling Option.Some(x)/Option.None as a call argument, a struct-literal
      field value, and a slice- and map-literal element each yields valid Go (parses
      under go/format) with no literal `Option.` token.
- [ ] Option.None lowers to `nil` and Option.Some(x) lowers to the *T pointer
      encoding (`&x` for an addressable argument, otherwise a boxed temporary) in
      every position above.
- [ ] The existing direct-return and Result.Ok-payload Option lowering still passes
      its current tests unchanged.
- [ ] A backend test exercising Option construction in var-assignment,
      call-argument, struct-field, and slice/map-literal positions exists and passes.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

Authors write goal source; `goal build` / `backend.Transpile` emits Go. No new CLI
surface. The interaction is purely the broadened set of source positions where
`Option.Some`/`Option.None` is accepted and lowered.

## Error Handling

A non-Option selector or call is unaffected and emits unchanged. Generated Go must
remain `go vet` clean; an Option construction that cannot be lowered stays an
existing backend error rather than emitting an undefined `Option.` reference.

## Out of Scope

- Value-position `match` over Result/Option (US-002).
- `?` on method calls (US-003).
- Changing the existing return / Result.Ok lowering encoding.

## Open Questions

- None. The mechanism is pinned by the prd notes (mirror `optionValueExpr`).
