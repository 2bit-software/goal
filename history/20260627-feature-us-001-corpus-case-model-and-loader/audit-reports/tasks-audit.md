# Tasks Audit — US-001

- 3 tasks, each independently committable, each <5 files, each with a concrete
  verify command.
- Ordering respects the dependency graph (model → fixture → test).
- Coverage: every plan file (corpus.go, manifest.json, corpus_test.go) appears in
  a task; every FR (FR-1..FR-5) and acceptance criterion is covered.
- No CRITICAL/MAJOR findings.

## Assumptions
- Tasks 1-3 will likely be implemented in a single agent turn since the package
  is tiny; they remain logically separable.
