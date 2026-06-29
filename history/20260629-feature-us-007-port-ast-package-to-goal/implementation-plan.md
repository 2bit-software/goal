# Implementation Plan — US-007 Port ast package to goal

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/ast/ast.goal` | Copy of internal/ast/ast.go — core AST node defs |
| `selfhost/ast/walk.goal` | Copy of internal/ast/walk.go — Walk traversal |
| `selfhost/ast/goal_decl.goal` | Copy of internal/ast/goal_decl.go — goal decl nodes |
| `selfhost/ast/goal_expr.goal` | Copy of internal/ast/goal_expr.go — goal expr nodes |
| `selfhost/ast/goal_stmt.goal` | Copy of internal/ast/goal_stmt.go — goal stmt nodes |

dump.go is intentionally NOT ported (reflection-driven debug Sexpr renderer,
off the compile path, not referenced by ast_test.go or any other ast file).

### Modified Files
| File | Changes |
|------|---------|
| `internal/selfhost/port_test.go` | Add TestPortedAstPackage (two-gate port test) |
| `prd.json` | Set US-007 passes:true (after green) |
| `progress.txt` | Append US-007 entry |

## Package Structure

```
selfhost/
  token/token.goal      (US-005, leaf)
  lexer/lexer.goal      (US-006, -> token)
  ast/                  (US-007, NEW, -> token)
    ast.goal
    walk.goal
    goal_decl.goal
    goal_expr.goal
    goal_stmt.goal
```

## Dependency Graph

1. selfhost/token (already ported)
2. selfhost/ast (depends on token only — same shape as lexer)
3. port_test gates (depend on token + ast layout)

## Interface Contracts

The ported package is a verbatim copy of `package ast` (Go superset == valid
goal), so all exported types/funcs (Node, Walk, Visitor, the goal node types,
etc.) keep their existing signatures. No signature changes.

Port test mirrors TestPortedLexerPackage:
```go
func TestPortedAstPackage(t *testing.T) {
    // Discover selfhost/token and selfhost/ast
    // layout = {"internal/token": tokenPkg, "internal/ast": astPkg}
    // selfhost.BuildTranspiled(layout)
    // deps = {"internal/token": tokenPkg}
    // selfhost.BuildAndTest("internal/ast", astPkg, []string{"../ast/ast_test.go"}, deps)
}
```

## Integration Points

- `project.Discover("../../selfhost/ast")` -> one package named "ast".
- `internal/selfhost.BuildTranspiled` / `BuildAndTest` — reused unchanged (the
  deps param added in US-006 already supports the in-module token dep).
- `task fixpoint` auto-discovers selfhost/ast (project.Discover walks the tree);
  no harness change.

## Testing Strategy

- Compile gate: BuildTranspiled over {token, ast}.
- Behavioral gate: BuildAndTest ast against ../ast/ast_test.go with token dep.
- Project gates: `task check`, `task build`, and `task fixpoint` stay green.
