# Technical requirements / research — US-007

## Package shape

`selfhost/ast` is the goal AST, modeled on `go/ast`, trimmed to goal's subset
plus goal-specific nodes (enum/sealed/implements/from-derive, match/patterns/
construction). Files: ast.goal (core Go-subset nodes + the Node/Decl/Stmt/Expr/
Spec category interfaces), goal_decl.goal, goal_expr.goal, goal_stmt.goal, and
walk.goal (the depth-first traversal type-switch over Node).

## Idiom candidates and findings

### Category interfaces Node/Decl/Stmt/Expr/Spec -> sealed interface
REFUSE. Converting these go/ast-mirrored marker interfaces to `sealed interface`
would (a) change the oracle-pinned public API (the `declNode()`/`stmtNode()`/
`exprNode()`/`specNode()` markers become a synthesized `isNAME()` marker and
every node struct would need `implements NAME for T`), and (b) per the §9
switch-coexistence rule, turn every plain type-switch over these types across
selfhost/sema, selfhost/backend, and selfhost/parser into a closed-enum-switch
COMPILE ERROR — a large out-of-scope blast radius. Walk's own `switch
node.(type)` would also be forced to `match`, which is only legal over a closed
enum/sealed scrutinee; since Node must remain a plain interface, the Walk
type-switch -> match conversion is refused too.

### FuncMod and ChanDir (`type X int` + iota) -> enum
REFUSE. Both are public, oracle-pinned `go/ast`-style ordered integer enums
consumed cross-package by `==`/`!=` comparisons and plain switches in
selfhost/sema (resolve, question, convert) and selfhost/backend (emit). A goal
enum lowers to a boxed sealed interface (not an int), so `==` comparison breaks
and those plain switches become §9 closed-enum-switch compile errors — all
out-of-scope and oracle-violating. Same canonical "ordered/comparable iota int,
keep as-is" case as token.Kind (US-005).

### Result/Option/? conversions
NONE APPLY. `selfhost/ast` has zero error-returning functions; it is pure data
plus the total `Walk` traversal. `goal fix selfhost/ast/*.goal` produces no diff
and reports nothing — AC-2 already holds.

## Outcome

Audit outcome is a recorded DECISION (DECISIONS.md), not a `.goal` source change
— identical in shape to US-005 (token) and US-006 (lexer). Verify with `goal
fix` (no diff/report), `task check`, `task build`, `task fixpoint`.
