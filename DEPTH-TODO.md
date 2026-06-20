# DEPTH-TODO — Phase B: go/types depth checks

Phase B of `ROADMAP_TO_GOAL.md`. Goal: **close the checker's type-information-dependent
deferrals by transpiling to Go and asking `go/types`** — the must-use stored-then-dropped
class (refused), 07 signature identity, 12 conversion recursion, value-position `match`,
and the residual cross-package 02/06/08 cases. This is the depth track you prioritized; it
sits directly on Phase A's substrate.

Loop-ready in the `CHECKER-TODO.md`/`BUILD-MODEL-TODO.md` style: a **thesis**, the **claim
to prove by spike**, a **dependency-ordered unit queue**, and the **open decisions**.

---

## Thesis — how the depth checker works

The lexical checker (`internal/check`) runs on the **original** goal source, before
lowering, and defers anything it cannot resolve by name. Phase B adds a **second checker
stage** that runs on the **lowered Go** (Phase A's `TranspilePackage` output) and answers
those deferrals with real type information — exactly the roadmap's "lean on the Go
toolchain" applied to the checker.

Four facts make this tractable and on-thesis:

### Claim B1 — `go/types` is the type checker; we don't build one (stdlib only)
goal is **zero-dependency** (`go.mod` has no requires; tests are stdlib only). Phase B keeps
that: it uses only stdlib `go/parser`, `go/types`, `go/importer`, `go/token`, `go/ast`. There
is **no `x/tools/go/ssa`** — so flow facts (a Result bound then never read) come from
`types.Info.Defs`/`Uses` plus `go/ast` walks, not from an SSA library. The hard type
questions (identity, assignability, interface satisfaction) are `types.Identical`,
`types.AssignableTo`, `types.Implements`.

### Claim B2 — positions map back to `.goal` for FREE
The lowered Go already carries `//line file.goal:N` directives (Phase A U5). `go/scanner`,
`go/parser`, and the `token.FileSet` **honor `//line` natively**, so `fset.Position(node.Pos())`
on the parsed lowered Go reports the **`.goal`** file and line. The depth checker therefore
emits goal-located diagnostics with no new mapping layer — the same directives that map
compiler errors map our analysis. (Within-body precision inherits U5's per-declaration
granularity; the per-statement upgrade is still future work.)

### Claim B3 — error-tolerant type-checking
A buggy goal program may not fully type-check, but the depth checks still need partial type
info. `types.Config{Error: collect}` (a non-nil Error handler) makes the checker **collect
errors and keep going** instead of stopping at the first, so `Info` is populated as far as
possible. Genuine Go type errors are themselves diagnostics worth surfacing (already
goal-mapped via B2).

### Claim B4 — the goal tables tell the depth check what to ask
Each deferred guarantee is a typed query *scoped by a goal fact*: "is this var, which goal
typed as a `Result`, ever read?" / "does T, which goal said `implements I`, satisfy I?". The
name-keyed `analyze.Tables` (now package-merged, U2) name the constructs; `go/types` answers
the type question about them. The two compose: tables locate, `go/types` decides.

### What the thesis buys / where the depth checker lives
A new stage `internal/typecheck` (separate from the lexical `internal/check`): it takes a
package's lowered Go + the merged tables, loads it with `go/types`, and runs per-guarantee
typed checks that return goal-located diagnostics. `goal check` runs **both** stages
(lexical pre-lowering + typed post-lowering) and renders their diagnostics together.

---

## Claim to prove first (SPIKE — before the unit queue)

- [x] **SPIKE-B1 — load + query + map. PASSED 2026-06-20.** Transpiled a goal package
  (`Speaker` interface, `Dog struct implements Speaker`, a non-implementer `Rock`, a used local
  `msg`), parsed the lowered Go with stdlib `go/parser`, and type-checked it with stdlib
  `go/types` under an error-collecting `Config{Error: collect}`. **All three held:**
  1. **Load:** check completed with 0 collected errors; `Info.Defs=11`, `Info.Uses=10` populated.
  2. **Query:** `types.Implements(Dog, Speaker)=true`, `(Rock, Speaker)=false` (real identity
     discrimination); `Info.Defs`/`Uses` correctly reported the local `msg` as used (must-use
     primitive available with no SSA).
  3. **Map:** `fset.Position(Dog) = zoo.goal:7` — the U5 `//line` directives carry through
     `go/parser`→`go/types`, **line-accurate**, so depth diagnostics are goal-located for free.
  > Outcome: no fallback needed; the name-table mapping path is unnecessary. Stdlib-only (`go/parser`,
  > `go/types`, `go/importer`, `go/token`) confirmed sufficient — zero-dep holds. Queue stands.

---

## Unit queue (dependency-ordered; one per iteration after the spike)

- [ ] **B1 — `go/types` harness (`internal/typecheck`).** Load a package's lowered Go
  error-tolerantly; expose `{fset, pkg, info, files}` plus helpers to (a) get the `.goal`
  position of a node and (b) resolve a generated object back to its goal name. The foundation
  every check below uses. *Depends on SPIKE-B1, Phase A U4.*

- [ ] **B2 — 07 implements via real type identity.** Replace the lexical check's
  textual-after-normalization signature comparison with `types.Implements`/`types.Identical`,
  killing the alias-spelled-differently false-mismatch (the documented §07 lexical ceiling).
  *Depends on B1.*

- [ ] **B3 — 03 must-use, stored-then-dropped (the refused class).** A `Result`/`Option`-typed
  local (goal-typed per the tables) that is `Info.Defs`-defined but never in `Info.Uses` — and
  not explicitly discarded — is an unused result → Error. Lifts the explicit refusal in
  DECISIONS §03. *Depends on B1.*

- [ ] **B4 — 12 conversion recursion.** Resolve map/`Option`/pointer/nested-struct field types
  via `go/types` and check derive totality through them; close the out-of-package types the
  lexical check deferred. *Depends on B1.*

- [ ] **B5 — value-position untyped `x := match` (lowering completion).** With the result type
  now inferable via `go/types`, complete the deferred value-position `match` / stored
  Result/Option sum-encoding lowering (§8.7). NOTE: this is a **lowering** gap in
  `internal/pass`/`pipeline`, fed by B1's type info — cross-cutting with the front-end, not a
  pure checker unit. *Depends on B1.*

- [ ] **B6 — promote residual 02/06/08 deferrals.** Where `go/types` resolves a case the
  lexical checker (even package-merged) still defers as a Warning, upgrade it to a type-backed
  Error; otherwise re-record as a genuine narrower residue. *Depends on B1–B4.*

**Done when:** each type-dependent deferral in `ROADMAP_TO_GOAL.md` §0 is either a type-backed
Error with a goal-located message, or a re-recorded narrower residue with reason; `goal check`
runs both stages.

---

## Open decisions (resolve at the named unit, not now)

- **Stage home & integration** (B1): `internal/typecheck` separate from `internal/check`;
  `goal check` runs lexical (pre-lowering) then typed (post-lowering) and merges diagnostics.
  Decide dedup when both stages flag the same thing (prefer the type-backed one).
- **Importer** (B1): `go/importer.Default()` (gc export data) vs `importer.ForCompiler(…,
  "source", …)`. Default is simplest for stdlib imports; revisit if a goal program imports
  third-party packages.
- **Cost / when it runs** (B1/U6): type-checking is heavier than lexing. Decide whether the
  typed stage runs on every `goal check` only, or also gates `goal build`. *Lean: `check` only.*
- **Diagnostic coordinates** (B1): depth diagnostics carry `token.Position` (already
  goal-mapped); render alongside the lexical checker's byte-offset diagnostics uniformly.
- **Per-statement precision**: B-stage findings inherit U5's per-declaration `//line` accuracy;
  the per-statement source map (needs pass Replacement journals) stays deferred unless a check
  proves it needs column-exact positions.

---

## Pointers
- `ROADMAP_TO_GOAL.md` §0 — the deferred-class table this phase closes.
- `BUILD-MODEL-TODO.md` — Phase A substrate (U4 package output, U5 `//line` map) this rests on.
- `DECISIONS.md` §03/§07/§12 — the refusals/assumptions naming the `go/types` ceiling.
- `internal/check/check.go` — the lexical stage; the typed stage mirrors its diagnostic shape.

_Status: thesis drafted 2026-06-20; SPIKE-B1 PASSED 2026-06-20 — stdlib go/types loads the
lowered package, answers type-identity + Defs/Uses, and positions map to `.goal` via `//line`
(line-accurate). Thesis proven, queue stands. Next: B1 (the typecheck harness)._
