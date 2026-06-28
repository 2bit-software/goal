# Completeness Audit — US-004

No CRITICAL or MAJOR findings. The spec mirrors the proven US-003 pattern and the
existing `// want` marker semantics in internal/check/check_test.go.

## Findings
- MINOR: Multiple markers on one line are permitted by the existing harness; spec
  does not state this explicitly. Runner will follow the existing per-line list
  semantics. Non-blocking.
- MINOR: Empty `// want ""` (clean-program assertion via absence of markers) is
  covered by FR-2/FR-3 jointly. Non-blocking.

## Assumptions
- File-mode, single-source check cases only (package-mode is out of scope, US-009).
- Diagnostic line is derived via check.OffsetToPosition, matching the existing
  harness exactly.
