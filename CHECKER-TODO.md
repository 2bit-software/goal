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

- [x] **08-no-zero-value** — field-completeness
  - Slot: `internal/check/fields.go` (`checkFields`). Testdata: `testdata/check/08-no-zero-value/`.
  - Guarantee: every `T{…}` / `Enum.Variant{…}` literal names every field unless it uses
    `...defaults`; an omission without the spread is an **Error**.
  - Spec: §08 in `goal-design-spec.md`; `features/08-no-zero-value/`; §8.0 erasure.
  - Reuse: defaults pass literal-locator; `Tables.Structs`, `Tables.Enums[…].FieldSet`.
  - Deps: none. Defer: literal whose type isn't resolvable at the site → Warning.
  - **Done:** covers in-file struct literals `T{…}` (Error on omission; `...defaults` = complete) and
    paren-form variant constructions `Enum.Variant(…)` (every field required; no `...defaults`).
    Deferred (located Warning, `unresolved-literal-type`): any literal whose type isn't named in-file
    (out-of-package type, unnamed/inferred literal). No `analyze.Tables` extension needed — used
    existing `Structs` + `Enums`. Brace disambiguation (func-body / decl-body / keyword braces)
    handled lexically via `scan.ScanFuncs` + enum/struct decl-span scan. See `DECISIONS.md` §08.
  - **Fix (false-positive):** `...derive(src)` spreads are now complete-by-construction (like
    `...defaults`), and variant payload-binding arms in `match` (`Status.Active(a)`) are recognized as
    bindings (via match arm-pattern spans) and skipped — both previously drew a spurious `missing-field`
    Error. Defer-never-guess preserved (the fix only suppresses would-be Errors on recognized spreads /
    arm patterns). Closes the 02↔08 and 12↔08 shared-harness interactions. See `DECISIONS.md` §08 fix.

- [x] **02-match** — match exhaustiveness
  - Slot: `internal/check/exhaustive.go` (`checkExhaustive`). Testdata: `testdata/check/02-match/`.
  - Guarantee: a `match` over an enum covers every variant or has an explicit `_`; a gap
    without `_` is an **Error** (the case lowering would otherwise make a silent panic-default).
  - Spec: §02; `features/02-match/`; §8.1 encoding, §8.2 default rule.
  - Reuse: match pass locators (`scan.MatchQualifier`, `scan.MatchBodyBrace`); `Tables.Enums[…].VSet`.
  - Deps: none. Defer: untyped `x := match …` / value-position scrutinee → Warning.
  - **Done:** covers all match positions (statement, `return match`, `var x T = match`, and the
    untyped `x := match` the lowering defers) — the enum is resolved from the **arm qualifiers**
    (`Status.Pending`), not the scrutinee, so the value-position deferral did **not** apply here.
    Error `non-exhaustive-match` lists every missing variant qualified, in declaration order; an
    explicit `_` rest-arm = complete. Deferred (located Warning, `unresolved-match-enum`): a match on
    an enum not declared in-file (out-of-package). Non-enum matches (Result/Option, owned by 03/06)
    are skipped silently. No `analyze.Tables` extension — used existing `Enums[…].Variants`/`VSet`.
    Note: payload-binding arms (`Active(a)`) are lexically a variant construction and trip the
    08-fields check under the shared harness, so testdata uses data-less variants. See `DECISIONS.md` §02.

