# US-008 port parser package to goal

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (no branch — loop runs on linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Reimplement internal/parser as goal source under selfhost/parser, importing the
already-ported token, lexer, and ast packages. The ported parser must transpile
through the goal front-end (US-002 smoke gate), the generated Go must compile, and
the existing internal/parser behavioral tests must pass against the transpiled
output.
