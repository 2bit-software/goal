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

- [x] **B1 — `go/types` harness (`internal/typecheck`).** Load a package's lowered Go
  error-tolerantly; expose `{fset, pkg, info, files}` plus helpers to (a) get the `.goal`
  position of a node and (b) resolve a generated object back to its goal name. The foundation
  every check below uses. *Depends on SPIKE-B1, Phase A U4.*
  - **Done:** `typecheck.Load(*project.Package) (*Package, error)` → `{Fset, Types, Info, Files,
    Tables, Errors}` + `GoalPos`/`Lookup`; error-collecting config (type errors non-fatal, mapped
    to `.goal`), `importer.Default()`. 2 tests (typed view + error tolerance). DECISIONS Phase B §B1.

- [x] **B2 — 07 implements via real type identity.** Replace the lexical check's
  textual-after-normalization signature comparison with `types.Implements`/`types.Identical`,
  killing the alias-spelled-differently false-mismatch (the documented §07 lexical ceiling).
  *Depends on B1.*
  - **Done:** `typecheck.CheckImplements` — locates clauses in source, verifies via
    `types.MissingMethod(*T, I)`; resolves in-package and qualified (`io.Writer`) interfaces. Alias
    false-mismatch eliminated; qualified interfaces checked not deferred. 4 tests. DECISIONS §B2.

- [x] **B3 — 03 must-use, stored-then-dropped (the refused class).** A `Result`/`Option`-typed
  local (goal-typed per the tables) that is `Info.Defs`-defined but never in `Info.Uses` — and
  not explicitly discarded — is an unused result → Error. Lifts the explicit refusal in
  DECISIONS §03. *Depends on B1.*
  - **Done:** `typecheck.CheckMustUse` covers the two flow subsets Go itself does NOT catch (the
    simple bound-then-unused local is already a Go "declared and not used" error): (1)
    `discarded-result-error` — `v, _ := f()` / `_, _ = f()` discarding the error of an open-E
    Result call (Error, at the `_`); (2) `dropped-stored-result` — an unexported Result/Option
    struct field never read via a selector (Error), with exported-field-never-read-in-package
    deferred as an `unresolved-dropped-field` Warning. Result fields read from go/types (the
    injected `Result` type), sidestepping the comma-split bug in `analyze.parseStructBody`; Option
    fields confirmed via the tables. 9 tests. Deferrals (selector-callee, open-E Result field,
    selector-write, interprocedural drop) recorded. DECISIONS §B3. No CLI wiring (same as B2).

- [x] **B4 — 12 conversion recursion. CLOSED by reassessment (2026-06-21) — substance delivered by
  the front-end (Lowering L1/L2), NO non-vacuous depth check exists.** Resolve map/`Option`/pointer/
  nested-struct field types and check derive totality through them; close out-of-package. *Depends on B1.*
  - **BLOCKED (2026-06-20):** the depth checker runs on *lowered* Go, but `derive.go` refused to lower
    these classes, so they never transpiled. Delivering B4 needed the **derive pass extended first**.
  - **Resolved (2026-06-21), after the front-end workstream lowered them (LOWERING-TODO L1/L2):** a
    B4 *depth check* is **vacuous as an error-producer**, so none was written (a vacuous check would be
    the "false signal of progress" the BLOCKED note warned against). Why: the derive pass lowers a
    conversion **only when every leaf resolves**, so a lowered conversion is **type-sound by
    construction** (`go/types` finds nothing — confirmed: both L1/L2 round-trip outputs type-check
    clean); a genuinely incomplete conversion **doesn't lower** (pass errors → `goal build` fails →
    the depth stage never sees it). The two real "type-backed not textual" wins B4 named —
    **alias-identity** (`type Name = string` assignable to `string`) and **out-of-package** — do not
    *lower* today (the pass is textual; out-of-package is the gated L5), so the depth stage can't
    observe them either. The residual is therefore **front-end lowering work, not a depth check**:
    alias/assignable-identity in `resolveField` (+ an analyze alias table), and out-of-package via the
    L5 type-feedback path. See DECISIONS §B4 reassessment (2026-06-21).

- [ ] **B5 — value-position untyped `x := match` (lowering completion).** With the result type
  now inferable via `go/types`, complete the deferred value-position `match` / stored
  Result/Option sum-encoding lowering (§8.7). NOTE: this is a **lowering** gap in
  `internal/pass`/`pipeline`, fed by B1's type info — cross-cutting with the front-end, not a
  pure checker unit. *Depends on B1.*

