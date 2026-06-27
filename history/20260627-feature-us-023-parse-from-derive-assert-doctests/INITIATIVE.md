# US-023 Parse from/derive, assert, doctests

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no feature branch — loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Parse the remaining goal constructs so the parser grammar is complete:
`from func` and `derive func` declarations (bodyless and bodied), `assert`
statements (bare and printf-message form), and `///` doctest comments as
structured AST nodes. A test parses the 10-assert, 11-doctests, and
12-derive-convert example inputs and asserts their node shapes.
