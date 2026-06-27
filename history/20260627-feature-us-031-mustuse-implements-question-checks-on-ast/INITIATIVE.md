# US-031 Reimplement mustuse/implements/question checks

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the current branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Reimplement the remaining lexical static checks over the AST in `internal/sema`,
so all diagnostics survive the front-end change:

- **must-use** (feature 03-result): a Result-returning call dropped on the floor is an Error.
- **implements** (feature 07): a `type T struct implements I` must satisfy I.
- **question / ?-arity-refusal** (feature 05-question-prop open-E, and 06-error-e closed-E):
  open-E `?` callee must end in error; closed-E `?` needs a registered `from func` when the
  error enum differs; `Result.Err(X)` must stay closed over the function's error enum.

Acceptance: every must-use, implements, and question case in `testdata/check`
(03-result, 06-error-e, 07-implements) passes through the AST-based `corpus.SemaCheck`
via the corpus check runner.
