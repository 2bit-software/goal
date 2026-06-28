# us-045-goal-fmt-source-printer

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

US-045: Add a `goal fmt` source formatter (printer) that prints goal source from
the AST while preserving comments and is idempotent. Acceptance criteria:

1. `goal fmt` prints goal source from the AST preserving comments, and is idempotent.
2. A test asserts `fmt(fmt(src)) == fmt(src)` for every corpus `.goal` input and that
   comments are retained on a sample.
