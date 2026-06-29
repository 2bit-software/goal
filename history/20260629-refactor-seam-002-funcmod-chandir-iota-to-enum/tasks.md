# Implementation Tasks — SEAM-002

> ATOMIC SEAM. Unlike a normal feature, the sub-steps below are NOT individually
> green: once the iota constants become enum variants, every unconverted
> `==`/`switch` references an undefined identifier or a sealed value (a §9
> compile error), so the tree is red until the LAST edit lands. The
> PostToolUse `task check` hook will report transient failures between edits —
> expected; only the final atomic state is gated. Everything lands in ONE commit.
> Verification (Task 6) runs the real gates after all edits.

## Task 1: Define FuncMod and ChanDir as goal enums
**Status**: completed
**Files**: `selfhost/ast/goal_decl.goal`, `selfhost/ast/ast.goal`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2; and FR-3 same-package site (ast.goal:170)
**Verify**: edits applied (full gate deferred to Task 6)

### Instructions
- `goal_decl.goal`: replace `type FuncMod int` + the `const (...)` iota block with
  `enum FuncMod { FuncPlain; FuncFrom; FuncDerive }` (one variant per line, no
  trailing punctuation per DECISIONS §01-enums). Keep the existing `//` doc text;
  do NOT convert to `///` (a `///` between variants fails the parse).
- `ast.goal`: same for `enum ChanDir { SendRecv; SendOnly; RecvOnly }`.
- `ast.goal:170` (same-package consumer, in `FuncDecl.Pos()`): replace
  `if d.Mod != FuncPlain && d.ModPos != (token.Pos{})` with a match-bound bool:
  `notPlain := match d.Mod { FuncMod.FuncPlain => false; FuncMod.FuncFrom => true; FuncMod.FuncDerive => true }` then `if notPlain && d.ModPos != (token.Pos{}) { return d.ModPos }`.

## Task 2: Parser — qualified construction + zero-value fix
**Status**: completed
**Files**: `selfhost/parser/parser.goal`
**Depends on**: Task 1
**Spec coverage**: FR-4, FR-6 (zero-value invariant)
**Verify**: edits applied (full gate deferred to Task 6)

### Instructions
- `parser.goal:365`: `fd := &ast.FuncDecl{}` -> `fd := &ast.FuncDecl{Mod: ast.FuncMod.FuncPlain}`.
- `parser.goal:226/229`: `ast.FuncFrom`/`ast.FuncDerive` -> `ast.FuncMod.FuncFrom`/`ast.FuncMod.FuncDerive`.
- `parser.goal:511`: `Dir: ast.SendRecv` -> `Dir: ast.ChanDir.SendRecv`.
- `parser.goal:515/520`: `c.Dir = ast.RecvOnly`/`ast.SendOnly` -> `ast.ChanDir.RecvOnly`/`ast.ChanDir.SendOnly`.
- `goal_stmt.goal`: NO change (param type `ast.FuncMod` + `fd.Mod = mod` are fine).

## Task 3: sema — convert FuncMod/ChanDir consumers to match
**Status**: completed
**Files**: `selfhost/sema/question.goal`, `selfhost/sema/resolve.goal`, `selfhost/sema/convert.goal`
**Depends on**: Task 1
**Spec coverage**: FR-3
**Verify**: edits applied (full gate deferred to Task 6)

### Instructions
- `question.goal:210`: split the `!ok` type-assert guard out FIRST
  (`fn, ok := d.(*ast.FuncDecl); if !ok { continue }`), then
  `plain := match fn.Mod { ast.FuncMod.FuncPlain => true; ast.FuncMod.FuncFrom => false; ast.FuncMod.FuncDerive => false }`,
  then `if !plain || fn.Recv != nil || fn.Name == nil || fn.Type == nil || fn.Body == nil { continue }`.
