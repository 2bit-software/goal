# Verify — Quality

## Error handling / deferral discipline
Each check follows "defer, never guess": unresolved facts emit a located Warning, not
a false Error. Verified by the deferral fixtures (`deferred_qualified`,
`deferred_external_iface`, `defer_unknown_callee`, `defer_err_value`,
`defer_underscore_discard`) — all clean (no unclaimed Error).

## Edge cases covered
- Closed-E Result drop (`dropped_closed_result`) — must-use attaches to ModeResultClosed too.
- Embedded-interface obligation (`embedded_missing` / `embedded_satisfied`) — folded via
  `requiredMethods` with a cycle guard.
- Pointer- vs value-receiver satisfaction (`satisfied_pointer`/`satisfied_value`) — Methods
  keyed by star-stripped receiver, so both satisfy.
- Sealed interface trivially met (`sealed_trivial`).
- Same-enum vs cross-enum closed `?` (`ok_same_e` vs `missing_from`/`ok_from_present`).

## Tests that actually assert behavior
The corpus runner matches each inline `// want "substr"` against a diagnostic on the
same line and fails on any unclaimed Error — i.e. the message wording is asserted, not
just the count. Unit tests assert severity + code + message substring per case.

## Contradictions / risks
- `sema.Check` now runs the new checks for ALL `SemaCheck` callers (02/08 runners
  included). Confirmed no 02/08 fixture triggers a new Error (full suite green).
- One simplification (documented in assumptions): closed-E Err passthrough is "non-
  `E.Variant` arg ⇒ defer" rather than the lexical checker's richer passthrough
  analysis. Produces no false Error; equivalent for the corpus gate.

No quality blockers.
