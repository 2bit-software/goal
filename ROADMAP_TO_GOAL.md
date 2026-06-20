# ROADMAP TO GOAL — making the transpiler language fully realized & usable

This is the **master strategy document** for taking `goal` from "a complete front-end +
static checker" to "a language I can actually build real, multi-file programs in, with
editor feedback." It records the **thesis**, the **decisions that fix the plan's shape**,
the **architectural insight** that orders the work, and the **phased order of operations**.

Each phase below is deliberately a *pointer* to its own loop-ready breakdown file (e.g.
`BUILD-MODEL-TODO.md`), mirroring how `CHECKER-TODO.md` drove the checker. This file is the
"why and in what order"; the per-phase files are the "what, exactly, one unit at a time."

> Scope note: this roadmap covers the **Go-transpile path only**. The `goscript`/interpreter
> path is explicitly out of scope (consistent with the audit, see `TODO.md` cross-cutting notes).

---

## 0. Where we are today (the honest baseline)

| Layer | Status | Evidence |
|-------|--------|----------|
| **Front-end pipeline** (`internal/pipeline`, `internal/pass`) | **Complete** — all 11 features compose, round-trip to independently-compiling Go | `go test ./...` green; `testdata/*.goal`/`.go.expected` |
| **Static checker** (`internal/check`) | **Complete** — all 7 guarantees land located diagnostics | `CHECKER-TODO.md` 7/7 `[x]`; `testdata/check/` |
| **`goalc` CLI** (`cmd/goalc`) | Single-file → Go on stdout; runs checker; `-test`, `-nocheck` | `cmd/goalc/main.go` |
| **Playground** (`cmd/goal-wasm`, `site/`) | Functional WASM, same pipeline, CI-gated by `playground-e2e/` | `site/goal.wasm` |

**What this means:** a developer can today hand-write **one** `.goal` file, run `goalc`, and
get checked, compilable Go. That is a real transpiler — but it is **single-file**, has **no
project model**, **no cross-file resolution**, **no source map back from generated Go**, and
**no editor integration**. Those gaps are what "fully usable" must close.

### The checker's known deferrals (the depth backlog)

