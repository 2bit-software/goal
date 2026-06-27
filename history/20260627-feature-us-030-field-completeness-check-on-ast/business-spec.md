# Field-completeness check on the AST (US-030) — Business Specification

## Overview

goal guarantees there are no silent zero values (spec §8): a struct or enum
variant must be constructed with every field named, or it must explicitly opt
out by spreading defaults. The legacy checker enforced this by scanning a flat
token stream and reconstructing literal/block/pattern shape with brace
heuristics. This feature reimplements the same guarantee over the parsed AST so
the diagnostic survives the front-end change and is correct by construction.

## Functional Requirements

### FR-1: Struct-literal completeness
A composite literal of a struct declared in the file SHALL be reported as an
error when it omits one or more declared fields, naming the omitted field(s).

### FR-2: Variant-construction completeness
An enum variant construction `Enum.Variant(field: value, …)` SHALL be reported
as an error when it omits one or more of that variant's declared fields. A
data-less variant is trivially complete. Variants have no `...defaults` escape.

### FR-3: Spread opt-out
A struct literal that includes a `...defaults` (§8.5) or `...derive(src)` (§12)
spread SHALL be treated as complete by construction and produce no diagnostic.

### FR-4: Deferral on unresolved type
A keyed literal whose type is not declared in the file SHALL be deferred with a
located Warning ("field-completeness deferred"), never assumed complete and
never reported as an error.

### FR-5: Match-binding is not construction
A `Enum.Variant(binding)` appearing as a match-arm pattern SHALL NOT be treated
as a construction and SHALL NOT be checked for field completeness.

### FR-6: Diagnostic parity
Diagnostics SHALL carry the same severities and messages as the legacy check so
the corpus's inline `// want` markers are satisfied unchanged.

## Acceptance Criteria

- [ ] sema implements the field-completeness check over the AST (deferrals as
      Warnings, violations as Errors).
- [ ] A struct literal omitting a single field reports `omits required field
      `<name>``.
- [ ] A struct literal omitting multiple fields reports `omits required fields
      `<a>`, `<b>`` in declaration order.
- [ ] A variant construction omitting a field reports `omits required field
      `<name>``.
- [ ] A keyed literal of an undeclared type reports a Warning containing
      `field-completeness deferred` and no Error.
- [ ] A literal completed by `...defaults` or `...derive(src)` produces no Error.
- [ ] A complete struct literal (including a tagged struct) and a data-less
      variant produce no Error.
- [ ] A match-arm payload binding produces no Error.
- [ ] Every case in testdata/check/08-no-zero-value passes through the sema
      checker via the corpus runner.

## User Interactions

Developers run `goal check`; the field-completeness diagnostics are emitted by
the AST-based checker. In this story the behavior is exercised through the
corpus check runner against the 08-no-zero-value golden cases.

## Error Handling

- Omission without a completing spread → Error (rejects the program).
- Unresolved literal type → Warning (advisory deferral, never a false reject).

## Out of Scope

- must-use, implements, and ?-arity/refusal checks (US-031).
- Lowering/emitting `...defaults` or `...derive` (US-038/US-039) — this story
  only *recognizes* the spreads as opt-outs.
- Qualified (`pkg.T{…}`) and generic struct-literal completeness — not present
  in the corpus and deferred to a later pass.

## Open Questions

None — the behavior is pinned by 9 existing golden cases with inline markers.
