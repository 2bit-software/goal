# US-006 Port lexer package to goal

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-28 |
| plan | completed | 2026-06-28 |
| tasks | completed | 2026-06-28 |
| implement | completed | 2026-06-28 |
| verify | in_progress | 2026-06-28 |

## Description

Reimplement internal/lexer as goal source under selfhost/lexer importing the
ported selfhost/token, so tokenization is goal-native. It must transpile via the
US-002 smoke gate, the generated Go must compile (unicode/unicode/utf8 pass
through as foreign imports), and the existing internal/lexer tests must pass
against the transpiled package.
