# US-024 enforce capability denial

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — repo runs the loop on the current branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | complete | 2026-06-27 |
| plan | complete | 2026-06-27 |
| tasks | complete | 2026-06-27 |
| implement | complete | 2026-06-27 |
| verify | pending | - |

## Description

The goscript interpreter must turn a DENIED capability into a refusal: running
under a CapabilitySet that does not grant a capability causes the corresponding
host effect (today: the stdout write behind fmt.Println / emitStdout) to fail
with a located, named capability error instead of silently performing — or
silently skipping — the effect. The sandbox must be real.
