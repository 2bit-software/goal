# Implementation Plan — US-032 Emit the Go subset from AST

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/testdata/plain_full.goal` | Plain-Go fixture exercising the full ordinary-Go subset, notably a `switch` with `case`/`default`, plus struct/map/slice composites, defer, and multi-return — the behavioral-tier witness for AC-2. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/emit.go` | Add `*ast.SwitchStmt` + `*ast.CaseClause` cases to the `stmt` type switch (new `switchStmt`/`caseClause` helpers), mirroring `ifStmt`/`forStmt`. Refresh the package doc to note US-032 completes the ordinary-Go subset. |
| `internal/backend/backend_test.go` | Add `TestASTEngineBehavioralTierFull` running `plain_full.goal` through `corpus.RunCompile` (build+vet), and a focused `TestASTEngineEmitsSwitch` asserting a `switch` source transpiles to valid Go containing `switch`/`case`/`default`. |

## Package Structure

```
internal/backend/
  emit.go              (modified: + switchStmt/caseClause)
  backend.go           (unchanged)
  backend_test.go      (modified: + full-subset + switch tests)
  testdata/
    plain.goal         (unchanged, US-026 fixture)
    plain_full.goal    (new fixture)
```

## Dependency Graph

1. `emit.go` switch/case emission (no new deps; uses existing `block`, `expr`,
   `exprList`, `stmt`).
2. `testdata/plain_full.goal` (a goal/Go source, no code deps).
3. `backend_test.go` tests (depend on 1 and 2; reuse `corpus.RunCompile` +
   `corpus.TranspilerFunc(backend.Transpile)` seam from US-026).

## Interface Contracts

No exported API change. New unexported emitter helpers in `emit.go`:

```go
func (e *emitter) switchStmt(s *ast.SwitchStmt)   // emits: switch [init;] [tag] { clauses }
func (e *emitter) caseClause(c *ast.CaseClause)    // emits: case e1, e2: stmts  |  default: stmts
```

`SwitchStmt` fields (from internal/ast/ast.go): `Init ast.Stmt`, `Tag ast.Expr`,
`Body *ast.BlockStmt` (List of `*ast.CaseClause`). `CaseClause` fields:
`List []ast.Expr` (nil ⇒ default), `Body []ast.Stmt`.

## Integration Points

- `emit.go` `stmt(s ast.Stmt)` type switch gains `case *ast.SwitchStmt:` →
  `e.switchStmt(s)`. The switch body is a `*ast.BlockStmt` whose elements are
  `*ast.CaseClause`; `caseClause` is dispatched from `switchStmt` directly (not
  via the generic `block`, since a case clause is not a normal statement line).
- `backend.Transpile` (parse → Emit → Format) is unchanged; the new switch
  support flows through it automatically, format-once via `GoFormatter`.

## Testing Strategy

- `TestASTEngineEmitsSwitch`: inline goal source with a tag-and-default switch
  inside a func; `backend.Transpile` then `go/format.Source` must succeed and the
  output must contain `switch`, `case`, and `default`.
- `TestASTEngineBehavioralTierFull`: build a `corpus.Case{Kind:KindTranspile,
  Mode:ModeFile, Input:"internal/backend/testdata/plain_full.goal"}` and assert
  `corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile))`
  returns nil (generated Go builds + vets). `-short`-skipped like the existing
  behavioral test, since it spawns the go toolchain.
- Existing `TestASTEngineBehavioralTier` (plain.goal) stays as the minimal
  witness; the new test extends coverage to the full subset.

## Spec Traceability

- FR-1 (full subset incl. switch) → emit.go switch/case helpers + fixture.
- FR-2 (format once) → unchanged `backend.Transpile` GoFormatter pass.
- FR-3 (behavioral conformance) → `TestASTEngineBehavioralTierFull`.
- FR-4 (goal constructs still gated) → unchanged `default` fail arms in emit.go;
  no goal-specific node added.
