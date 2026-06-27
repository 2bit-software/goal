# US-020 parse enum sealed implements

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Extend internal/parser to parse the goal closed-type declaration surface:
enum declarations (data-less and payload variants), sealed interface
declarations, and the `implements` clause on a struct type declaration. A test
parses the 01-enums and 07-implements example inputs and asserts the
variant/field/implements structure.
