# Implementation Tasks — SEAM-CAP-3b

Note: this is one atomic capability touching ast+parser+sema+backend across the
internal/ Go compiler and the selfhost/ .goal mirror. The pieces only compile
together (undefined symbols mid-edit are expected per the Codebase Patterns note);
the whole set lands in one commit. Tasks are an ordering, not separate commits.

## Task 1: AST TypePattern node
**Status**: pending
**Files**: internal/ast/goal_expr.go, internal/ast/walk.go, internal/ast/ast_test.go,
selfhost/ast/goal_expr.goal, selfhost/ast/walk.goal
**Depends on**: none
**Spec coverage**: FR-2, FR-5
### Instructions
Add `TypePattern{Type Expr; Lparen token.Pos; Binding *Ident; Rparen token.Pos}`
mirroring VariantPattern (Pos/End/exprNode). Walk Type then Binding. Mirror in
selfhost. Add ast_test coverage proving it is a distinct node type and walks its
children.

## Task 2: Parser type-pattern arms
**Status**: pending
**Files**: internal/parser/goal_match.go, selfhost/parser/goal_match.goal
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-5
### Instructions
In `parsePattern`, when the pattern starts with `*` (token.MUL/STAR), parse a
`TypePattern`: parse the type expr (a `*T` StarExpr), then an optional `(binding)`.
Leave the `_` and variant-pattern branches unchanged. Mirror in selfhost.

## Task 3: Sema implementor registry
**Status**: pending
**Files**: internal/sema/sema.go, internal/sema/resolve.go,
selfhost/sema/sema.goal, selfhost/sema/resolve.goal
**Depends on**: none
**Spec coverage**: FR-1, FR-5
### Instructions
Add `SealedImpls map[string][]string` to Info. Initialize it in Resolve, union+dedup
in Merge. In resolveTypeDecl, when a StructType has an `Implements` clause, append the
struct name to SealedImpls[ifaceName]. Mirror in selfhost.

## Task 4: Sema exhaustiveness
**Status**: pending
**Files**: internal/sema/check.go, selfhost/sema/check.goal, internal/sema/sealed_match_test.go
**Depends on**: Task 1, Task 3
**Spec coverage**: FR-4
### Instructions
In checkOneMatch, detect TypePattern arms. Resolve the sealed interface from the
first arm's concrete type via reverse lookup over SealedImpls restricted to sealed
keys. Require all implementors covered or a `_` rest-arm; else emit a
`non-exhaustive-match` (02-match) Error naming the missing implementors. Defer (no
false reject) when the iface/implementors can't be resolved in-file. Mirror in
selfhost. Add sema regression test.

## Task 5: Backend sealedMatch lowering
**Status**: pending
**Files**: internal/backend/lower.go, internal/backend/emit.go,
selfhost/backend/lower.goal, selfhost/backend/emit.goal,
internal/backend/sealed_match_test.go
**Depends on**: Task 1, Task 3
**Spec coverage**: FR-3, FR-5
### Instructions
Add `isSealedMatch(m)` (any arm is TypePattern). Add `sealedMatch(m,pos,name)`
emitting `switch [guard :=] subject.(type)` with `case <type>:` per arm (render via
e.expr on the type), `_`->default, else panicking default; bind+rename like
enumMatch. Dispatch from matchStmt, returnStmt, tryVarMatch/matchValue,
tryAssignMatch BEFORE the matchQualifier checks. Mirror in selfhost. Add behavioral
regression test (transpile + build + run == type-switch baseline).

## Task 6: Gates + bookkeeping
**Status**: pending
**Files**: DECISIONS.md, prd.json, progress.txt
**Depends on**: Task 1-5
**Spec coverage**: all
### Instructions
Run task check, task build, task fixpoint. Record DECISIONS.md section. Mark
SEAM-CAP-3b passes:true. Append progress.txt entry.
