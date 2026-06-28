# US-019 Parse expressions with Pratt and postfix ?

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Replace the parser's minimal operand+postfix expression grammar with
precedence-correct (Pratt / precedence-climbing) parsing: binary operators at
Go's five precedence levels, unary/prefix operators, and the postfix `?` unwrap
operator as `ast.UnwrapExpr`. Selector/call/index postfix chains continue to
bind tightest. A test asserts `f(x)?`, `a.b?`, and a mixed-precedence binary
expression parse to the expected tree shape.
