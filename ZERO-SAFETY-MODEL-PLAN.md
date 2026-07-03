# Plan — safety-only struct literals (feature 08 revision)

> **Status:** design plan, no code yet. Supersedes the "completeness + safety" reading of
> feature 08 with a **safety-only** reading, keeping `...defaults` as accepted-but-redundant
> legacy so no existing code breaks.

## Thesis

Today a struct literal must set **every** field (or write `...defaults`), and `...defaults`
refuses fields whose zero is unsafe. That bundles two ideas: *completeness* ("you named all
fields") and *safety* ("no field is left in a crashing zero state"). The completeness half is
ceremony — a forgotten *safe* zero (`email: ""`) is a logic mistake, not a crash, and Go
programmers already understand zero values. The safety half is the real product.

**New rule (one line):**

> Omitted fields default to their zero. A field whose zero is **unsafe** (nil `map`, raw
> `*T` pointer, `chan`, `func`, method-interface, or an `enum`/sealed sum with no valid
> variant) must be set explicitly.

`...defaults` stays valid but becomes a **no-op signal** — omission already means "default
safely." Legacy code keeps compiling; new code just omits safe fields.

## What is caught vs. dropped

| Omitted field's zero | Today | After |
|---|---|---|
| `string`/`int`/`bool`/float | error `[missing-field]` | ✅ defaults |
| nil slice, `Option[T]` (None) | error `[missing-field]` | ✅ defaults |
| nil `map`, raw `*T`, `chan`, `func`, method-iface | error (`[missing-field]`, or `[unsafe-default]` under `...defaults`) | ❌ **error** — set explicitly |
| `enum` / sealed sum (no zero variant) | error | ❌ **error** — set explicitly |

**Dropped:** the "you forgot a safe field" / struct-evolution catch. Deliberate — by the
"nothing built is unsafe" principle, a wrong-but-safe zero is a logic bug, out of scope.
**Kept:** every genuine footgun (nil map/pointer/chan/func/enum) still forces explicitness.

## Touch-points (grounded)

| File | Change |
|---|---|
| `internal/sema/fields.goal` | The core change. `[missing-field]` (`:98` struct, `:187` variant) currently fires for *any* missing field. Repurpose: for each missing field, run the existing zero-safety classifier (the same one that already gates the autofix at `:109`); if the zero is **safe**, allow it (no error); if **unsafe**, emit `[unsafe-default]` when `...defaults` is at the site (existing path at `:250`), else the new `[unsafe-zero]`. Delete the safe-field completeness error. Apply the same to variant construction (`:187`) — relaxed, `[unsafe-zero]` only. |
| `internal/sema/zerosafety.goal` | No logic change — `ZeroSafety` is the classifier we reuse. (Already promoted/exported.) |
| `internal/typecheck/nozero.goal` | Depth-stage completeness (`litDiag`, `elided-missing-field` `:117`, `generic-missing-field` `:113`) must relax the same way: only flag an omitted field when its resolved type has an unsafe zero. |
| `internal/backend/lower.goal` | `...defaults` expansion (`zeroLit` `:395`) stays for legacy identity. New: a *plain* partial literal `T{a: x}` (no `...defaults`) with safe omissions must lower — it already emits as ordinary Go `T{a: x}` (Go zero-fills), so likely little/no change; verify the emitter doesn't assume completeness. |
| `internal/guide/catalog.goal` | **Remove** the `[missing-field]` entry; **add** an `[unsafe-zero]` entry (message reads for plain omission — no "`...defaults`" phrasing); **leave `[unsafe-default]` unchanged**. Regenerate `AI-KNOWLEDGE-BOOTSTRAP.md` (gated by `TestBootstrapGoldenMatches` + catalog-parity). |
| `goal-design-spec.md` §3.5, `DECISIONS.md` §08, `docs/STATUS.md`, `docs/by-example.md` | Reframe feature 08 from completeness+safety to safety-only. The by-example "Rejecting an incomplete literal" case omits *safe* fields — it must be rewritten to omit an **unsafe** field (else it no longer rejects). |
| `testdata/check/08-no-zero-value/`, `internal/corpus/generate_test.go` | Re-baseline (below). |

## Settled decisions

1. **Error codes (settled).** **Retire `[missing-field]` entirely** — there is no longer a
   completeness error. Two codes carry the safety error, chosen by whether `...defaults` is
   literally written at the site:
   - **`[unsafe-zero]`** (new) — a *plain* omitted field whose zero is unsafe (no `...defaults`).
   - **`[unsafe-default]`** (kept, unchanged message/behavior) — the same situation when
     `...defaults` *is* present. Kept to minimize churn on existing `...defaults` cases.

   So the check is: for each missing field with an **unsafe** zero → emit `[unsafe-default]`
   if `...defaults` is at the site, else `[unsafe-zero]`. A missing field with a **safe** zero
   → allowed, no error, regardless of `...defaults`.
2. **Variant construction (settled): relax** — omitted *safe* payload fields default; an
   omitted *unsafe* payload field is `[unsafe-zero]` (variants never had `...defaults`, so the
   `[unsafe-default]` code never applies here).
3. **`...defaults` expansion (settled): keep expanding** to explicit Go zeros — byte-stable
   legacy, zero corpus churn for existing `...defaults` sites.
4. **Autofix (settled).** The old `[missing-field]` autofix inserted `field: <zero>` for all
   missing fields; the new error names *one unsafe field*, so its suggested fix sets *that*
   field explicitly. Simplify accordingly.

## Migration & test re-baseline

Non-breaking for source (`...defaults` still parses; complete literals still valid), but the
**checker's expectations flip** for cases that omitted safe fields:

- `testdata/check/08-no-zero-value/incomplete_single.goal`, `incomplete_struct.goal`,
  `variant_incomplete.goal` — these `// want [missing-field]` cases will now **pass** if they
  omit safe fields. Convert each to omit an **unsafe** field (to keep testing the real error),
  or move to positive cases. `unsafe_default.goal` stays (still rejects).
- Add new positive cases: partial literal omitting only safe fields → clean; `Option[T]`
  field omitted → clean.
- Bump pinned counts in `internal/corpus/generate_test.go` (`check` currently 66).
- Regenerate goldens (`task update-goldens`) and the manifest (`corpus-gen`) as needed;
  regenerate `AI-KNOWLEDGE-BOOTSTRAP.md`.

## Verification (every phase)

`task check` + `task fixpoint`. Fixpoint is the guardrail: the compiler must still transpile
itself identically — this change alters what the *checker accepts*, not what code *means*.

## Relationship to "make the compiler pass `goal check`"

This plan is the **foundation** for that goal, done cleanly:

1. After the model change, re-run `goal check ./internal/...`. The 223 errors collapse to
   only the **genuinely-unsafe** omissions (bare `Value{}` with raw `*MapValue`/`*StructValue`,
   `emitter{…}` with nil `map` fields). Everything that omitted only safe fields passes for
   free — no helpers, no edits.
2. The residual is the honest one: those structs use **raw nullable `*T`**, which is
   un-idiomatic goal. Follow-on work (separate): change `Value`'s pointer fields to
   `Option[T]` (whose zero is the safe `None`) and initialize the emitter's maps explicitly.
   Then the compiler fully self-checks and `goal check` can gate CI over `internal/`.

## Phasing (loop-ready)

- **P1 — Relax lexical check.** `fields.goal`: safe omission allowed, unsafe omission errors
  regardless of `...defaults`; keep `...defaults` parsing. Local unit/corpus cases. Gate.
- **P2 — Relax depth check.** `typecheck/nozero.goal`: same relaxation for elided/generic
  literals. Gate.
- **P3 — Catalog + spec + docs.** Code taxonomy (decision 1), generalize messages, spec
  §3.5, DECISIONS entry, STATUS, rewrite the by-example "incomplete literal" case to an unsafe
  one, regen bootstrap. Gate.
- **P4 — Re-baseline corpus/testdata.** Convert the `incomplete_*` cases, add safe-omission
  positives, bump pinned counts, regen goldens. Gate.
- **P5 — (follow-on, separate feature) Option-ize the compiler.** `Value` → `Option[T]`
  fields; explicit emitter maps; measure `goal check ./internal/...` → 0; optionally gate CI.

P1–P4 are the language change; P5 is the compiler cleanup this unlocks.
