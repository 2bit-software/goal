# goal — Feature Audit TODO

Working through each language feature to pin down **(A) syntax**, **(B) Go transpile target**,
and **(C) a runnable per-feature reference transpiler in Go (no error checking yet)**.

**How to run:** `/loop` with `FEATURE-AUDIT-PROMPT.md`. Each iteration does the first unchecked
feature below, writes its three artifacts under `features/<NN-name>/`, checks the box, and stops.

**Syntax is user-driven:** the syntax step (Deliverable A) pauses and asks you to choose between
concrete candidate syntaxes via `AskUserQuestion` (with code previews) before anything is written.
Because that prompt is interactive, run the syntax step in the **main loop** — a detached
sub-agent can't ask you questions. Sub-agents are fine for Deliverables B and C.

**Order is dependency-aware** — the closed sum-type encoding (feature 01) is the spine that
`match`, `Result`, and `Option` all reuse, so it must go first. Do not reorder past a dependency.

Per-feature deliverables (every item):
- `features/<NN-name>/SYNTAX.md` — final grammar + examples + rationale
- `features/<NN-name>/TRANSPILE.md` — input→Go pairs + lowering rules + erasure/preservation
- `features/<NN-name>/transpiler/` — standalone Go transpiler + passing `transpile_test.go`
- `features/<NN-name>/examples/` — `*.goal` / `*.go.expected` pairs

---

## Foundation

- [ ] **01-enums** — Closed sum types (real enums)
  - Spec: §2, codegen §8.1
  - Deps: none (this is the spine)
  - Nail down: single-block declaration form **and** the optional sealed-interface form; variant
    payload syntax; construction syntax (`Status.Active(since: now())`); data-less variants.
  - Transpile to: sealed interface + one struct per variant + unexported `isStatus()` marker
    (§8.1). Both closedness forms target the **same** encoding.

## Tier 1 — error-catchers