- `resolve.goal:218`: keep the tagless `switch {}`; replace the first case test
  with a match-bound bool `isConv := match d.Mod { ast.FuncMod.FuncFrom => true; ast.FuncMod.FuncDerive => true; ast.FuncMod.FuncPlain => false }`; `case isConv:`.
- `resolve.goal:458` (typeString, `case *ast.ChanType:`): replace `switch x.Dir`
  with value-position `match`; former `default` body maps to the explicit
  `ast.ChanDir.SendRecv => "chan " + typeString(x.Value)` arm; keep RecvOnly/SendOnly bodies.
- `convert.goal:34`: split `!ok` out, then `isDerive := match fd.Mod { ast.FuncMod.FuncDerive => true; ast.FuncMod.FuncFrom => false; ast.FuncMod.FuncPlain => false }`; `if !isDerive { continue }`.

## Task 4: backend — convert FuncMod/ChanDir consumers to match
**Status**: completed
**Files**: `selfhost/backend/emit.goal`
**Depends on**: Task 1
**Spec coverage**: FR-3
**Verify**: edits applied (full gate deferred to Task 6)

### Instructions
- `emit.goal:361` (funcDecl): replace the control `switch d.Mod` with
  `isDerive := match d.Mod { ast.FuncMod.FuncPlain => false; ast.FuncMod.FuncFrom => false; ast.FuncMod.FuncDerive => true }` then
  `if isDerive { e.deriveDecl(d); return }`; FuncPlain/FuncFrom fall through to
  ordinary emission. Drop the `default: e.fail(...)` arm. Preserve the
  explanatory comments.
- `emit.goal:2309` (chanType): replace `switch x.Dir` with statement-position
  `match`; former `default` body `e.p("chan ")` becomes the
  `ast.ChanDir.SendRecv` arm; keep RecvOnly/SendOnly arms. Then `e.expr(x.Value)`.

## Task 5: Relocate the FuncMod oracle test (test-divergence resolution)
**Status**: completed
**Files**: `internal/ast/ast_test.go`, `internal/ast/funcmod_test.go` (new)
**Depends on**: (none — test-only; can land with the rest)
**Spec coverage**: AC "task check green after relocation"
**Verify**: `go test ./internal/ast/` green against Go-iota internal/ast
(but commit verification is Task 6)

### Instructions
- Remove the from/derive/plain `FuncDecl` block (~lines 247-284) from
  `TestWalkGoalDeclChildren` in `internal/ast/ast_test.go`, leaving the file with
  NO `FuncMod`/`FuncPlain`/`FuncFrom`/`FuncDerive`/`ChanDir` references.
- Create `internal/ast/funcmod_test.go` (`package ast`) with a new
  `TestFuncDeclModifier(t *testing.T)` carrying those assertions. RE-CREATE the
  `collect`/`assertChildren` closures locally (they are function-local in
  ast_test.go, not package-level) and import `goal/internal/token`.
- internal/ast itself stays Go iota; do NOT add funcmod_test.go to
  `internal/selfhost/port_test.go:142`.

## Task 6: Document + verify + finalize
**Status**: completed
**Files**: `DECISIONS.md`, `progress.txt`, `prd.json`
**Depends on**: Tasks 1-5
**Spec coverage**: FR-5, all gate ACs
**Verify**: `task check` && `task build` && `task fixpoint` (FIXPOINT OK);
corpus behavioral tier (`TestASTEngineWholeCorpusBehavioralGate`) unchanged.

### Instructions
- Run the three gates. Fix any failure (red is expected only until all of
  Tasks 1-5 are applied).
- DECISIONS.md: add a SEAM-002 section recording the FuncMod/ChanDir enum
  conversion AND the token.Kind iota refusal (numeric-identity: kindNames[k]
  indexing, literalBeg<k<literalEnd range arithmetic, contiguous numbering) per
  AC-1's escape hatch.
- After gates are green: commit (single atomic commit), set SEAM-002
  `passes: true` in prd.json, append the SEAM-002 entry to progress.txt.
