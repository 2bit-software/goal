# Research — US-039 derive/from lowering

## Summary

No external research required: this is a port of an in-repo, known-good encoding.
The legacy splice pass `internal/pass/derive.go` already implements the exact
field-by-field conversion lowering and produces build+vet-clean Go (it is the
source of the checked-in features/12 goldens). The task is to reimplement that
lowering on the AST backend, reading resolved facts from `sema.Info` instead of
the token-scanned `analyze.Tables`.

## Reference mapping (legacy -> AST)

| Legacy (internal/pass/derive.go) | AST equivalent (this story) |
|---|---|
| `analyze.Tables.Structs` | `sema.Info.Structs` (`map[string][]sema.Field`) |
| `analyze.Tables.FromRegistry` | `sema.Info.FromRegistry` (`map[[2]string]sema.ConvEntry`) |
| token scan of `derive func ...` | `*ast.FuncDecl{Mod: ast.FuncDerive}` (Params/Results/Body) |
| token scan of override literal | `Body` -> `ReturnStmt` -> `CompositeLit` (KeyValueExpr + `...derive(src)` SpreadElement) |
| `__goal_v%d` temp names | scope-aware `emitter.gensym` |

The conversion-strategy helpers (`derefType`, `ptrInner`, `arrElem`, `mapKV`,
`elemConv`, `splitReturn`, `findField`) operate purely on the already-resolved
type strings, so they port verbatim into `internal/backend/lower.go`.

## AST shapes confirmed (parsed features/12 examples)

- bodyless total derive (`slice.goal toIDs`): `FuncDecl{Mod:FuncDerive, Body:nil}`,
  one named param, one result.
- bodyless fallible derive (`from_storage.goal fromStorage`): result list is
  `(EventExecution, error)` -> fallible.
- bodied derive with overrides (`to_storage.goal toStorage`): `Body` returns a
  `CompositeLit` whose `Elts` are `KeyValueExpr` overrides (`Audit: _` is the skip,
  value is `Ident "_"`) plus a `SpreadElement{X: CallExpr{Fun: Ident "derive",
  Args:[Ident "e"]}}`.

## Confidence

High — the encoding is already proven by the splice engine and its goldens; the
only change is the fact source (sema vs analyze) and the gensym naming.

## Open questions / out of scope

- Foreign-package derive (US-009 fixture) runs via the splice engine through
  corpus.RunPackage; not part of this AST-backend story.
- Exact golden parity is US-042 (gensym names differ).
