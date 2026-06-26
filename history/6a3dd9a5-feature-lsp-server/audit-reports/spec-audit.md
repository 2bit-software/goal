# Spec Audit — Goal LSP M1

## Completeness

- Requirements cover the full open→edit→clear→close lifecycle (FR-001, 005, 006).
- Multi-error reporting (FR-007) and severity mapping (FR-003) are specified and
  match the verified behavior of `check.Run` (accumulates, sorted; Error vs Warning).
- Acceptance criteria are concrete and testable against a known sample file.
- **MINOR**: cross-file-dependent guarantees won't fire under single-file M1 scope —
  explicitly documented in Out of Scope; acceptable for the slice.

## AI-consumer readiness

- Exact reuse surface named with signatures (`check.Analyze`, `OffsetToPosition`,
  `check.Diagnostic` fields) — an implementer needs no guessing for the data flow.
- Position-base conversion (1-based → 0-based, exclusive end) called out as the key
  correctness risk.
- Library/transport/scheduling decisions are fixed with defaults, removing ambiguity.
- **MINOR**: token-length for diagnostic ranges is not always available from a byte
  offset; plan step should decide a default range width (e.g. to end of the
  offending identifier/line). Flagged for the plan.

## Verdict

No CRITICAL or MAJOR findings. Two MINOR items, both resolved or deferred to the
plan step. Spec is ready to proceed to planning.
