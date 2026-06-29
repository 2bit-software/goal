# SEAM-002 — Buildability Audit (implementation-plan.md)

Scope: can an agent execute the plan step-by-step and reach `task check` / `task
build` / `task fixpoint` green? Verified the plan's file paths, line numbers,
edit/dependency order, and the match/construction forms against the real repo at
`/Users/morgan/Projects/personal/goal`, including the enabling capabilities
(`internal/backend/lower.go` `matchQualifier`/`enumRef`/`enumOf`,
`internal/sema/foreign.go` `goalForeignDecls`) and the SEAM-CAP/CAP-2 fixtures.

## Verdict

**Buildable.** No CRITICAL or MAJOR defects. Every lowering form the plan relies
on is implemented in BOTH the internal (Go) backend and the selfhost (.goal)
backend, and is proven by existing fixtures; the construction-invariant reasoning
is sound; the atomic-single-commit requirement is correctly stated; all cited
line numbers are accurate. Two MINOR wording hazards in the plan/research could
trip a literal executor but are self-correcting.

---

## Verification of the plan's factual claims (all confirmed)

### File paths and line numbers — ACCURATE
- `selfhost/ast/goal_decl.goal:20` `type FuncMod int` + iota (`FuncPlain`/`FuncFrom`/`FuncDerive`). Confirmed.
- `selfhost/ast/ast.goal:672` `type ChanDir int` + iota; `SendRecv`/`SendOnly`/`RecvOnly` at 676–680. Confirmed (research said "ast.goal:672"; exact).
- Same-package site `selfhost/ast/ast.goal:170` — `if d.Mod != FuncPlain && d.ModPos != (token.Pos{})` inside `FuncDecl.Pos()`. Confirmed at line 170.
- `selfhost/sema/question.goal:210` — `if !ok || fn.Mod != ast.FuncPlain || ...`. Confirmed.
- `selfhost/sema/resolve.goal:217-218` — tagless `switch {` / `case d.Mod == ast.FuncFrom || d.Mod == ast.FuncDerive:`. Confirmed.
- `selfhost/sema/resolve.goal:458` — `switch x.Dir {` over ChanDir (RecvOnly/SendOnly/default). Confirmed.
- `selfhost/sema/convert.goal:34` — `if !ok || fd.Mod != ast.FuncDerive`. Confirmed.
- `selfhost/backend/emit.goal:361` — `switch d.Mod {`; `case ast.FuncPlain, ast.FuncFrom:`/`case ast.FuncDerive:`/`default: e.fail("unsupported func modifier %v", d.Mod)`. Confirmed at 361–374.
- `selfhost/backend/emit.goal:2309` — `switch x.Dir {` chanType. Confirmed.
- Construction: `parser.goal:226/229` (`p.parseModFuncDecl(ast.FuncFrom/.FuncDerive)`), `parser.goal:511/515/520` (`Dir: ast.SendRecv`, `c.Dir = ast.RecvOnly/.SendOnly`), `parser.goal:365` (`fd := &ast.FuncDecl{}`). All confirmed.
- Reference-only no-change sites: `goal_stmt.goal:47` param `mod ast.FuncMod`, `goal_stmt.goal:50` `fd.Mod = mod`, `ast.goal:161` field `Mod FuncMod`. Confirmed; correctly excluded.

### Construction-site completeness — CONFIRMED COMPLETE
`grep` for `FuncDecl{`/`ChanType{` across `selfhost/` returns ONLY
`parser.goal:365` and `parser.goal:511`. There are no other (e.g. backend-
synthesized) FuncDecl/ChanType literals. The FR-6 zero-value invariant (enum zero
is `nil`, not the former iota zero) is therefore closed by exactly the two
construction owners the plan names: `parser.goal:365` gets an explicit
`Mod: ast.FuncMod.FuncPlain`, and `parser.goal:511` already sets `Dir` explicitly.
No FuncDecl/ChanType can carry a `nil` modifier/direction into a `match`.

### Lowering forms vs. SEAM-CAP / SEAM-CAP-2 — ALL SUPPORTED, PROVEN BY FIXTURES
- `matchQualifier` (`internal/backend/lower.go:188`) handles both `*ast.Ident`
  (same-package `FuncMod.FuncPlain`, used at `ast.goal:170`) and
  `*ast.SelectorExpr` (cross-package `ast.FuncMod.FuncFrom`). Confirmed.
