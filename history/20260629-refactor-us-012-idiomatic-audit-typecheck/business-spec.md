# US-012 Idiomatic audit: typecheck — Business Specification

## Overview

`selfhost/typecheck` is the depth-checker harness of the self-hosted goal compiler. This
story audits it (step 3 of the SELF-HOST IDIOMATIC plan) so it reads as idiomatic goal
rather than transpiled Go: fallible helpers should use Result/Option + `?` where natural,
and in-file enum switches should become `match` where they fit. Where a goal idiom cannot
be applied without changing behavior or a public oracle-pinned signature, the decision to
keep the Go construct is recorded in DECISIONS.md rather than force-fit.

## Functional Requirements

### FR-1: Idiomatic fallible functions
Genuinely-fallible package-internal helpers express failure with Result/Option and `?`
propagation wherever that conversion is behavior-preserving and does not change a public
signature pinned by an oracle test.

### FR-2: No remaining auto-convertible propagation
`goal fix` over the package produces no source diff — there are no remaining
auto-convertible `(T,error)`+manual-`if err` propagation sites it would mechanically rewrite.

### FR-3: Documented refusals
Every fallible function or enum-shaped construct that is deliberately left as Go is recorded
in DECISIONS.md with a concrete reason.

### FR-4: Behavioral equivalence
The transpiled package passes the typecheck depth tests, and the whole-compiler bootstrap
stays byte-identical (`task fixpoint`).

## Acceptance Criteria

- [ ] Fallible depth-check functions use Result/Option with `?` where natural (or the
      non-applicability is recorded as a refusal with reason).
- [ ] `goal fix selfhost/typecheck/*.goal` reports no remaining auto-convertible propagation
      sites (no source diff).
- [ ] typecheck depth tests pass against the transpiled package.
- [ ] `task check`, `task build`, and `task fixpoint` are all green.

## User Interactions

None directly — this is a compiler-internal package. The user-visible surface is the depth
diagnostics it produces, which must remain identical.

## Error Handling

Error behavior must be preserved exactly. `Load` continues to wrap transpile/parse failures
with context (`fmt.Errorf("...: %w", err)`); `Check` continues to surface those failures
through the TypeChecker interface unchanged.

## Out of Scope

- Cross-package switch->match coordination (belongs to the whole-tree US-013 sweep).
- Changing any public signature an oracle test pins (`Load`, `TypeChecker.Check`).
- Sealing/enum-ifying ordered iota ints consumed via `==` / numeric literals.

## Open Questions

None. The package survey is conclusive: the only error-returning functions are exported,
oracle-pinned `Load` and the interface-pinned `Check` method, both with non-`?`-eligible
propagation; the lone iota type (`litClass`) does not fit `enum`.
