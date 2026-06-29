# US-011 Idiomatic audit: backend

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch — linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-29 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Idiomatic audit of selfhost/backend (the largest ported package): convert
genuinely-fallible package-internal helpers to Result/`?` where behavior-
preserving, convert switch-over-in-file-enum to `match` where it fits, and
record genuine refusals in DECISIONS.md. Gated by backend tests, the corpus
transpile + behavioral tiers, and a byte-identical `task fixpoint`.