- [x] **B6 — promote residual 02/06/08 deferrals.** Where `go/types` resolves a case the
  lexical checker (even package-merged) still defers/misfires, upgrade it to a type-backed
  Error; otherwise re-record as a genuine narrower residue. *Resequenced ahead of B4 (user-authorized);
  the "depends on B1–B4" was conservative sequencing — B6's 08 work is independent of B4's feature 12.*
  - **Done (2026-06-20):** `typecheck.CheckNoZeroValue` (`nozero.go`) — promotes the **feature-08**
    residual: **elided composite literals** (type omitted, inferred from a surrounding array/slice/map:
    `[]Inner{{a: 1}}`, `map[string]Inner{"k": {a: 1}}`, `[N]Inner{{…}}`). These are valid Go that
    silently zero-fills omitted fields; the lexical scan can't type the bare `{…}` and **misfires** on
    the surrounding `Inner{` (reports the wrong field set), while `go/types` resolves the inferred type
    and reports the field-accurate Error (`elided-missing-field`, goal-located). Scoped to in-package
    named structs (`pkg == p.Types` ∧ in `Tables.Structs`) so the guarantee stays off imported Go
    structs / injected sum types. 8 tests. No harness/CLI change.
  - **Probe correction:** the reassessment's `Outer{inner: {a: 1}}` example is **invalid Go**
    (struct-field-value elision isn't allowed — only array/slice/map elements/keys), so it is *not* a
    type-backed case; it surfaces as a collected Go error and is deferred. The valid elision positions
    above are the real win. See DECISIONS §B6.
  - **Follow-up done (2026-06-21):** generic-instantiated literals (`Box[int]{val: 1}` omitting `tag`)
    now promoted too — also lexically missed (and not in the analyze tables), resolved via go/types
    (`generic-missing-field`). The `Tables.Structs` guard was replaced with a declaration-position guard
    (`isGoalDeclared`: in-package + `.goal` decl position) so generics are admitted and injected prelude
    structs (Ok/Err) stay excluded. See DECISIONS "B6 follow-up."
  - **Deferred (narrower residue):** qualified out-of-package literals (`pkg.T{…}`) — not goal's
    guarantee; cross-*package* 02/06 (unexported sealed markers not enumerable across a boundary;
    imported Go structs carry no goal contract). Recorded, not faked. See DECISIONS §B6.

**Done when:** each type-dependent deferral in `ROADMAP_TO_GOAL.md` §0 is either a type-backed
Error with a goal-located message, or a re-recorded narrower residue with reason; `goal check`
runs both stages. **`goal check` now runs both stages** (integration unit, 2026-06-21 — see
DECISIONS "Integration — wire the depth stage into `goal check`"); the remaining open items are
B4/B5 (front-end-gated) and the recorded narrow residue.

---

## Open decisions

- **RESOLVED (integration, 2026-06-21) — stage integration & dedup.** `goal check` runs the lexical
  stage then the typed depth stage and merges them; when both flag the same construct (file basename +
  line + feature), the **type-backed finding wins** (the lexical one is dropped). Raw `go/types` errors
  (`Package.Errors`) are **not** surfaced by `check` yet — `importer.Default()` can false-positive on
  third-party imports; `goal build` remains the gate for real Go type errors. Cost: the typed stage runs
  on `check` only, not `build`/`run`. See DECISIONS for the full rationale (incl. the importer caveat).
- **Stage home** (B1, settled): `internal/typecheck` separate from `internal/check`;
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

_Status: thesis drafted 2026-06-20; SPIKE-B1 PASSED 2026-06-20. **B1–B3 + B6 done** (harness, 07
implements, 03 must-use, 08 elided-literal promotion — B6 resequenced ahead of B4 with user
authorization). **Depth stage WIRED into `goal check` (2026-06-21):** both stages now run, dedup
prefers the type-backed finding — see DECISIONS "Integration." The front-end workstream
(LOWERING-TODO) then lowered the recursion (L1/L2) and bounded value-position `match` (L3), which
**CLOSED B4** — not as a depth check (vacuous: a lowered conversion is type-sound by construction) but
by recognizing its substance was front-end lowering; see DECISIONS "B4 — CLOSED by reassessment
(2026-06-21)." **B5** is partly done (L3 value-position inference); its stored `Result`/`Option`
sum-encoding half (§8.7, LOWERING L4) and out-of-package (L5) remain front-end work. The depth-checker
track has delivered every deferred class that survives transpilation and is decidable from the lowered
Go (07 identity, 03 stored/discarded must-use, 08 elided/generic literals) **and surfaces them through
the CLI**; the rest is front-end lowering (LOWERING-TODO L4/L5) or recorded narrow residue
(cross-*package* 02/06). See DECISIONS §B6, "Integration," "B4 … reassessment," and "Phase B queue
reassessment."_
