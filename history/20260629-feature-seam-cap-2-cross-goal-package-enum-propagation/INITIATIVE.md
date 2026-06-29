# SEAM-CAP-2: cross-.goal-package enum/sema-fact propagation

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Make an enum (and its §8.1 facts) defined in a sibling `.goal` package visible to
consumers in OTHER `.goal` packages during the real per-package `goal build ./selfhost`
bootstrap. SEAM-CAP enabled cross-package enum-match lowering only when the defining
package is available as generated `.go`; the live bootstrap transpiles each package from
sibling `.goal` SOURCE and `foreignDecls` reads only `.go`, so a sibling-.goal-defined
enum is invisible to dependents. Extend foreign enrichment so a resolved import dir that
holds only `.goal` source is lexed/parsed and its exported enums (+ §8.1 facts) folded in,
and lower bare cross-package variant construction to the §8.1 form. Apply in BOTH
internal/ and selfhost/.
