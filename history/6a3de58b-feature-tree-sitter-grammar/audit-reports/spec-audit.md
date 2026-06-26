# Spec Audit — tree-sitter-goal M1

## Completeness
- Requirements pin a measurable bar (zero ERROR over 103 files) — strong, testable.
- goal's full delta set is enumerated as first-class nodes (FR-003), sourced from the
  established lexer/pass facts of prior layers.
- ASI (no semicolons) called out (FR-004) — the classic Go-grammar pitfall.
- **MINOR**: "no exotic Go fidelity" boundary is acceptable but should be re-checked if a
  corpus file fails to parse — any ERROR node is a real gap, not out-of-scope. Mitigated by
  the parse-all-corpus acceptance gate.

## AI-consumer readiness
- Toolchain, location, file list, and verify commands are concrete.
- Highlight capture names specified, so query authoring is unambiguous.
- **MINOR**: external scanner (`scanner.c`) left conditional ("only if needed") — the plan
  step should decide quickly whether pure-DSL can express raw strings + ASI, to avoid churn.

## Verdict
No CRITICAL/MAJOR findings. Two MINOR items deferred to plan/implement. Ready to plan.
