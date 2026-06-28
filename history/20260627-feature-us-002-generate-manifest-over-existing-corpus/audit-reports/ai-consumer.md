# Audit: AI-Consumer Readiness — US-002

## Findings

- The data format is fully specified by the existing `corpus.Manifest`/`Case`
  types (US-001) — field names and types are concrete.
- Acceptance criteria are directly assertable: count transpile vs check cases;
  load the file without error; regenerate and compare bytes.
- Directory roots to walk are explicit (`features/*/examples`, top-level
  `testdata`, `testdata/check`).

No CRITICAL or MAJOR findings. An AI agent can implement without clarifying
questions.

## Assumptions

- Paths in the manifest are stored repo-root-relative with forward slashes.
- Check cases carry an empty `Expected` and `Normalize=none`; inline `// want`
  markers are honored by a later runner (US-004), not by this generator.
