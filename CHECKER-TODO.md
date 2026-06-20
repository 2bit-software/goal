# Checker work queue — one guarantee per iteration

The front-end (11/11 features) lowers proven-valid input and **erases** each feature's
static guarantee. This queue is those guarantees, now landing in `internal/check`. The
checker scaffold is **already built**: a stable spine (`internal/check/check.go`) plus
one registered, documented **slot** per guarantee. Each iteration fills in **one slot**
— it does not design the module.

**How to run:** `/loop` with `prompt.md`. Each iteration implements the first unchecked
guarantee below, fills its slot file + adds testdata, verifies, checks the box, and stops.

**The architecture is decided — do not re-litigate (see `internal/check/check.go` doc):**
no new parser; reuse `internal/scan` + `internal/analyze.Tables`; run on the **original**
source before lowering; positions are byte offsets; **defer with a located Warning, never
guess.** Extend `analyze.Tables` when a check needs a fact it lacks; record the extension
in `DECISIONS.md`.

Order is by self-containment / value: the most local, inference-free guarantees first.

---

- [ ] **08-no-zero-value** — field-completeness
  - Slot: `internal/check/fields.go` (`checkFields`). Testdata: `testdata/check/08-no-zero-value/`.
  - Guarantee: every `T{…}` / `Enum.Variant{…}` literal names every field unless it uses
    `...defaults`; an omission without the spread is an **Error**.
  - Spec: §08 in `goal-design-spec.md`; `features/08-no-zero-value/`; §8.0 erasure.
  - Reuse: defaults pass literal-locator; `Tables.Structs`, `Tables.Enums[…].FieldSet`.
  - Deps: none. Defer: literal whose type isn't resolvable at the site → Warning.

- [ ] **02-match** — match exhaustiveness
  - Slot: `internal/check/exhaustive.go` (`checkExhaustive`). Testdata: `testdata/check/02-match/`.
  - Guarantee: a `match` over an enum covers every variant or has an explicit `_`; a gap
    without `_` is an **Error** (the case lowering would otherwise make a silent panic-default).
  - Spec: §02; `features/02-match/`; §8.1 encoding, §8.2 default rule.
  - Reuse: match pass locators (`scan.MatchQualifier`, `scan.MatchBodyBrace`); `Tables.Enums[…].VSet`.
  - Deps: none. Defer: untyped `x := match …` / value-position scrutinee → Warning.

- [ ] **07-implements** — interface satisfaction
  - Slot: `internal/check/implements.go` (`checkImplements`). Testdata: `testdata/check/07-implements/`.
  - Guarantee: `type T struct implements I` — T has every method I declares, signatures
    matching; a missing/mismatched method is an **Error**. (Sealed I = marker, trivially met.)
  - Spec: §07; `features/07-implements/`.
  - Reuse: implements pass clause-locator; `Tables.Sealed`. **Likely needs a method index
    added to `analyze.Tables`** — add it; record in `DECISIONS.md`.
  - Deps: none. Defer: signature equality ambiguous across aliases/embedding → Warning.

- [ ] **06-error-e** — closedness & From-totality
  - Slot: `internal/check/closed.go` (`checkClosed`). Testdata: `testdata/check/06-error-e/`.
  - Guarantee: closed-E `Result[T, E]` stays closed (Err values are E variants) and every
    `?` across error types has a registered `from func`; a missing conversion is an **Error**.
  - Spec: §06; `features/06-error-e/`.
  - Reuse: `Tables.FuncSignatures` (ModeResultClosed, T/E); `Tables.FromRegistry`;
    `Tables.Enums[E].VSet`; closed pass `?`/function pairing.
  - Deps: none (independent of 03). Defer: propagated error type unresolvable at `?` → Warning.

- [ ] **12-derive-convert** — conversion totality
  - Slot: `internal/check/convert.go` (`checkConvert`). Testdata: `testdata/check/12-derive-convert/`.
  - Guarantee: `derive func g(s S) T` is total — every target field is reachable
    field-by-field, via a `from func`, or via an exception clause; an unreachable field is an **Error**.
  - Spec: §12; `features/12-derive-convert/`.
  - Reuse: derive pass field-correspondence + exception clause; `Tables.Structs`, `Tables.FromRegistry`.
  - Deps: **06** (generalizes its From-totality). Defer: map/Option/nested recursion and the
    two bespoke shapes (pmk_upgrade, patterns JSON) → Warning.

- [ ] **03-result** — must-use
  - Slot: `internal/check/mustuse.go` (`checkMustUse`). Testdata: `testdata/check/03-result/`.
  - Guarantee: a Result-returning call's value is consumed (`?`, match, inspected assign,
    or explicit discard); dropping it is an **Error**.
  - Spec: §03; `features/03-result/`.
  - Reuse: `Tables.FuncSignatures` (ModeResult/ModeResultClosed callees); question pass call-site locating.
  - Deps: none. Defer: cover local statement-level drop; defer real flow analysis (stored,
    passed on, then dropped) → Warning. First candidate to graduate onto `go/types` if the
    lexical model is too weak — note that boundary in `DECISIONS.md` if you hit it.

- [ ] **10-assert** — static-provable subset (minimal, reserved)
  - Slot: `internal/check/assert.go` (`checkAssert`). Testdata: `testdata/check/10-assert/`.
  - Guarantee: an `assert` whose condition is a statically-decidable constant proven false
    is an **Error**; a tautology may be a dead-code **Warning**. Everything else stays a
    runtime check — do not over-reach.
  - Spec: §10; `features/10-assert/`. The audit **reserved** this subset; keep it conservative.
  - Reuse: assert pass locator. Deps: none. Defer: any non-constant condition → emit nothing.

---

## Notes for the loop

- The **CLI is already wired**: `goalc` runs the checker before lowering, prints diagnostics
  to stderr, and rejects on any Error (`-nocheck` to skip). An empty slot is a no-op, so the
  build stays green until a guarantee is implemented.
- After 03/12 hit their lexical ceiling, the planned next move is **lowering to `go/ast` +
  `go/types`** for the type-dependent residue — *not* a hand-written Go parser/type-checker.
  That is a separate, later workstream; defer to it with located Warnings until then.