- [ ] **02-match** — Pattern-matching `match` with exhaustiveness
  - Spec: §3.1, codegen §8.2
  - Deps: 01-enums
  - Nail down: `match { ... }` with `=>` arms, in-arm payload binding, deliberate `_` rest-arm.
    Resolve §9 **switch-coexistence** rule + error message ("plain `switch` on closed enum → use
    `match`"). Decide statement-vs-expression `match` syntax.
  - Transpile to: Go type-switch; bound names → field accesses on the narrowed type; **proven-
    exhaustive → `panic("unreachable: ...")` default**; explicit `_` → **real** default (§8.2).
    Expression-position `match` → `var x T` before the switch + assignment per arm (no IIFE).
  - Note: exhaustiveness is the checker's job — the reference transpiler **assumes** input is
    exhaustive and just emits the panic-default. No checking.

- [ ] **03-result** — `Result[T, E]` as the error channel (open-`E` common case)
  - Spec: §3.2, codegen §8.3
  - Deps: 01-enums, 02-match
  - Nail down: `Result[T, E]`, `Ok(...)` / `Err(...)` construction, `Result`-as-whole-return rule.
    Scope this item to the **open-`E`** (`Result[T, error]`) common case; closed-`E` is feature 06.
  - Transpile to: the **keystone** — `Result[T, error]` consumed immediately → native Go
    `(T, error)`; `Ok(v)` → `(v, nil)`; `Err(e)` → `(zero, e)` (§8.3). Note the immediate-vs-
    stored fork (§8.7): stored `Result` value → sum encoding fallback; handle immediate for v1.
  - Note: must-use is the checker's job — not implemented here.

- [ ] **04-option** — `Option[T]` / nil-safety
  - Spec: §3.6, codegen §8.4
  - Deps: 01-enums, 02-match
  - Nail down: `Option[T]`, `Some(...)` / `None`, the requirement that it be destructured via
    `match` to access the inner value.
  - Transpile to: pointer strategy for reference types (`None`→`nil`, `Some(u)`→`&u`, access via
    proven nil-check, §8.4); value types (`Option[int]`) box to `*int` for v1 (sum encoding is a
    later optimization). Same immediate-vs-stored fork as `Result`.

- [ ] **05-question-prop** — `?` propagation
  - Spec: §3.7, codegen §8.3
  - Deps: 03-result, 04-option
  - Nail down: postfix `?` on `Result` and `Option`; early-return-the-`Err`/`None`, else unwrap.
  - Transpile to: **open-`E` only for v1** — `x := f()?` → `x, err := f(); if err != nil { return
    zero, err }` (the idiomatic `if err != nil` the model knows). `Option` `?` → nil-check early
    return. Closed-`E` `?` needs `From`-conversion → defer to feature 06 / mark unsupported here.

- [ ] **06-error-e** — Error type `E`: open *and* closed, one mechanism
  - Spec: §3.3, codegen §8.3 (closed-`E` fork)
  - Deps: 03-result, 05-question-prop
  - Nail down: closed error enum as `E` (e.g. `enum ParseError { ... }`); the **one-mechanism-
    one-knob** constraint (open↔closed differ only in whether `E` is constrained). Resolve §9
    **`From`-style conversion** shape for `?` across mismatched closed error enums.
  - Transpile to: closed-`E` `Result` → **sum encoding** (not native tuple); `?` over closed `E`
    → type-switch-and-return with a `From`-conversion call in the `Err` arm (§8.3).
  - Note: lint-level open-vs-closed *policy* is not a transpile concern; only the two lowerings are.

- [ ] **07-implements** — Explicit `implements`
  - Spec: §3.4, codegen §8.5
  - Deps: none (additive)
  - Nail down: `implements io.Writer for JSONWriter` (or annotation form) — additive assertion,
    **not** nominal typing; structural satisfaction stays the default.
  - Transpile to: **erased** (Go's structural typing satisfies it). Optionally emit the free
    `var _ io.Writer = JSONWriter{}` assertion (recommended, §8.5). The reference transpiler emits
    this assertion; it does **not** verify the methods exist (checker's job).

- [ ] **08-no-zero-value** — Required-field struct construction
  - Spec: §3.5, codegen §8.5
  - Deps: 01-enums (for default-valued enum fields in examples)
  - Nail down: all fields required on construction (default, not opt-in); the explicit-defaults
    form. Resolve §9 **explicit-defaults syntax** (`...defaults` placeholder is illustrative).
  - Transpile to: generates **nothing extra** — output is the ordinary struct literal with all
    fields present; `...defaults` lowers to explicit per-field default values (§8.5). The
    reference transpiler passes complete literals through and expands the defaults form; it does
    **not** reject incomplete literals (checker's job).

## Tier 1.5 / Tier 2 — supporting

- [ ] **09-pure** — Lightweight `pure` annotation
  - Spec: §4.2, codegen §8.5
  - Deps: none (additive)
  - Nail down: `pure func ...` marker. *Not* a granular effect system.
  - Transpile to: **erased** to a plain `func` (§8.5). The reference transpiler strips the `pure`
    keyword; it does **not** check for effects (checker's job).

- [ ] **10-assert** — Runtime asserts
  - Spec: §4.3, codegen §8.6
  - Deps: none
  - Nail down: `assert <expr>` statement; reserve (don't build) the static-checkable subset.
  - Transpile to: `if !(cond) { panic("assertion failed: <expr text>") }` including the source
    expression text (§8.6). Design the lowering toggleable via build tag (note it; v1 need not
    fully implement stripping).

- [ ] **11-doctests** — Runnable doctests
  - Spec: §4.1, codegen §8.6
  - Deps: none
  - Nail down: doctest form in doc comments (`/// >>> add(2, 3)` / expected-output line); the hard
    requirement that there is **no way to silently not-run**.
  - Transpile to: generated `_test.go` files running under `go test` (§8.6). The reference
    transpiler extracts doctests from comments and emits `func TestDoctest_...`. (goscript's own
    runner is out of scope — Go transpile path only.)

---

## Cross-cutting notes (apply during relevant features, not separate items)

- **Immediate-vs-stored analysis (§8.7):** for `Result`/`Option`, the native-tuple/pointer
  strategy applies only when the value is consumed immediately. Stored as a first-class value →
  sum encoding. v1 reference transpilers handle the immediate case and note the fallback.
- **Hygiene:** all generated temporaries use the `__gop_` prefix.
- **Erased vs preserved (§8.0):** static guarantees erased; runtime semantics preserved; proven-
  unreachable points get a defensive `panic`, never silent fall-through.
- **goscript:** out of scope for this audit pass — we are pinning the Go+ → Go path only.