- `enumRef` (`lower.go:244`) is the construction-side twin, same Ident /
  `pkg.Enum` key shapes. Confirmed.
- Cross-package value-position match (`return match l {...}` with
  `light.Light.On => ...`): `testdata/package/cross-pkg-enum/use.goal` `label`.
- Cross-package statement-position match (`match l { ... => println(...) }`):
  same fixture, `describe`.
- Bare cross-package construction lowering to `pkg.Enum(pkg.Enum_Variant{})`:
  `internal/backend/testdata/goalenum/use/use.goal` `pick` returns `mood.Mood.Happy`.
- Assignment-position `name := match` with **bool** inference (the plan uses this
  in 5 sites: question `plain`, resolve `isConv`, convert `isDerive`, ast
  `notPlain`, emit `isDerive`): `inferMatchType`/`armBodyType` map a `true`/`false`
  arm to `"bool"` in BOTH `internal/backend/emit.go:1594` and
  `selfhost/backend/emit.goal` (`armBodyType` returns `("bool", true)`; error
  text at `emit.goal:1541` lists "bool" as an inferable result type). Fixpoint-safe.
- `goalForeignDecls` (`internal/sema/foreign.go:301`) projects exported sibling-
  `.goal` enums into `info.Enums`, so a cross-package `ast.FuncMod`/`ast.ChanDir`
  match/construction resolves during `goal build ./selfhost`. Confirmed present.

All forms exist in the selfhost tree too (`selfhost/backend/lower.goal`,
`emit.goal` `matchQualifier`/`enumMatch`/`tryAssignMatch`/`inferMatchType`), so
stage1==stage2 fixpoint is achievable.

### Edit/dependency order and atomicity — CORRECT
The plan's claim (lines 50–53) that "a plain switch over a now-sealed enum is a
§9 compile error, so all of 1–4 must land together" is sound: once `FuncMod`
becomes an enum, the bare constant `ast.FuncPlain` ceases to exist (it becomes
the `ast.FuncMod_FuncPlain{}` variant struct), so any unconverted `==`/`switch`
site references an undefined identifier and fails to compile. A single atomic
commit covering ast + parser + sema + backend + the test split is required and is
what the plan prescribes. No forward references; no contradictions in ordering.

### Test-divergence handling — CORRECT
- Only two `_test.go` files reference the symbols: `internal/ast/ast_test.go`
  (port-gated at `internal/selfhost/port_test.go:142`,
  `BuildAndTest("selfhost/ast", ..., []string{"../ast/ast_test.go"}, deps)` —
  confirmed at line 142) and `internal/parser/goal_stmt_test.go` (NOT in any
  `BuildAndTest` slice; the parser gate uses `parser_test.go` at line 232).
- The FuncMod block is `internal/ast/ast_test.go:247–284` inside
  `TestWalkGoalDeclChildren` (confirmed). Relocating it to a new internal-only
  `internal/ast/funcmod_test.go` (kept OUT of the port-gate slice) leaves
  `ast_test.go` symbol-free so it compiles against the enum-transpiled
  selfhost/ast. `internal/ast/{ast.go,goal_decl.go}` stay Go iota, so
  `goal_stmt_test.go` (internal-vs-internal, both iota) still compiles
  unchanged. Analysis is accurate.

---

## Findings

