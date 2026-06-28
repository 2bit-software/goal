# Implementation Plan — US-002

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/testdata/match_value_result_return.goal` | Fixture: return-position Result match. |
| `internal/backend/testdata/match_value_option_assign.goal` | Fixture: assignment-position Option match. |

(Fixtures only if needed; tests may inline source via mustTranspile-style helpers.
Prefer inline source strings in the test, consistent with existing backend tests
that build temp modules.)

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/emit.go` | Generalize `resultMatch`/`closedResultMatch`/`optionMatch` to `(m, pos, name)`; add `armBodyRenamedWrap`; wire Result/Option value-match into `returnStmt`, `tryVarMatch`, `tryAssignMatch`. |
| `internal/backend/backend_test.go` | Add a test covering value-position Result match (return) and Option match (assignment + return). |

## Dependency Graph

1. Generalize `resultMatch`/`closedResultMatch`/`optionMatch` signatures to accept
   `(m *ast.MatchExpr, pos matchPos, name string)` and emit arm bodies through
   `armWrap` / new `armBodyRenamedWrap` (statement position unchanged: posStmt).
2. Update `matchStmt` to call them with `posStmt, ""`.
3. Wire dispatch in `returnStmt` (posReturn), `tryVarMatch` (posVar),
   `tryAssignMatch` (posVar, type via `inferMatchType`).
4. Add backend test.

## Lowering shapes

- Return-position Result (open E):
  `v, err := <subj>; if err != nil { return <errBody> } else { return <okBody> }`.
- Return-position Option:
  `if o := <subj>; o != nil { [v := *o]; return <someBody> } else { return <noneBody> }`.
- Assignment-position: emit `var name T` (T from `inferMatchType`), then the same
  split with arms `name = <body>`.

## Testing

Backend test transpiles each shape, asserts no error / valid Go (temp-module build
or go/format parse) and that both arm bodies appear. Subjects mirror
features/03-result (`parse(input)`) and features/04-option (`find(id)`) call shapes.
