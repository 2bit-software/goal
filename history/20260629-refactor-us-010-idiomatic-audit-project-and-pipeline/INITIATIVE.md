# US-010 Idiomatic audit: project and pipeline

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-29 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Step 3 of 3 (idiomatic audit) of the SELF-HOST IDIOMATIC PRD, sixth audit:
selfhost/project and selfhost/pipeline. Convert genuinely-fallible internal
helpers to Result/Option + `?` where behavior-preserving (per the US-009 rule:
an open-E (T,error)->Result conversion lowers to byte-identical Go and is safe
only on an exported fn with no in-tree callers and no oracle test). Record
genuine refusals-with-reason in DECISIONS.md. The US-003 verbatim self-host is
the behavioral oracle: never change a public signature its tests pin; project
and pipeline tests must pass against the transpiled packages; task fixpoint must
stay byte-identical. Per-package machine check: `goal fix` reports no remaining
auto-convertible propagation sites.
