# Completeness Audit — US-017 (iteration 2)

Re-audit after the spec revision that added the "Test harness and oracle"
section and expanded Error Handling / Out of Scope.

## Resolution of iteration-1 findings

- **C1 (CRITICAL) — fixtures never reach Err/None/conversion branches.** RESOLVED.
  The spec now states tests use INLINE programs modeled on the 05/06 shapes but
  adapted so each branch fires (Err on sentinel input, None for missing key,
  closed-E callee that genuinely errs so `from` runs). The fixture files are
  explicitly NOT loaded verbatim.
- **M1 — `?` in methods would falsely refuse.** RESOLVED. Out of Scope now states
  method-position `?` pushes a none-shaped sig and hits the FR-5 refusal by
  design; method `?` propagation is out of scope.
- **M2 — "Open Questions: None" overclaimed; silent conversion gap.** RESOLVED.
  Error Handling now requires a located refusal when a closed-E `from` conversion
  cannot be resolved (never a silent mistyped Err); Open Questions updated.
- **M3 — `qprop_erronly` (bare error) contradiction.** RESOLVED. Out of Scope
  explicitly excludes the bare `func error` callee from asserted shapes
  (best-effort: nil continues).
- **M4 — operand not a Result/Option value.** RESOLVED. Error Handling adds a
  located refusal for a non-variant operand.
- **MINOR (empty stack / refusal format / mid-expression).** Addressed: empty/
  none-shaped enclosing sig is the FR-5 refusal; refusals carry the `interp:`
  prefix and tests assert on substrings.

## Verdict

No CRITICAL or MAJOR findings remain. Implementable from the spec.

## Assumptions

- Tests assert on `Value` shape (Kind / Variant.Tag / payload) via the existing
  `evalFn` helper, matching result_test.go / option_test.go.
- Refusal error strings are asserted by substring, not exact text.
