# Implementation Plan — SEAM-CAP-3b

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/sealed_match_test.go` | Behavioral regression: transpile a same-package sealed-interface `match`, build+run in a temp `module goal`, assert identical result to `switch x:=n.(type)`. |
| `internal/sema/sealed_match_test.go` | Sema regression: non-exhaustive sealed match -> Error; exhaustive / `_` -> accepted. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/ast/goal_expr.go` | New `TypePattern` node (Type Expr, Binding *Ident, Lparen/Rparen). |
| `internal/ast/walk.go` | Walk TypePattern (Type, Binding). |
| `internal/ast/ast_test.go` | Cover TypePattern node identity + walk. |
| `internal/parser/goal_match.go` | `parsePattern`: `*`-leading -> `parseTypePattern` producing `*ast.TypePattern`. |
| `internal/sema/sema.go` | Add `SealedImpls map[string][]string` to Info (keep `Sealed map[string]bool`). |
| `internal/sema/resolve.go` | Init + Merge (union/dedup) SealedImpls; populate from `StructType.Implements` in `resolveTypeDecl`. |
| `internal/sema/check.go` | `checkOneMatch`: branch on TypePattern arms -> sealed exhaustiveness over SealedImpls. |
| `internal/backend/lower.go` | Helpers: `isSealedMatch(m)`, `typePatternType`/binding accessors, sealed-iface reverse lookup if needed. |
| `internal/backend/emit.go` | New `sealedMatch(m,pos,name)` path; dispatch from matchStmt/returnStmt/tryVarMatch/tryAssignMatch/matchValue. |
| `selfhost/ast/goal_expr.goal` | Mirror TypePattern node. |
| `selfhost/ast/walk.goal` | Mirror walk. |
| `selfhost/parser/goal_match.goal` | Mirror parser. |
| `selfhost/sema/sema.goal` | Mirror SealedImpls field. |
| `selfhost/sema/resolve.goal` | Mirror resolve/merge/populate. |
| `selfhost/sema/check.goal` | Mirror exhaustiveness. |
| `selfhost/backend/lower.goal` | Mirror helpers. |
| `selfhost/backend/emit.goal` | Mirror sealedMatch + dispatch. |
| `DECISIONS.md`, `prd.json`, `progress.txt` | Record decision, mark passes, log. |

## Dependency Graph

1. AST: `TypePattern` node + walk (foundation; both internal + selfhost).
2. Parser: produce TypePattern (depends on 1).
3. Sema Info: `SealedImpls` field + resolve/merge/populate (depends on 1).
4. Sema check: exhaustiveness (depends on 1, 3).
5. Backend: sealedMatch lowering + dispatch (depends on 1, 3).
6. Tests + fixtures (depends on 2,4,5).
7. selfhost mirror of 1-5 (kept valid goal; need not USE the feature) — landed
   in lockstep so `task fixpoint` and the port gate stay green.

## Interface Contracts

```go
// ast
type TypePattern struct {
    Type    Expr       // the concrete implementor type (e.g. *Ident)
    Lparen  token.Pos  // "("; zero if no binding
    Binding *Ident     // narrowed-value binding; or nil
    Rparen  token.Pos  // ")"; zero if no binding
}
func (p *TypePattern) Pos() token.Pos
func (p *TypePattern) End() token.Pos
func (*TypePattern) exprNode() {}

// sema.Info
SealedImpls map[string][]string // interface name -> implementor concrete type names

// backend
func isSealedMatch(m *ast.MatchExpr) bool          // any arm is *ast.TypePattern
func (e *emitter) sealedMatch(m *ast.MatchExpr, pos matchPos, name string)
```

Case label render: for TypePattern with Type `*Ident`, emit `case *Ident:` via the
emitter's existing expr renderer. Binding: `switch <guard> := subject.(type)` when an
arm uses its binding; rename binding -> guard in the arm body (reuse e.renames).

## Integration Points

- `emit.go` matchStmt (line ~2024), returnStmt (~1463), tryVarMatch (~1517),
  tryAssignMatch (~1564), matchValue (~1531): add an `isSealedMatch(m)` branch BEFORE
  the matchQualifier switch so a TypePattern match routes to `sealedMatch`.
- `check.go` checkOneMatch (~105): if arms are TypePatterns, run sealed
  exhaustiveness instead of the enum path.

## Testing Strategy

- Behavioral: mirror `internal/backend/crosspkg_goal_enum_test.go` style — transpile
  goal source via backend, write to temp `module goal`, `go build` + run, compare to
  a hand-written type-switch baseline (or assert printed output).
- Sema: `sema.Analyze(src)` over a non-exhaustive sealed match asserts a
  `non-exhaustive-match` Error; exhaustive + `_` cases assert no Error.
- Gates: `task check`, `task build`, `task fixpoint`.
