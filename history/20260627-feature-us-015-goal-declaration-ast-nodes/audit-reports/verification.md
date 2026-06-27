# Verification — US-015

## Gates (prd.json verifyCommands)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages ok; internal/ast green)

## Acceptance criteria
| Criterion | Evidence |
|-----------|----------|
| ast defines EnumDecl/Variant/PayloadField, SealedInterfaceDecl, ImplementsClause, and from/derive FuncDecl modifiers | internal/ast/goal_decl.go (nodes + FuncMod); internal/ast/ast.go (FuncDecl.Mod/ModPos, StructType.Implements) — compiles. |
| A test asserts Walk descends into each new node's children | internal/ast/ast_test.go `TestWalkGoalDeclChildren` — PASS. Asserts Walk descends EnumDecl→Name/Variants, Variant→Name/PayloadField, PayloadField→Name/Type, SealedInterfaceDecl→Name/Methods, StructType→ImplementsClause→Type; plus from/derive Mod recorded and FuncDecl.Pos()==ModPos. |

## Result
All acceptance criteria met; all gates green. Implementation verified.
Committed as f11594a on ralph/ast-frontend-rewrite.
