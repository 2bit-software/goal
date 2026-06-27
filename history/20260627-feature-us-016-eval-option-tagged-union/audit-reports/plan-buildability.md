# Plan Audit — Buildability

- Dependency order valid: constants (value.go) before their use (eval.go);
  ctor before its interception caller (same file, Go resolves order-free).
- Interface contracts agree: `VariantVal(typeID, tag string, fields map) Value`
  and `payloadValue(*Variant)(Value,bool)` already exist with the needed
  signatures; `armScopeFor` change is a boolean-condition widening only.
- File paths verified against the existing tree (internal/interp/{value,eval,
  interp}.go all exist; option_test.go does not yet — no conflict).
- Integration points are specific: exact functions named (evalCallMulti,
  evalSelector, armScopeFor) with the exact guard shape to mirror (Result block).
- Compiles at each step: adding constants is inert; adding evalOptionCtor +
  interception is additive; armScopeFor edit is local.

No CRITICAL/MAJOR findings.

## Assumptions

- No new package imports required, preserving the US-022 dependency envelope.
