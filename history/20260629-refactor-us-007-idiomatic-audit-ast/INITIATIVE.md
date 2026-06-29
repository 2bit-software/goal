# US-007 Idiomatic audit: ast

**Type**: refactor
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch — linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-28 |
| plan | completed | 2026-06-28 |
| tasks | completed | 2026-06-28 |
| implement | completed | 2026-06-28 |
| verify | in_progress | 2026-06-28 |

## Description

Idiomatic audit of `selfhost/ast` (step 3 of the SELF-HOST IDIOMATIC plan, the
highest idiomatic-opportunity package). Evaluate node-kind interface groups
(Node/Decl/Stmt/Expr/Spec) for `sealed interface`/`enum` representation, and
convert switch/type-switch-over-node-kind to `match` where it fits. Convert
where the goal idiom genuinely fits and is behavior-preserving; record
refusals-with-reason in DECISIONS.md. The US-003 verbatim selfhost is the
behavioral oracle — never change a public signature/type its tests pin; ast
tests must still pass against the transpiled package; task fixpoint must stay
byte-identical green. Per-package machine check: `goal fix` reports no remaining
auto-convertible propagation sites.
