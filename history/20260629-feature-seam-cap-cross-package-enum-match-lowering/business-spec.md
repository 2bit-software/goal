# Cross-package enum-match lowering — Business Specification

## Overview

The goal compiler lowers a `match` over an enum to a Go type-switch over the
§8.1 closed-sum encoding. Today this works only when the enum is defined in the
same package as the `match` (even across files). When the enum is defined in an
imported package, lowering fails. This feature makes the backend lower a `match`
over an enum defined in an imported package, so cross-package enum consumers
(the SEAM-002/003/004 conversions) compile and behave identically to the
equivalent hand-written switch.

## Functional Requirements

### FR-1: Qualified variant patterns are recognized
A `match` whose arms name `pkg.Enum.Variant` (the enum imported from `pkg`) is
recognized as an enum match — not rejected as an unsupported match.

### FR-2: Imported enums resolve
The compiler resolves the definition of an enum declared in an imported package
when classifying and lowering a cross-package `match`, the same way it already
resolves imported struct field sets. This is not limited to the built-in
Result/Option types.

### FR-3: Correct lowering
A cross-package enum `match` lowers to a Go type-switch over the imported sum
encoding (`case pkg.Enum_Variant:`), behaviorally identical to a hand-written
switch over the same imported types.

### FR-4: No regression to existing matches
Same-package enum matches, and Result/Option matches, continue to lower exactly
as before.

## Acceptance Criteria

- [ ] A package-mode program that `match`es over an enum imported from another
      package transpiles without error (no `unsupported statement-position match`).
- [ ] The transpiled package compiles and links (`go build`).
- [ ] The lowered switch produces the same runtime result as the equivalent
      hand-written Go type-switch over the imported variant types.
- [ ] Same-package enum, Result, and Option matches are unchanged.
- [ ] `task check`, `task build`, `task fixpoint` are green; the corpus
      behavioral tier is unchanged.

## User Interactions

A goal author writes, in package B:

    import a "module/pkg/a"

    func f(x a.Mode) string {
        return match x {
            a.Mode.On  => "on"
            a.Mode.Off => "off"
        }
    }

and the compiler lowers it without requiring the enum to be re-declared locally.

## Error Handling

- An unknown/unsupported match qualifier still yields the descriptive
  `unsupported statement-position match` error (behavior preserved for genuinely
  unsupported cases).
- An import that cannot be resolved leaves its enums unknown (non-fatal), exactly
  as foreign struct resolution already degrades.

## Out of Scope

- Value-position `:=` match type inference across packages beyond what the
  existing `inferMatchType` already supports (statement/return/var positions are
  the focus).
- Converting any real iota types to enums (that is SEAM-002/003/004); this story
  delivers only the backend capability plus a regression fixture.
- Payload-bearing cross-package variant field-set reconstruction beyond what the
  fixture needs (the real seam targets are tag-only enums).

## Open Questions

- None blocking. The reconstruction of foreign enums from their generated §8.1
  encoding is the chosen mechanism; it mirrors the existing foreign-struct
  enrichment seam.
