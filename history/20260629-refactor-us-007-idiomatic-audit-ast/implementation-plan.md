# Implementation Plan — US-007 Idiomatic audit: ast

## Outcome summary

The audit concludes with a recorded DECISION and NO `.goal` source change —
identical in shape to US-005 (token) and US-006 (lexer). Every idiom candidate
in `selfhost/ast` is refused for a concrete behavior-preserving reason; the only
artifact is a new DECISIONS.md ledger section.

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `DECISIONS.md` | Add a "self-host idiomatic audit — US-007 (ast)" section recording the refusals (category interfaces -> sealed interface; FuncMod/ChanDir -> enum; Walk type-switch -> match) and the no-Result/? finding, each with reason. |
| `prd.json` | Set US-007 `passes: true` (done by loop-runner after green verify). |
| `progress.txt` | Append the US-007 iteration entry (loop-runner). |

## Dependency Graph

1. Confirm the package facts (done in research): no error-returning funcs; the
   only node-kind type-switch is Walk over the plain `Node` interface; FuncMod
   and ChanDir are `type X int` + iota consumed cross-package.
2. Write the DECISIONS.md ledger entry.
3. Verify (`goal fix`, `task check`, `task build`, `task fixpoint`).

## Idiom decisions (the substance of the audit)

- **Category interfaces Node/Decl/Stmt/Expr/Spec -> sealed interface: REFUSE.**
  Changing the go/ast-mirrored marker interfaces to `sealed interface` would
  change the oracle-pinned public API (markers `declNode()`/`stmtNode()`/
  `exprNode()`/`specNode()` -> synthesized `isNAME()`; every node struct needs
  `implements NAME for T`) and, per the §9 switch-coexistence rule, turn every
  plain type-switch over these types across selfhost/sema, selfhost/backend, and
  selfhost/parser into a closed-enum-switch COMPILE ERROR — a large out-of-scope
  blast radius. Consequently Walk's own `switch node.(type)` cannot become
  `match` (match needs a closed enum/sealed scrutinee; Node stays a plain
  interface).

- **FuncMod -> enum: REFUSE.** Public, oracle-pinned `type FuncMod int` + iota,
  consumed cross-package by `==`/`!=` (sema/question.goal, sema/convert.goal,
  sema/resolve.goal) and a plain `switch` (backend/emit.goal:362). A goal enum
  lowers to a boxed sealed interface, breaking `==` and turning the plain switch
  into a §9 closed-enum-switch compile error — all out of scope.

- **ChanDir -> enum: REFUSE.** Same shape: public `type ChanDir int` + iota,
  consumed by plain switches in sema/resolve.goal and backend/emit.goal.

- **Result/Option/?: NONE APPLY.** `selfhost/ast` has zero error-returning
  functions; `goal fix selfhost/ast/*.goal` already produces no diff and no
  report (AC-2 holds).

## Integration Points

DECISIONS.md is the canonical refusal ledger; the new section sits after the
US-006 (lexer) section, matching that format.

## Testing Strategy

No new tests. Behavior preservation is proven by the existing gates:
- `goal fix selfhost/ast/*.goal` — no diff, no report (machine check / AC-2).
- `task check` — runs go vet + the full Go test suite incl. the internal/selfhost
  port gate (transpiles selfhost/ast and runs internal/ast tests against it) and
  internal/ast's own tests.
- `task build` — builds the toolchain.
- `task fixpoint` — goal-c-1 and goal-c-2 emit byte-identical Go (the oracle).
