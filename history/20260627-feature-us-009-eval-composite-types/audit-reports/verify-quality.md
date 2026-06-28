# Verify — Quality — US-009

## Checks
- Error handling: every refusal is a located `fmt.Errorf("interp: ...")` —
  out-of-range index, non-string map key, field-on-non-struct, index-non-
  collection, nil-map write, compound-on-absent-key, `:=` on an index/field
  target, positional struct literal. None are silent nils or panics.
- Edge cases tested: continue/break inside range, compound assignment through
  slice element and struct field, map-range determinism (sorted keys).
- Tests assert real behavior (concrete arithmetic/string results), not just
  "no error".
- Reference semantics correct: `StructValue`/`MapValue` are pointer-backed and
  `[]Value` shares its backing array, so element/field/entry writes are visible
  through the looked-up binding (exercised by the *Assign tests).

No CRITICAL/MAJOR findings.

- **MINOR**: arrays (`[N]E`) fold into the slice path; non-string map keys and
  positional struct literals are explicit, descriptive refusals (deferred by
  the spec), not silent gaps.

## Assumptions
- Same as verify-acceptance.md. internal/interp stays dependency-clean (US-022).