- [x] **07-implements** — interface satisfaction
  - Slot: `internal/check/implements.go` (`checkImplements`). Testdata: `testdata/check/07-implements/`.
  - Guarantee: `type T struct implements I` — T has every method I declares, signatures
    matching; a missing/mismatched method is an **Error**. (Sealed I = marker, trivially met.)
  - Spec: §07; `features/07-implements/`.
  - Reuse: implements pass clause-locator; `Tables.Sealed`. **Likely needs a method index
    added to `analyze.Tables`** — add it; record in `DECISIONS.md`.
  - Deps: none. Defer: signature equality ambiguous across aliases/embedding → Warning.
  - **Done:** covers in-file, non-sealed interfaces — Error `unimplemented-method` (method absent)
    and `method-signature-mismatch` (name present, normalized signature differs), located at the
    `implements` clause; value- and pointer-receiver methods both count; in-file embedded interfaces
    are folded into the obligation. Sealed interfaces (feature 01) are trivially met → skipped.
    Deferred (located Warning, `unresolved-interface`): a qualified (`io.Writer`) interface, an
    interface not declared in this file, or one embedding such — method set unreadable lexically.
    **`analyze.Tables` extended** with a method index: `Interfaces` (iface → methods),
    `EmbeddedIfaces` (iface → embedded names), `Methods` (type → methods); signatures normalized
    (param names + whitespace stripped) for equality. Residual: alias-equal-but-differently-spelled
    signatures could false-mismatch in principle — needs `go/types`; the in-file cases here don't hit
    it and cross-package cases are deferred. See `DECISIONS.md` §07 (checker).

- [x] **06-error-e** — closedness & From-totality
  - Slot: `internal/check/closed.go` (`checkClosed`). Testdata: `testdata/check/06-error-e/`.
  - Guarantee: closed-E `Result[T, E]` stays closed (Err values are E variants) and every
    `?` across error types has a registered `from func`; a missing conversion is an **Error**.
  - Spec: §06; `features/06-error-e/`.
  - Reuse: `Tables.FuncSignatures` (ModeResultClosed, T/E); `Tables.FromRegistry`;
    `Tables.Enums[E].VSet`; closed pass `?`/function pairing.
  - Deps: none (independent of 03). Defer: propagated error type unresolvable at `?` → Warning.
  - **Done:** covers (1) **closedness** — every `Result.Err(E.Variant)`/`E.Variant(payload…)` inside a
    closed-E function must name a variant of *that function's* error enum E: a foreign enum is Error
    `err-outside-closed-enum`, a non-variant name is Error `unknown-error-variant`; and (2)
    **From-totality** — a `?` whose in-file direct-call callee returns a *different* closed E needs a
    registered `from func`, else Error `missing-from-conversion` (same E passes through). Located at the
    `?` token / the `Result` of `Result.Err`. Deferred (located Warning): `?` callee not an in-file
    closed-E Result func → `unresolved-question-error`; closed-E func whose E isn't an in-file enum →
    `unresolved-error-enum`; `Result.Err(X)` whose X isn't a lexical `E.Variant` (bound var/call/larger
    expr) → `unresolved-err-value`. No `analyze.Tables` extension — used `FuncSignatures` +
    `FromRegistry` + `Enums[E].VSet`; per-function spans re-derived locally (closed pass's `funcSpans`
    is private). See `DECISIONS.md` §06 (checker).

- [x] **12-derive-convert** — conversion totality
  - Slot: `internal/check/convert.go` (`checkConvert`). Testdata: `testdata/check/12-derive-convert/`.
  - Guarantee: `derive func g(s S) T` is total — every target field is reachable
    field-by-field, via a `from func`, or via an exception clause; an unreachable field is an **Error**.
  - Spec: §12; `features/12-derive-convert/`.
  - Reuse: derive pass field-correspondence + exception clause; `Tables.Structs`, `Tables.FromRegistry`.
  - Deps: **06** (generalizes its From-totality). Defer: map/Option/nested recursion and the
    two bespoke shapes (pmk_upgrade, patterns JSON) → Warning.
  - **Done:** walks each `derive func` the way `pass.expandDerives` does (same param/return/override
    parse) and asserts every unmentioned target field is resolvable via `resolveField`'s order — same
    type, registered `from func`, or built-in `[]A→[]B` slice recursion. Errors: `unsourced-field` (no
    same-named source field), `unbridged-field` (differing type pair, no registered conversion),
    `fallible-in-total-derive` (only a fallible conversion exists for a total derive). Overrides
    (`Field: expr`) and `_`-skips count as author-supplied. Deferred (located Warning):
    `unresolved-derive-type` (target/source not an in-file struct — out-of-package) and
    `unresolved-derive-field` (map/`Option`/pointer/nested recursion the v1 deriver keeps minimal, incl.
    the bespoke pmk_upgrade/patterns shapes). No `analyze.Tables` extension — used `Structs` +
    `FromRegistry`; the pass's parse helpers are mirrored privately (can't import `internal/pass`).
    Note: `...derive(src)` body literals trip the 08 field-completeness check under the shared harness
    (08 only knows `...defaults`), so clean testdata uses bodyless derives or fully-named literals — the
    08↔`...derive` interaction is recorded in `DECISIONS.md` §12 as a follow-up. See `DECISIONS.md` §12 (checker).

