# fix-iota-const-block-miscompile

**Type**: bug
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch; no branch per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| reproduce | in_progress | 2026-06-29 |
| fix | pending | - |
| verify | pending | - |
| done | pending | - |

## Description

US-001 — Fix iota const-block miscompile. Idiomatic Go enum const blocks must
transpile correctly so the compiler's own iota-based declarations (e.g.
internal/token) survive a round-trip. `const ( Red Color = iota; Green; Blue )`
silently miscompiles (exit 0): `Green` and `Blue` collapse into `Green Blue`
(name + type), losing Blue's iota value.
