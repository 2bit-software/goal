# Implementation Plan — SEAM-002

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/ast/funcmod_test.go` | Internal-only (Go-iota) home for the FuncMod Walk/Pos/Mod assertions relocated out of the port-gated `ast_test.go`, so the shared oracle compiles against both internal/ast (iota) and transpiled selfhost/ast (enum). |

### Modified Files
| File | Changes |
|------|---------|
| `selfhost/ast/goal_decl.goal` | `type FuncMod int` + iota block -> `enum FuncMod { FuncPlain; FuncFrom; FuncDerive }` (newline-separated tag variants; doc text preserved). |
| `selfhost/ast/ast.goal` | `type ChanDir int` + iota block -> `enum ChanDir { SendRecv; SendOnly; RecvOnly }`. Convert same-package `d.Mod != FuncPlain` (line ~170) to a match-bound bool. |
| `selfhost/sema/question.goal` | `fn.Mod != ast.FuncPlain` -> split `!ok` guard, match-bound `plain` bool. |
| `selfhost/sema/resolve.goal` | `d.Mod == ast.FuncFrom || ast.FuncDerive` -> match-bound `isConv` bool inside the tagless switch; `switch x.Dir {...}` -> value-position `match` (SendRecv = former default). |
| `selfhost/sema/convert.goal` | `fd.Mod != ast.FuncDerive` -> split `!ok`, match-bound `isDerive` bool. |
| `selfhost/backend/emit.goal` | `switch d.Mod {...}` funcDecl -> `isDerive := match...` + `if isDerive { e.deriveDecl(d); return }` (drop default fail); `switch x.Dir {...}` chanType -> statement-position `match`. |
| `selfhost/parser/parser.goal` | Construction spelling: `ast.FuncMod.FuncFrom/.FuncDerive`, `ast.ChanDir.SendRecv/.RecvOnly/.SendOnly`; `&ast.FuncDecl{}` -> `&ast.FuncDecl{Mod: ast.FuncMod.FuncPlain}`. |
| `internal/ast/ast_test.go` | Remove the FuncMod block (~247-284) from `TestWalkGoalDeclChildren` (relocated to funcmod_test.go). Leaves the file FuncMod/ChanDir-free. |
| `DECISIONS.md` | Record the FuncMod/ChanDir conversion + the token.Kind iota refusal (numeric-identity) under a SEAM-002 section. |
| `prd.json` | Set SEAM-002 `passes: true` (after all gates green; via skill, not hand-edit of intent). |
| `progress.txt` | Append the SEAM-002 entry; add any reusable pattern to the top block. |

No change: `selfhost/parser/goal_stmt.goal` (param type + plain assignment),
`selfhost/ast/ast.goal:161` field type, `selfhost/token/*` (Kind stays iota).
`internal/ast/{ast.go,goal_decl.go}` stay Go iota (bootstrap reference compiler).

## Package Structure

```
selfhost/
  ast/      <- enum defs (FuncMod, ChanDir) + same-pkg match
  sema/     <- match consumers (question, resolve, convert)
  backend/  <- match consumers (emit)
  parser/   <- qualified construction + zero-value fix
internal/ast/
  ast_test.go      <- FuncMod block removed
  funcmod_test.go  <- NEW (internal-only)
```

## Dependency Graph (build/edit order — atomic single commit)

1. `selfhost/ast` enum defs (FuncMod, ChanDir) + same-package ast.goal:170 match.
2. `selfhost/parser` construction spelling + zero-value fix (depends on 1's variants).
3. `selfhost/sema` + `selfhost/backend` match consumers (depend on 1).
4. `internal/ast/ast_test.go` split + `internal/ast/funcmod_test.go` (test-only).
5. `DECISIONS.md`.

All of 1-4 must land together — a plain switch over a now-sealed enum is a §9
compile error, so partial application leaves the tree red. (The PostToolUse
`task check` hook will report transient failures until the last edit lands;
only the final green state matters.)

## Interface Contracts

- `FuncMod` / `ChanDir` remain the type names; only the underlying form changes
  (iota int -> §8.1 sealed-interface enum). Field types `Mod FuncMod`,
  `Dir ChanDir` unchanged in source.
- Construction: `ast.FuncMod.FuncPlain` (lowers to `ast.FuncMod(ast.FuncMod_FuncPlain{})`).
- Match arm pattern: cross-package `ast.FuncMod.FuncFrom`; same-package `FuncMod.FuncFrom`.

## Integration Points

- Cross-package enum-match lowering: backend `matchQualifier` / `enumOf`
  (SEAM-CAP) — already present in internal/ + selfhost/.
- Cross-.goal-package enum visibility + bare-construction lowering:
  `goalForeignDecls` / `enumRef` (SEAM-CAP-2) — already present in both trees.
- Port gate: `internal/selfhost/port_test.go:142` compiles `../ast/ast_test.go`
  against transpiled selfhost/ast — keep funcmod_test.go OUT of that slice.

## Testing Strategy

- No new selfhost test fixtures: the conversion is exercised end-to-end by the
  existing self-host port gates (parser/sema/backend) + `task fixpoint`.
- `internal/ast/funcmod_test.go` preserves the relocated FuncMod Walk/Pos/Mod
  unit coverage for the Go-iota internal/ast under `task check`.
- Gates: `task check`, `task build`, `task fixpoint`; corpus behavioral tier
  (`internal/corpus.TestASTEngineWholeCorpusBehavioralGate`) unchanged.
