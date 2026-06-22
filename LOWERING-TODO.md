# LOWERING-TODO — Phase B′: front-end lowering completion

The authorized **front-end workstream** that unblocks the two depth-checker units gated on it
(`DEPTH-TODO.md` B4, B5). This phase **lifts the depth-checker loop's guardrail**: it edits the
lowering passes (`internal/pass`, `internal/pipeline`) deliberately — work the checker loop
forbids. It sits between Phase A (build substrate, done) and the remainder of Phase B (the
type-backed checks that consume what this phase lowers).

Loop-ready in the `DEPTH-TODO.md` style: a **thesis**, the **one architectural decision** that
shapes everything, a **dependency-ordered unit queue**, and the **open decisions**.

---

## Thesis — what this phase completes, and the wall it hits

The lexical lowering passes refuse classes they cannot lower **with the information they have at
lowering time**. Two units are blocked on this:

- **B4 (feature 12, derive-convert recursion):** `internal/pass/derive.go` `resolveField` lowers
  only `sf == tf`, a registered `from func`, and `[]A→[]B`. It refuses `map[K]A→map[K]B`,
  `Option[A]→Option[B]`, `*A→*B`, and nested-struct `A→B` recursion; and `genConversion` reads
  `t.Structs[tgtType]` (in-package only), so it refuses out-of-package target/source structs.
- **B5 (value-position `match`, §8.7):** `internal/pass/match.go` `classifyPosition` defers
  `name := match …` because it needs the **inferred result type**; and the §8.7
  immediate-vs-stored analysis (a stored `Result`/`Option` must fall back to the sum encoding) is
  unimplemented.

**The wall, stated once:** a purely-lexical pass that runs *before* type-checking cannot resolve a
fact that needs *real types* — an imported struct's fields (B4 out-of-package), or the common type
of a `match`'s arms (B5 value position). Everything in this phase is either (a) **tractable
lexically** — in-package recursion, lexically-inferable match types — or (b) **gated on type
info** — out-of-package derive, general value-position match. The queue is ordered (a) before (b),
and (b) is gated on the architectural decision below.

**Invariant (do not regress):** every pass keeps the round-trip discipline — lower to Go that
compiles independently, and keep `go test ./...` green (the `testdata/*.goal`/`.go.expected`
round-trip suite is the contract). A class that still can't be lowered is **deferred with a located
error**, never silently mis-lowered — a wrong lowering is worse than an honest refusal (the same
rule the checker holds).

---

## The one architectural decision (resolve before any (b) unit)

How does a lexical pass get the type info it lacks? Two options; this phase **starts with A** and
escalates to B only if a real program needs it.

