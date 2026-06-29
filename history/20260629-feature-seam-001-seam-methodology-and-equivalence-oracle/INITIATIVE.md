# SEAM-001 seam methodology and equivalence oracle

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch; no branch created)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Establish a documented, repeatable gate for cross-package idiom changes (the
SEAM PRD) that may alter emitted Go. The per-package audits (US-005..US-013)
required byte-identical emitted Go; seam stories relax that gate and re-prove
equivalence via fixpoint self-consistency, the corpus behavioral tier, and
reviewed golden regeneration. This story is documentation + procedure only:
add a "Seam methodology" section to DECISIONS.md. No source idiom change.
