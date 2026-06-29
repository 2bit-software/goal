# Verify — Quality — SEAM-003

## Code quality

- Conversion follows the proven SEAM-002 idiom: `enum` + qualified `Enum.Variant`
  references + value-position `match` bound to bools for guards.
- Exhaustiveness: matches over always-set values use full variant enumeration; matches
  over possibly-zero values (`csig`/`sig` when `!known`/`!ok`) deliberately use a nil-safe
  `_` rest-arm, documented inline at each site.
- The `resolve.goal` arity/ends-in-error switch was split into two value-position matches
  rather than a statement match with an empty `ModeNone => {}` arm (empty match arms are
  the documented SEAM-002 anti-pattern).
- `SeverityLabel` is a legitimate production function (the enum cannot carry a String
  method), not a test-only shim; its `internal/sema` mirror is the only added internal
  symbol, justified inline for port-gate compatibility.

## Test integrity

- No test was weakened to pass: the shared sema/typecheck tests still assert Severity via
  `SeverityLabel(...)` (full coverage, both representations). Mode assertions were relocated
  intact to internal-only `mode_test.go`, not deleted.
- The behavioral port gates transpile selfhost/{sema,backend,typecheck} as enums and run
  the rewritten/relocated tests against them — real coverage of the converted code.

## Findings

No CRITICAL or MAJOR. MINOR: two enum-classification strategies coexist (SeverityLabel
helper vs Mode relocation); justified by whether the type has a production string form, and
documented in DECISIONS.md.

## Assumptions

- Adding a mirrored `SeverityLabel` to internal/sema (Go) is acceptable; it is a thin
  Stringer wrapper used by white-box tests, exported, never flagged unused.
