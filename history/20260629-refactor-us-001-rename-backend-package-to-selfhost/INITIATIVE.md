# US-001 Rename backend package to selfhost

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch; no feature branch per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Mirror internal/backend into selfhost/backend as .goal source (verbatim copy of
the 6 non-test .go files; no idiomatization — that is US-004/US-011). Wire the
selfhost transpile-and-build smoke gate (BuildTranspiled) and the behavioral gate
(BuildAndTest) over the self-contained subset of the backend tests, following the
established port pattern (token..pipeline already ported).
