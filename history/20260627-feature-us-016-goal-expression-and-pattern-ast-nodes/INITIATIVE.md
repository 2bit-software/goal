# us-016-goal-expression-and-pattern-ast-nodes

**Type**: feature
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-26 |
| plan | completed | 2026-06-26 |
| tasks | completed | 2026-06-26 |
| implement | completed | 2026-06-26 |
| verify | in_progress | 2026-06-26 |

## Description

US-016: Add goal expression and pattern AST nodes so the three meanings of
`Enum.Variant(x)` (construct vs. destructure-bind vs. ordinary call) are
distinct node types. `internal/ast` gains MatchExpr/MatchArm,
VariantPattern/RestPattern, UnwrapExpr, VariantLit/LabeledArg, and
SpreadElement, each wired into Walk. A test asserts a construction VariantLit
and a destructuring VariantPattern are distinct node types and both walk
correctly. This is the structural fix for the Match-before-Enums ordering hack.