### MINOR-1 — "reuse the same-package collect/assertChildren helpers" is inaccurate
`technical-requirements-research.md:120-121` ("a new internal-only file
`internal/ast/funcmod_test.go` (package `ast`, reusing the same-package
`collect`/`assertChildren` helpers)") is misleading: in `ast_test.go`, `collect`
and `assertChildren` are **function-local closures** defined inside
`TestWalkGoalDeclChildren` (`internal/ast/ast_test.go:188` `collect := func(...)`,
`:194` `assertChildren := func(...)`), NOT package-level helpers. Only the
`collector` type (`ast_test.go:12`) is package-level. The new file must therefore
**redefine** `collect`/`assertChildren` inside its own test function (and import
`token` for `token.Pos`). A literal executor that calls a non-existent
package-level `collect(...)` would hit a compile error. Self-correcting once the
agent re-creates the closures, but the wording should say "re-create", not "reuse".

### MINOR-2 — `///` vs `//` doc-comment hazard in FR-1/FR-2
`business-spec.md:35-37` (FR-1) and FR-2 require the existing per-constant
"`///`/`//` doc text … preserved as comments attached to the enum / its variants."
The parser (`internal/parser/parser.go:62-69`) **strips ordinary `//` COMMENT
tokens** but **keeps `///` DOC_COMMENT tokens in the stream**, and `parseVariant`
(`internal/parser/goal_decl.go:47`) calls `p.ident()` directly with NO
DOC_COMMENT handling between variants. The existing per-constant docs are all
ordinary `//` (`goal_decl.goal:23/25/27`, `ast.goal:675/677/679`), so keeping
them as `//` is safe (they are stripped; behavior unaffected). But if an agent
"upgrades" them to `///` to make them attach as doc, a `///` token between
variants will reach `p.ident()` and **fail the parse**, breaking `task build`.
Recommendation: explicitly instruct keeping the variant docs as `//`.
(The FR-1 example block itself shows variants with no comments, which is also
safe.) Low residual risk given the source is already `//`.

### MINOR-3 — no dedicated fixture exercises bool-arm `:= match`
The plan leans on assignment-position `name := match` with **bool**-valued arms
in five sites. The capability is implemented in both trees (see above), but the
only `:= match` value-inference fixture (`testdata/match_infer_value.goal`)
exercises enum→enum and enum→string, not enum→bool. The code path
(`armBodyType` → `"bool"`) is present and the corpus/fixpoint gates would catch a
regression, so this is informational, not a blocker. No new fixture is required
by the plan and none is needed for correctness.

### Note (not a plan defect) — audit-prompt fixture path
The task referenced `internal/backend/testdata/package/cross-pkg-enum/use.goal`;
the actual cross-package match fixture lives at
`testdata/package/cross-pkg-enum/use.goal` (repo root), importing
`goal/internal/backend/testdata/extenum`. It exists and proves the value- and
statement-position cross-package match forms. The plan does not cite a wrong path.

---

## Could an agent execute each step and reach green?

Yes. Concretely:
1. `selfhost/ast`: rewrite the two iota blocks as `enum FuncMod {...}` / `enum
   ChanDir {...}` (variant docs as `//`, see MINOR-2); convert `Pos()` at
   `ast.goal:170` to `notPlain := match d.Mod { FuncMod.FuncPlain => false; ... }`.
2. `selfhost/parser`: qualify the five construction sites as
   `ast.FuncMod.*`/`ast.ChanDir.*` and add `Mod: ast.FuncMod.FuncPlain` at line 365.
3. `selfhost/sema` + `selfhost/backend`: convert the `==`/`!=`/`switch` consumers
   to match per the per-site target forms (all forms supported in both trees).
4. Split `internal/ast/ast_test.go` (remove 247–284) and add
   `internal/ast/funcmod_test.go` (re-create the closures — MINOR-1).
5. `DECISIONS.md` + `prd.json` + `progress.txt`.

All in one atomic commit. `task check` (incl. corpus behavioral gate + port
gates), `task build`, and `task fixpoint` should pass because: the construction
invariant prevents `nil` reaching any `match`; every match/construction form is
implemented in both the internal and selfhost backends and proven by fixtures;
and the only port-gated symbol reference (`ast_test.go`) is made symbol-free.

## Assumptions

- The bootstrap toolchain that compiles `selfhost/*.goal` is the internal Go
  compiler (`internal/{lexer,parser,sema,backend}`), and `task fixpoint`
  recompiles selfhost with the resulting selfhost binary; therefore every form
  must be supported in BOTH `internal/backend` and `selfhost/backend`. I verified
  both. If the build wiring differs, the dual-tree verification still covers it.
- `task check`/`task build`/`task fixpoint` behave as the spec describes (I did
  not execute them — read-only audit). The PostToolUse `task check` hook will go
  red mid-edit; only the final atomic state is asserted, per the plan.
- Per-variant `//` comments placed between enum variants are stripped by the
  parser and do not affect parsing or emitted behavior (verified
  `parser.go:62-69`); I did not run a parse over the exact converted text.
- "Doc text preserved" (FR-1/FR-2) is satisfied by retaining the existing `//`
  comment text in source; the spec does not require the comments to survive into
  emitted Go (they did not under iota either).
- Line numbers cited reflect the repo HEAD at audit time (branch `main`,
  `bcf5593`); an atomic edit shifts later line numbers, which is expected.