- **Option A — bounded lexical inference + honest deferral (recommended start).** Lower the cases
  whose type *is* lexically recoverable (in-package structs via `t.Structs`; a `match` whose arms
  share an obvious lexical type — one enum's variants, or a single primitive-literal kind), and keep
  the located deferral for the rest. Cheap, keeps the pure-lexical pipeline, ships value now.
  *Limit:* leaves out-of-package derive and general value-position match deferred (recorded residue).
- **Option B — type-feedback re-lowering (escalation).** Transpile once, load the result into the
  Phase B `internal/typecheck` go/types harness (already built), read the inferred types, then
  re-lower the deferred constructs with those types. General and correct, but a real architectural
  addition (a second lowering round, position-map implications, cost). Only build this if Option A's
  deferrals turn out to block real multi-package programs.

> Why start with A: it matches the project's "defer, don't over-build" discipline and delivers the
> common cases (single-package programs, in-package conversions) without a new architecture. B is a
> deliberate, separately-justified escalation, not a default.

---

## Unit queue (dependency-ordered; one per iteration)

- [ ] **L1 — derive: in-package container & pointer recursion.** Extend `resolveField` to lower
  `map[K]A→map[K]B` (same key, recurse on `A→B`), `*A→*B`, and `[N]A→[N]B`, reusing the existing
  `[]A→[]B` shape and `elemConv`/registry resolution. Total conversions in v1 (same as slices);
  a fallible element conversion inside a container is deferred with a located error. **Pass-order
  note:** derive runs *after* option (pass 4 → 7), so an `Option[A]` field is already spelled `*A`
  by the time derive generates code — handle the pointer spelling, and verify against the tables
  (which carry the *original* `Option[…]` spelling). *Tractable lexically. Unblocks the in-package
  container slice of B4.*

- [ ] **L2 — derive: nested in-package struct recursion.** A target field of struct type `B` sourced
  from struct type `A`, both in-package, with no `from func`: recurse field-by-field (synthesize the
  inline conversion, or require a registered/derivable path and otherwise defer with a located
  error naming the missing leaf). Reuses `t.Structs` for both structs. *Tractable lexically.
  Completes the in-package portion of B4.*

- [ ] **L3 — value-position `name := match` via bounded lexical type inference.** In
  `classifyPosition`/`lowerMatch`, when the arm bodies share a lexically-inferable result type
  (all arms construct variants of one enum; or all arms are literals of one primitive kind), infer
  `T` and lower to the existing `var name T; switch …` shape. Keep the located deferral for arms
  whose common type is not lexically recoverable. *Option-A scope of B5; the residue stays deferred.*

- [ ] **L4 — stored `Result`/`Option` sum-encoding fallback (§8.7).** Implement the
  immediate-vs-stored analysis: when a `Result`/`Option` is used as a first-class value (element of
  a slice/map literal, a struct field, passed/returned as a value rather than `?`/`match`-ed at the
  site), lower it to the **sum encoding** instead of the native tuple/pointer. This is the spec's §9
  open question — **define the analysis precisely first** (likely: a `Result`/`Option` is "stored"
  unless it is the direct scrutinee of `?`/`match` or the sole return expression). Hard; may want
  the user to confirm the exact rule before building. *Completes B5's hard half.*

- [ ] **L5 — (GATED on Option B) out-of-package derive + general value-position match via type
  feedback.** Only if the architectural decision escalates to B. A re-lowering step that consults
  the go/types harness for imported struct fields (B4 out-of-package) and general match result types
  (B5 general). *Do not start without the explicit Option-B decision.*

**Then (back in the depth loop):** with L1–L4 landed, **B4** and **B5** depth checks become
deliverable on the now-lowered Go — return to `DEPTH-PROMPT.md` for those.

**Done when:** the derive pass lowers in-package map/Option/pointer/nested recursion; value-position
`match` lowers for lexically-inferable types; stored `Result`/`Option` falls back to the sum
encoding; and each remaining class (out-of-package derive, non-inferable value-position match) is
either delivered via Option B or recorded as a narrow, located-deferral residue with reason.

---

## Open decisions (resolve at the named unit, not now)

- **Architecture for type-gated lowering** (before L5): Option A (bounded lexical + deferral) vs
  Option B (type-feedback re-lowering). Lean: A now, B only on demonstrated need.
- **Immediate-vs-stored rule** (L4, spec §9): the exact predicate for "this `Result`/`Option` is
  stored, box it." Confirm with the user before building L4.
- **Fallible container recursion** (L1): v1 defers a fallible element/value conversion inside a
  container with a located error (matches the slice v1 rule); revisit if a real derive needs it.
- **Round-trip testdata:** new lowering cases need `testdata/*.goal` + `.go.expected` pairs, not just
  unit tests — the round-trip suite is the contract. Keep `go test ./...` green every unit.

---

## Pointers
- `DEPTH-TODO.md` B4/B5 — the depth-checker units this phase unblocks; their BLOCKED notes name the
  exact refusals (DECISIONS §B4, "Phase B queue reassessment").
- `internal/pass/derive.go` — `resolveField`/`genConversion` (B4 surface).
- `internal/pass/match.go` — `classifyPosition` (B5 value-position deferral).
- `goal-design-spec.md` §8.7 + §9 — the immediate-vs-stored open question (L4).
- `features/12-derive-convert/`, `features/02-match/` — surface + transpile contracts.
- `internal/pipeline/pipeline.go` — pass order (derive after option; match before enums).

_Status: scoped 2026-06-21 (workstream opened on user authorization to lift the front-end
guardrail). No units started. Recommended first unit: **L1** — unblocked regardless of the
architecture decision, concrete and round-trippable._
