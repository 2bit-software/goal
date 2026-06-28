# US-002 Value-position match over Result and Option — Business Specification

## Overview

A goal author can use a `match` over a Result or Option where the match's result
is consumed as a value — either returned directly or assigned to a binding — and
have it lower to valid Go. Today this only works for enum matches; a Result/Option
match in value position fails transpilation with `unsupported expression
*ast.MatchExpr`.

## Functional Requirements

### FR-1: Return-position Result match
`return match r { Result.Ok(v) => A; Result.Err(e) => B }`, where r is a Result,
yields valid Go and does not fail with `unsupported expression *ast.MatchExpr`.

### FR-2: Assignment-position Option match
`x := match o { Option.Some(v) => A; Option.None => B }`, where o is an Option,
yields valid Go in which x is bound to the arm result.

### FR-3: Both arms emitted
Both the Ok/Some payload arm and the Err/None arm are reachable in the generated
Go — each arm's body is emitted.

### FR-4: No regression
Statement-position match over Result/Option, and value-position match over enums,
keep their existing lowering and tests unchanged.

## Acceptance Criteria

- [ ] Transpiling a return-position Result match yields valid Go, no
  `unsupported expression *ast.MatchExpr`.
- [ ] Transpiling an assignment-position Option match yields valid Go binding x.
- [ ] Both arm bodies appear in the generated Go for each case.
- [ ] Existing statement-position Result/Option and enum value-match tests pass.
- [ ] A backend test covers value-position match over both Result and Option, in
  both return and assignment positions, and passes.

## User Interactions

Goal source authored by the developer; surfaced through `goal build` / the AST
backend `Transpile` entry. No CLI change.

## Error Handling

A value-position Result/Option assignment whose arm bodies have no inferable
result type keeps the existing located deferral message (annotate as
`var x T = match …` or use `return match …`) — consistent with the enum path. No
new diagnostics.

## Out of Scope

- Enum value-match (already lowers).
- Statement-position Result/Option match lowering shapes (unchanged).
- Inferring arbitrary arm-body result types beyond the existing string/bool/enum
  set for `:=` assignment.

## Open Questions

None — the lowering mirrors the already-landed enum value-position path.
