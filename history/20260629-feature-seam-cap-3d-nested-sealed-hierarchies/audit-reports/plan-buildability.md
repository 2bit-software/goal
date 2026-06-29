# Plan buildability audit

- Dependency order valid: sema cascade → backend helper → backend emission →
  tests. No forward references.
- Interface contracts concrete: `cascadeSealedImpls()`, `sealedEmbeds(info,
  iface) []string`. Match existing helper styles (isSealed, addImplementor).
- File paths verified against the tree (resolve.go/.goal, lower.go/.goal,
  emit.go/.goal all exist; test dirs exist).
- Integration points name file + function + how.
- selfhost mirror requirement is explicit (port gate compiles .goal as Go).
- No CRITICAL/MAJOR findings.

## Assumptions
- selfhost .goal is line-for-line Go-superset; new helpers mirror exactly or the
  port gate fails to compile.
- Map-iteration nondeterminism in the cascade does not affect emitted Go
  (SealedImpls order is not emitted; marker order follows source-ordered
  EmbeddedIfaces). Fixpoint-safe.
