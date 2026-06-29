# SEAM-002 Quality Audit — FuncMod & ChanDir iota → goal enum

Commit audited: `2c81404`
Scope: correctness/quality of the enum conversion and the `match` ports across
`selfhost/{ast,sema,backend,parser}`, the zero-value invariant, and the
`internal/ast` test relocation.

## Verdict

The code change is **correct and behavior-equivalent**. Every required code
property holds. The one finding is a documentation defect: the commit added a new
SEAM-002 ledger entry but left the now-false prior refusal block in
`DECISIONS.md`, so the ledger self-contradicts.

## Findings

### MAJOR — Stale, contradictory refusal entry left in DECISIONS.md

`DECISIONS.md:2063` still carries the header **"`FuncMod` and `ChanDir` stay iota
`const` blocks — NOT goal `enum`s"** (Kind: refusal), the decision this very
commit reverses. Its body now states things that are affirmatively false against
the landed code, e.g. (lines 2075–2076) *"Enum values are boxed interface values,
not comparable integers, so these comparisons would not survive the conversion"*
and (2081–2082) *"Sealing these into enums makes each a §9 closed-enum-plain-switch
compile error in `sema`"* — both contradicted by the working `match` ports in
`question.goal`/`convert.goal`/`resolve.goal`/`emit.goal`.

The new entry (`DECISIONS.md:51`, "SEAM-002 — FuncMod & ChanDir iota → goal enum")
is excellent and complete, but it does not retract or mark-superseded the old
block. Its only "supersedes" note (added line 90–92) is scoped to the *token.Kind*
point, not to clearing the FuncMod/ChanDir refusal. A reader landing on line 2063
gets the opposite of the truth.

This does not affect build or runtime behavior, and the literal AC ("DECISIONS.md
records the conversion and the token.Kind refusal") is met by the new entry —
hence MAJOR, not CRITICAL. Fix: delete the 2063 block or annotate it
`SUPERSEDED by SEAM-002`.

### No other findings

All substantive code checks passed (details below). Nothing CRITICAL; no other
MAJOR or MINOR issues found.

## Verification detail (all PASS)

**Exhaustiveness — all 8 `match` sites cover all 3 variants, no wildcard `_`.**
Confirmed by reading each site and by `grep '_ =>' selfhost/` returning nothing.
- `ast.goal:170` `FuncDecl.Pos` `notPlain` — FuncPlain/FuncFrom/FuncDerive
- `sema/question.goal:213` `plainResultFuncs` `plain` — all 3
- `sema/convert.goal:37` `CheckConvert` `isDerive` — all 3
- `sema/resolve.goal:217` `resolveFunc` `isConv` — all 3
- `sema/resolve.goal:463` `typeString` ChanDir — RecvOnly/SendOnly/SendRecv
- `backend/emit.goal:367` `funcDecl` `isDerive` — all 3
- `backend/emit.goal:2308` `chanType` ChanDir — all 3

**Behavior equivalence — confirmed for each converted construct.**
- `question.goal:208–217` — the `!ok` type-assert guard is split out (`if !ok {
  continue }`) BEFORE the `match fn.Mod`, preserving the original `!ok || …`
  short-circuit. Critical: without the split, a non-FuncDecl decl yields nil `fn`
  and `match fn.Mod` would nil-fault. Correct.
- `convert.goal:31–42` — same `!ok` split-then-match pattern; equivalent to the
  original `if !ok || fd.Mod != ast.FuncDerive`.
- `resolve.goal:217–226` — the tagless `switch { case isConv: … case d.Recv != nil:
  … }` is retained per FR-3; only the inner `==`/`||` became the match-bound
  `isConv` bool. Correct.
- `resolve.goal:463` / `emit.goal:2308` — the former `default` ("chan "/`e.p("chan
  ")`) body is re-homed under the explicit `ast.ChanDir.SendRecv` arm. For the
  three valid directions this is identical output. Correct.
- `emit.goal:361` (old) — the `default: e.fail("unsupported func modifier")` arm
  was unreachable for the three valid variants and is dropped; FuncPlain/FuncFrom
  still fall through to the ordinary emit path, FuncDerive still routes to
  `e.deriveDecl(d); return`. Equivalent. (Spec §Error Handling explicitly blesses
  this removal.)

**Zero-value invariant — every constructor sets Mod/Dir explicitly.**
`grep` across `selfhost/` finds exactly one constructor each:
- `&ast.FuncDecl{Mod: ast.FuncMod.FuncPlain}` at `parser.goal:367`. The only other
  producer, `parseModFuncDecl` (`goal_stmt.goal:47`), reuses `parseFuncDecl()`
  then overwrites `fd.Mod = mod` — so no FuncDecl ever carries a nil Mod.
- `&ast.ChanType{… Dir: ast.ChanDir.SendRecv}` at `parser.goal:513`, later
  overwritten to RecvOnly/SendOnly as parsed.
No bare/unqualified spellings (`ast.FuncPlain`, `ast.SendRecv`) remain; all refs
are qualified `ast.FuncMod.*` / `ast.ChanDir.*` per FR-4.

**Construction syntax (FR-4).** Same-package site (`ast.goal:171`) uses
`FuncMod.FuncPlain` (no `ast.` prefix); cross-package sites use the qualified
`ast.FuncMod.*` / `ast.ChanDir.*`. Correct.

**No stray consumers.** A tree-wide grep for `==`/`!=`/`switch`/`case` over
`.Mod`/`.Dir` or the variant names finds only the converted `match` sites; the
remaining `==`/`switch` hits are all `sema.Mode` (SEAM-003, out of scope), not
FuncMod/ChanDir.

**internal/ast test relocation (AC "task check green").**
- `internal/ast/funcmod_test.go` re-creates `collect` and `assertChildren` as
  LOCAL closures inside `TestFuncDeclModifier` (confirmed they are function-local
  closures in `ast_test.go`, not package-level), and references the package-level
  `collector` type (`ast_test.go:12`). No redeclaration conflict.
- The relocated assertions are byte-for-byte the ones removed from
  `ast_test.go` (verified against the diff). The test asserts what it claims
  (Walk descent, `Mod` value, and `Pos()`/`ModPos` behavior for from/derive/plain).
- `go vet ./internal/ast/` exits 0; `go test ./internal/ast/ -run
  'TestFuncDeclModifier|TestWalkGoalDeclChildren'` → PASS.

**Doc text preservation (FR-1/FR-2).** The per-variant `//` comments and the
type-level doc comments are preserved on both enums in `goal_decl.goal` and
`ast.goal`.

## Assumptions

- I did **not** re-run `task fixpoint`, the full `task check`, or the corpus
  behavioral tier. I verified `internal/ast` compiles/vets/tests cleanly and
  reviewed the `.goal` diffs for equivalence; the fixpoint-OK / check-green /
  corpus-unchanged claims are taken from the commit message and treated as
  out-of-scope for this static quality pass.
- I assume the goal `enum`/`match` lowering capabilities (SEAM-CAP, SEAM-CAP-2)
  behave as the commit and DECISIONS.md describe (cross-package enum visibility,
  bare-construction lowering, exhaustive match with run-time fault on a nil
  scrutinee). The exhaustiveness/run-time-fault semantics the audit relies on are
  the ones stated in the spec's Error Handling section.
- I assume the spec's intent is that DECISIONS.md be a non-contradictory ledger;
  the literal AC text only requires that the conversion + token.Kind refusal be
  *recorded*, which is satisfied. The MAJOR finding is graded against ledger
  integrity, not the literal AC.
