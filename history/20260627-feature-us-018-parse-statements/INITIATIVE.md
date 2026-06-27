# US-018 Parse statements

**Type**: feature
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (no new branch; loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-26 |
| plan | done | 2026-06-26 |
| tasks | done | 2026-06-26 |
| implement | done | 2026-06-26 |
| verify | done | 2026-06-26 |

## Description

Extend internal/parser to parse function bodies into a real statement list:
assignment, short-var declaration, if/for/switch (incl. range and three-clause
for), return/defer/go/break/continue, nested blocks, and const/var/type
declarations used as statements. A test parses a function body containing each
statement form and asserts the statement list shape.