The checker is complete *as a lexical checker*. Every guarantee honestly **defers** the
classes it cannot resolve without real type information, surfaced as located `Warning`s (or,
for must-use's hardest case, an explicit refusal). These are documented in `DECISIONS.md` and
`CHECKER-TODO.md`; collected here because **closing them is the headline of this roadmap**:

| Guarantee | Deferred class | What it actually needs |
|-----------|----------------|------------------------|
| 03 must-use | `r := f()` stored-then-dropped / passed-on-then-dropped (never read) | dataflow over resolved types |
| 07 implements | alias-equal-but-differently-spelled signatures (textual-after-normalization could false-mismatch) | real type **identity** |
| 12 derive-convert | map / `Option` / pointer / nested-struct recursion; out-of-package types | real field types |
| 08 no-zero-value | out-of-package / positional / inferred-type literals | resolved symbols |
| 02 / 06 | out-of-file enums; unresolved callee / error-enum / err-value | cross-file symbols + types |
| (front-end lowering) | value-position untyped `x := match`; stored-Result/Option sum-encoding fallback (§8.7) | inferred result type |
| 10 assert | floats / unary / multi-term constant conditions | `go/constant` folding (optional) |

**Read the table top-to-bottom and the pattern is unmistakable: almost every deferral is
blocked on real type information.** That observation is what orders the entire roadmap (§3).

---

## 1. Thesis & guiding principles (do not regress these)

These are inherited from the existing design discipline (`README.md`, `NEXT-SESSION.md`,
`goal-design-spec.md` §8.0, `DECISIONS.md`) and must continue to hold for every phase.

1. **Located, machine-checkable feedback is the product.** Every feature exists to turn a
   *silent runtime failure* or a *human-judgment call* into a *located diagnostic an agent can
   act on*. A false guarantee is worse than an honest deferral — when unsure, emit a `Warning`
   and name what was unresolved.
2. **Stay Go-shaped; lean on the Go toolchain.** Every divergence from Go must earn its keep.
   The corollary that drives this roadmap: when we need a hard analysis answer, **transpile to
   Go and ask the Go toolchain**, rather than rebuild it.
3. **No new parser for `goal`.** The front-end is lexer + name-keyed tables + structural splice,
   not a full AST. This stays. (Using `go/types` on *generated Go* does **not** violate this —
   it parses Go, not goal. See §3.)
4. **Name-keyed facts, never byte offsets, across passes.** Offsets shift under splicing; every
   cross-pass/cross-file fact is keyed by symbol name and rebuilt from source.
5. **Format once, at the very end.** Only the driver calls `go/format`; intermediate source need
   only be *lexable*.
6. **Erase the guarantee, preserve the runtime, never silently fall through.** Proven-unreachable
   points get a defensive `panic`, never silent UB.
7. **Decision discipline:** every judgment call is recorded in `DECISIONS.md` as
   decision / assumption / refusal so it can be vetoed. Every workstream commits in
   reviewable/revertible units via the `/commit-message` skill.

---

## 2. Decisions that fix this plan's shape (settled 2026-06-20)

These were chosen explicitly and set the scope and ordering. Recorded here so future loops
don't re-litigate them.

- **Audience = "a tool I use myself."** Optimize for *me building real goal programs*, not for
  public adoption. ⇒ The build/run model and checker depth are first-class; editor plugins,
  distribution polish, and onboarding are **nice-to-have**, deferred to last.
- **LSP = diagnostics-first, keep the lexer.** When we get to editor support, the LSP serves
  diagnostics + name-table-backed go-to-def + hover + format. We do **not** build a full goal
  AST for completion/rename/semantic-tokens unless the lexer approach hits a concrete ceiling.
- **Depth before breadth.** Close the checker's deferred (type-information-dependent) classes
  **before** building out tooling/LSP. *Why this is not a detour: see §3.*

---

## 3. The architectural insight that orders everything

The two decisions "close depth first" and "a tool I use myself" look like they pull in
different directions (depth vs. usability). They **converge**, because of one fact:

> **Closing the depth backlog requires `go/types`. Running `go/types` requires a complete,
> compilable, multi-file Go *package*. Producing that package — with a shared prelude and a
> source map back to `.goal` — is most of the project/build model that "use it myself" needs.**

So the work that closes depth and the work that makes goal usable on real projects are the
**same foundation**, built once:

```
                       ┌─────────────────────────────────────────┐
                       │  Phase A: analysis substrate             │
                       │  multi-file transpile → one Go package   │
                       │  (shared prelude) + bidirectional         │
                       │  source map (.goal offset ↔ Go position) │
                       └───────────────┬──────────────────────────┘
                                       │ enables
                 ┌─────────────────────┼─────────────────────┐
                 ▼                     ▼                     ▼
        Phase B: go/types        (real multi-file      (LSP diagnostics
        depth checks +           `goal build ./...`    wire format,
        deferred lowerings        for daily use)        Phase C)
```

**This keeps the "no new parser" rule intact.** We never parse goal into an AST. We:
1. lower goal → Go (existing lexer pipeline),
2. load the *generated Go* with `go/types` (Go's own parser/checker, error-tolerant mode),
3. answer the type questions the lexical checker had to defer,
4. map every diagnostic back to `.goal` through Phase A's source map.

That is the thesis (`lean on the Go toolchain`) applied to the checker itself.

---

## 4. Phased order of operations

Sequence: **A → B → C → (D)**. A is the shared foundation; B is the prioritized depth work;
C is the diagnostics-first LSP; D is the deferred nice-to-haves. Each phase gets its own
loop-ready breakdown file (to be authored as we reach it).

### Phase A — Analysis substrate & project model  → `BUILD-MODEL-TODO.md`
**Goal:** `goal build ./...` produces a real, compilable Go package from a multi-file goal
project, plus a source map. This is the keystone; everything depends on it.

Likely units (to be broken down individually):
- **Workspace/package discovery** — walk a module, group `.goal` files into packages.
- **Cross-file `analyze.Tables` merge** — build unified, name-keyed tables across files; resolve
  a symbol declared in file A and used in file B (today single-file only). Define collision and
  visibility rules.
- **Shared prelude extraction** — emit the closed-E `Ok[T,E]/Err[T,E]` runtime **once** per
  package (a generated `goal_prelude.go` or an imported runtime pkg), not per-file (which would
  redeclare). Decide prelude-as-generated-file vs. imported-module.
- **Source map** — bidirectional `.goal` byte-offset ↔ generated-Go position, survivable across
  the splicing passes; the reusable library both `goal build` error-mapping and the LSP consume.
- **`goal` umbrella CLI** — `build` / `check` / `run` / `fmt` / `new`, wrapping `goalc`'s
  single-file core; map `go build`/`go vet` errors on generated Go back to `.goal` via the map.

**Done when:** a multi-file goal project builds & runs end-to-end, and a Go-compiler error in
passed-through code is reported at the correct `.goal` location.

### Phase B — `go/types`-backed depth checks (the priority)  → `DEPTH-TODO.md`
**Goal:** eliminate the deferred classes from §0's table using real type information.

Likely units:
- **`go/types` harness** — load Phase A's generated package error-tolerantly; expose a typed
  view (object/type/selection info) keyed back to goal symbols.
- **03 must-use, stored-then-dropped** — flow analysis (à la `unusedresult`/`unparam`) to catch a
  `Result`/`Option` that is bound and never consumed. This is the explicit refusal we can now lift.
- **07 signature identity** — replace textual-after-normalization with `go/types` type identity
  (kills the alias-spelling false-mismatch).
- **12 conversion recursion** — resolve map/`Option`/pointer/nested-struct field types and check
  totality through them; close out-of-package types.
- **Cross-file 02 / 06 / 08** — re-answer the out-of-file/out-of-package deferrals with resolved
  symbols (depends on Phase A's cross-file tables + types).
- **Deferred *lowerings*** — value-position untyped `x := match` and the §8.7 stored
  Result/Option sum-encoding fallback now have an inferred type available; complete them.

**Done when:** each `Warning`/refusal in the §0 table is either upgraded to a real `Error` with a
type-backed proof, or re-recorded as a genuine, narrower residue with reason.

### Phase C — Diagnostics-first LSP (keep the lexer)  → `LSP-TODO.md`
**Goal:** editor feedback with the analysis we already have; mostly glue on A + B.

Likely units:
- **Diagnostics over LSP** — `didOpen`/`didChange` → run pipeline + lexical check + (cached)
  `go/types` pass → `publishDiagnostics`, using the JSON/range shape from Phase A's source map.
- **Go-to-definition & hover** — served off the name-keyed tables (every symbol is already keyed).
- **Formatting** — delegate to `goal fmt` (see Phase D / interim note).
- *(Not in scope: completion, rename, find-all-refs, semantic tokens — these want a real AST;
  revisit only if the lexer approach hits a concrete ceiling, per §2.)*

### Phase D — Nice-to-haves (deferred; it's a personal tool)  → `TOOLING-TODO.md`
- **`goal fmt`** — a goal-aware formatter (gofmt can't parse goal). Trickiest non-LSP item;
  interim option: format the lowered Go + a line-oriented formatter for goal-only constructs.
- **Editor grammar + client** — one shared TextMate/tree-sitter grammar for highlighting + a thin
  VS Code LSP client (then Neovim/JetBrains as desired).
- **Distribution/DX** — `go install`, `goal new` scaffolding, `//go:generate goal build`, docs.

---

## 5. How we break each phase down (the working loop)

We reuse the discipline that built the checker:

1. **Author the phase's `*-TODO.md`** — a dependency-ordered queue of small, independently
   shippable units, each naming its spec/source pointers and its defer-boundary, exactly like
   `CHECKER-TODO.md`.
2. **One unit per iteration** — implement, add testdata/proof, record decisions in `DECISIONS.md`,
   verify (`go vet ./...`, `go test -count=1 ./...`), check the box, commit one reviewable unit,
   stop. (See `prompt.md` for the checker-loop template; a per-phase prompt will mirror it.)
3. **Honest deferral over false completion** — if a unit hits a real ceiling, record a
   refusal-with-reason and narrow the box; never claim a unit covers cases it doesn't.

---

## 6. Risks & open decisions (resolve at each phase boundary, not now)

- **Prelude delivery** — generated `goal_prelude.go` per package vs. a versioned imported runtime
  module. (Phase A.) Affects how `goal build` lays out output and how upgrades work.
- **Source-map fidelity under splicing** — passes splice bytes; the map must survive every pass.
  Decide map representation (per-pass replacement journal vs. final-diff) early in Phase A.
- **`go/types` error tolerance** — generated Go from a *buggy* goal program may not type-check;
  the harness must collect errors, not stop. (Phase B.)
- **Lexer ceiling for LSP** — if diagnostics-first proves insufficient for daily use, the
  AST-vs-lexer fork reopens (§2). Flag the specific missing capability before reconsidering.
- **`.sentrux` / codebase-memory indexing** — not set up for this repo; revisit if/when the
  tooling surface grows (deferred per prior session note).

---

## 7. Governing files (the map)

- `goal-design-spec.md` — language spec (features 01–11; read-only).
- `DECISIONS.md` — choice/assumption/refusal ledger, §01–§12 (append per unit).
- `CHECKER-TODO.md` — the (complete) checker queue; the template this roadmap's phase files copy.
- `NEXT-SESSION.md` — front-end architecture as built + the original checker plan.
- `README.md` — feature table + pipeline overview.
- `prompt.md` — the per-iteration checker-loop template (per-phase prompts mirror it).
- **This file** — the order of operations across phases; the per-phase `*-TODO.md` files are the
  unit-level breakdowns.

---

_Status (2026-06-20): **Phase A COMPLETE** (`BUILD-MODEL-TODO.md`, U1–U7) — multi-file
discover/transpile/build/run/check works, errors map to `.goal`. **Phase B underway**
(`DEPTH-TODO.md`) — SPIKE-B1 passed (stdlib `go/types` loads the lowered package, answers type
identity + Defs/Uses, positions map to `.goal`); next is B1, the typecheck harness._
