# SEAM-002: token.Kind (+ FuncMod, ChanDir) iota -> goal enum, tree-wide

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch — linear history on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Convert the self-hosted compiler's `FuncMod` and `ChanDir` iota types
(`selfhost/ast`) to goal `enum`s and convert every cross-package consumer
(`==`/`!=` comparisons and switch-over-them in `selfhost/{sema,backend,parser}`)
to `match`. Keep `selfhost/token` `Kind` as iota — it has a genuine
numeric-identity dependence (array indexing, range arithmetic, contiguous
numbering) — and document that refusal per AC-1's escape hatch.

This is a SEAM story under the relaxed gate (DECISIONS.md "Seam methodology"):
emitted Go may change; equivalence is re-proven via `task fixpoint`
(stage1==stage2), the corpus behavioral tier staying green, and reviewed
golden/oracle-test updates.