- [x] **03-result** — must-use
  - Slot: `internal/check/mustuse.go` (`checkMustUse`). Testdata: `testdata/check/03-result/`.
  - Guarantee: a Result-returning call's value is consumed (`?`, match, inspected assign,
    or explicit discard); dropping it is an **Error**.
  - Spec: §03; `features/03-result/`.
  - Reuse: `Tables.FuncSignatures` (ModeResult/ModeResultClosed callees); question pass call-site locating.
  - Deps: none. Defer: cover local statement-level drop; defer real flow analysis (stored,
    passed on, then dropped) → Warning. First candidate to graduate onto `go/types` if the
    lexical model is too weak — note that boundary in `DECISIONS.md` if you hit it.
  - **Done:** covers the statement-level drop — a Result-returning direct call (open-E `ModeResult`
    and closed-E `ModeResultClosed`) standing alone as an expression statement is Error
    `dropped-result`, located at the callee. All consuming/nested forms (`name := f(…)`,
    `match f(…)`, `f(…)?`, `return f(…)`, `g(f(…))`) are recognized as uses by the callee's
    immediate lexical neighbours; plain (`ModeNone`) callees carry no obligation. Deferred (located
    Warning): `_ := f(…)` whole-Result discard → `unresolved-result-discard` (the sanctioned
    explicit-discard surface isn't defined yet, SYNTAX.md §5); a statement-leading call with an
    expression continuation (`f(…).x`) → `unresolved-result-use`. **Refused (go/types boundary):** the
    assigned-then-unused class (`r := parse(x)` never read; stored/passed-on-then-dropped) needs real
    dataflow — left deferred, this is 03's graduation point onto `go/ast`+`go/types`. No
    `analyze.Tables` extension — used existing `FuncSignatures`. See `DECISIONS.md` §03 (checker).

- [x] **10-assert** — static-provable subset (minimal, reserved)
  - Slot: `internal/check/assert.go` (`checkAssert`). Testdata: `testdata/check/10-assert/`.
  - Guarantee: an `assert` whose condition is a statically-decidable constant proven false
    is an **Error**; a tautology may be a dead-code **Warning**. Everything else stays a
    runtime check — do not over-reach.
  - Spec: §10; `features/10-assert/`. The audit **reserved** this subset; keep it conservative.
  - Reuse: assert pass locator. Deps: none. Defer: any non-constant condition → emit nothing.
  - **Done:** folds the two purely-lexical constant shapes — a bare boolean literal
    (`assert false` → Error `assert-always-false`; `assert true` → Warning `assert-always-true`,
    dead code) and a two-integer-literal comparison (`LIT OP LIT`, OP ∈ `< <= > >= == !=`; false →
    Error, true → Warning). Condition bounded exactly as the assert pass (keyword at line-start,
    statement to next newline, condition = left of first top-level comma); located at the `assert`
    keyword. Deferred (emit **nothing** — runtime check stands, by design): any non-constant
    condition (identifier/call/field), and — kept conservative — floats, unary `!`/`-`, parens,
    multi-term expressions, and non-decimal/over-large integer literals. No `analyze.Tables`
    extension (`t` unused — folds tokens only). See `DECISIONS.md` §10 (checker).

---

## Notes for the loop

- The **CLI is already wired**: `goalc` runs the checker before lowering, prints diagnostics
  to stderr, and rejects on any Error (`-nocheck` to skip). An empty slot is a no-op, so the
  build stays green until a guarantee is implemented.
- After 03/12 hit their lexical ceiling, the planned next move is **lowering to `go/ast` +
  `go/types`** for the type-dependent residue — *not* a hand-written Go parser/type-checker.
  That is a separate, later workstream; defer to it with located Warnings until then.
