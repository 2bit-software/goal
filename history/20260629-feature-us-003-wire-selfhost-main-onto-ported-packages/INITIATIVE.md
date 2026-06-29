# US-003 Wire self-host main onto ported packages

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch — no feature branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-29 |
| plan | done | 2026-06-29 |
| tasks | done | 2026-06-29 |
| implement | done | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Make selfhost/main.goal and the whole selfhost tree import
goal/selfhost/{token,lexer,ast,parser,sema,project,pipeline,backend} instead of
any goal/internal/* package, so the goal-built compiler is self-contained. The
bootstrap builds goal-c-1 and goal-c-2 from the self-contained selfhost tree and
`task fixpoint` passes byte-identical, while the goal-built compiler passes the
corpus transpile + behavioral tiers (via `task check`). Verbatim stage — no
idiomatic rewrite yet. Establishes the differential ORACLE for later idiomatic
audits.
