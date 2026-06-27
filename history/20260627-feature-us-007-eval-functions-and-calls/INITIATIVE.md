# us-007-eval-functions-and-calls

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on the base branch, no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | - |
| tasks | completed | - |
| implement | completed | - |
| verify | pending | - |

## Description

US-007 "Eval functions and calls" of the goscript-runtime PRD. The interpreter
(internal/interp) must evaluate function declarations as values, bind
parameters, support multiple return values, and support recursion. Acceptance:
a unit test runs recursive factorial and fibonacci goal functions and asserts
their returned values.
