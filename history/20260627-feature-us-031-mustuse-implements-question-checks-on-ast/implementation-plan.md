# Implementation Plan — US-031

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/sema/mustuse.go` | `CheckMustUse(*ast.File, *Info)` — feature 03 must-use over the AST. |
| `internal/sema/implements.go` | `CheckImplements(*ast.File, *Info)` — feature 07 interface satisfaction over the AST, plus `requiredMethods` folding embedded interfaces. |
| `internal/sema/question.go` | `CheckQuestion` (feature 05 open-E `?` arity/refusal) and `CheckClosed` (feature 06 closed-E `?` From-totality + `Result.Err` closedness). |
| `internal/sema/mustuse_test.go`, `implements_test.go`, `question_test.go` | sema-package unit tests for each check. |
| `internal/corpus/sema_question_test.go` | corpus runner test driving 03-result, 06-error-e, 07-implements through `SemaCheck`. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/sema/sema.go` | Add `Interfaces map[string][]Method` and `EmbeddedIfaces map[string][]string` to `Info`. |
| `internal/sema/resolve.go` | Initialize the two new maps in `Resolve`; record in-file interface method sets + embeddings in `resolveTypeDecl` (and from `SealedInterfaceDecl`? no — sealed stays trivially-met). Build interface `Method` with the same `Sig` normalization as `resolveMethod`. |
| `internal/sema/check.go` | Wire `CheckMustUse`, `CheckImplements`, `CheckQuestion`, `CheckClosed` into `sema.Check`. |

## Package Structure

```
internal/sema/
  sema.go        (+Interfaces, +EmbeddedIfaces on Info)
  resolve.go     (+interface resolution)
  check.go       (+4 checks wired into Check)
  mustuse.go     (new)
  implements.go  (new)
  question.go    (new)
  *_test.go      (new unit tests)
internal/corpus/
  sema_question_test.go (new runner test)
```

## Dependency Graph

1. Extend `Info` (sema.go) + interface resolution (resolve.go) — foundation for implements.
2. `mustuse.go` — depends only on existing `Info.FuncSignatures` + the `visitorFunc` adapter in check.go.
3. `implements.go` — depends on (1).
4. `question.go` (`CheckQuestion` + `CheckClosed`) — depends on existing `FuncSignatures`/`Enums`/`FromRegistry`.
5. Wire into `sema.Check` (check.go) — depends on 2,3,4.
6. Unit tests + corpus runner test — depend on 5.

## Reused patterns

- `visitorFunc` ast.Walk adapter, `exprName`, `quoteVariants`, `plural`, `pronoun`,
  `Diagnostic` spine — all already in `internal/sema/check.go` / `fields.go`.
- `funcSig`, `paramTypeListFL`, `paramTypeList`, `joinTypes`, `receiverType`,
  `typeString` — already in `resolve.go`; interface methods reuse them.
- Corpus runner test mirrors `internal/corpus/sema_fields_test.go` (per-dir manifest
  walk through `RunCheck` + `CheckerFunc(SemaCheck)`).

## Risk / regression notes

- `sema.Check` now runs the new checks for ALL `SemaCheck` callers, including the
  existing 02-match and 08-no-zero-value runners. Verified by inspection: no 02/08
  fixture has a dropped Result statement, a struct `implements` clause, or a `?` in a
  Result function, so no new Error fires there. `go test ./...` confirms.
- `ImplementsClause.Type` is a single type name (one interface per struct in the
  corpus); handle the single-interface case.
