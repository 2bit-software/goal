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
- `DECISIONS.md` — append this feature's decisions, refused options (+why), and any undiscussed
  assumptions to the running ledger (root-level, shared across all features)

---

## Foundation

- [x] **01-enums** — Closed sum types (real enums)
  - Spec: §2, codegen §8.1
  - Deps: none (this is the spine)
  - Nail down: single-block declaration form **and** the optional sealed-interface form; variant
    payload syntax; construction syntax (`Status.Active(since: now())`); data-less variants.
  - Transpile to: sealed interface + one struct per variant + unexported `isStatus()` marker
    (§8.1). Both closedness forms target the **same** encoding.
  - **Done:** `features/01-enums/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Chose
    brace-named payloads `Active { since: Time }`, qualified labeled-call construction
    `Status.Active(since: now())`, newline-separated variants, and `sealed interface` + per-variant
    `implements` for the standalone form (`sealed` confirmed as the closedness marker — not
    redundant with `implements`, which is closedness-agnostic). Both forms lower to the one §8.1
    encoding. Transpiler is stdlib-only (`text/scanner` + span-splice + `go/format`);
    `go test ./...` passes (3/3 examples) and all generated Go compiles.

## Tier 1 — error-catchers

- [x] **02-match** — Pattern-matching `match` with exhaustiveness
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
  - **Done:** `features/02-match/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Chose
    bind-the-value `Status.Active(a) => a.since`, qualified variants, and a unified
    statement/expression `match`. Resolved switch-coexistence: plain `switch` on a closed enum is a
    **compile error** redirecting to `match` (checker-enforced; transpiler passes plain `switch`
    through). Lowering matches §8.2 exactly — type-switch, payload→`__gop_v.Field`, panic-default for
    exhaustive vs real `default` for `_`, `return`/`var x T` value positions (no IIFE). Untyped
    `x := match` is valid surface but its lowering is **deferred** (needs the checker's inferred
    type); transpiler rejects it with a located message. `go test ./...` passes (4/4) and all four
    generated packages compile + `go vet` clean.

- [x] **03-result** — `Result[T, E]` as the error channel (open-`E` common case)
  - Spec: §3.2, codegen §8.3
  - Deps: 01-enums, 02-match
  - Nail down: `Result[T, E]`, `Ok(...)` / `Err(...)` construction, `Result`-as-whole-return rule.
    Scope this item to the **open-`E`** (`Result[T, error]`) common case; closed-`E` is feature 06.
  - Transpile to: the **keystone** — `Result[T, error]` consumed immediately → native Go
    `(T, error)`; `Ok(v)` → `(v, nil)`; `Err(e)` → `(zero, e)` (§8.3). Note the immediate-vs-
    stored fork (§8.7): stored `Result` value → sum encoding fallback; handle immediate for v1.
  - Note: must-use is the checker's job — not implemented here.
  - **Done:** `features/03-result/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Chose
    **qualified** `Result.Ok(...)` / `Result.Err(...)` (one uniform sum-type construction rule with
    01-enums) and always-explicit `Result[T, error]` (no shorthand). Implements the §8.3 keystone:
    return type → native `(T, error)`, `Ok(v)`→`(v, nil)`, `Err(e)`→`(__gop_ok, e)`, and a
    statement-position `match` → the idiomatic `if __gop_err != nil { … } else { … }`. Zero value
    handled via **named returns** (`(__gop_ok T, __gop_err error)`) since the no-checking transpiler
    can't synthesize a type-correct zero literal; Ok-binding-unused discards with `_`. Value-position
    Result match and stored Results (§8.7 sum fallback) are deferred with a located message; must-use
    and the explicit-discard surface are the checker's job (not built). `go test ./...` passes (3/3)
    and all three generated packages compile + `go vet` clean.

- [x] **04-option** — `Option[T]` / nil-safety
  - Spec: §3.6, codegen §8.4
  - Deps: 01-enums, 02-match
  - Nail down: `Option[T]`, `Some(...)` / `None`, the requirement that it be destructured via
    `match` to access the inner value.
  - Transpile to: pointer strategy for reference types (`None`→`nil`, `Some(u)`→`&u`, access via
    proven nil-check, §8.4); value types (`Option[int]`) box to `*int` for v1 (sum encoding is a
    later optimization). Same immediate-vs-stored fork as `Result`.
  - **Done:** `features/04-option/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Chose
    `Option[T]` bracket (not `T?`, keeping `?` for propagation) and qualified
    `Option.Some(...)` / `Option.None` (uniform with enums/Result). Implements the §8.4 pointer
    strategy: `Option[T]`→`*T`, `Option.None`→`nil`, `Option.Some(v)`→`&v` (bare ident) or a boxed
    temp otherwise, and `match`→`if __gop_o := …; __gop_o != nil { x := *__gop_o; … } else { … }`.
    Value types box to `*int` (v1); the Some deref alias is emitted only when the binding is used.
    Value-position Option match and stored Options deferred with a located message; must-use is the
    checker's job. `go test ./...` passes (3/3) and all three generated packages compile + `go vet`
    clean.

- [x] **05-question-prop** — `?` propagation
  - Spec: §3.7, codegen §8.3
  - Deps: 03-result, 04-option
  - Nail down: postfix `?` on `Result` and `Option`; early-return-the-`Err`/`None`, else unwrap.
  - Transpile to: **open-`E` only for v1** — `x := f()?` → `x, err := f(); if err != nil { return
    zero, err }` (the idiomatic `if err != nil` the model knows). `Option` `?` → nil-check early
    return. Closed-`E` `?` needs `From`-conversion → defer to feature 06 / mark unsupported here.
  - **Done:** `features/05-question-prop/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Chose
    **`?` always on the RHS of an assignment** — `name := expr?` keeps the value, `_ := expr?`
    discards it (propagate failure only); no bare `expr?` (explicit/consistent with the `_`-discard
    marker). Mode (Result vs Option) is the enclosing function's return type. Lowering matches §8.3:
    Result `?` → `name, __gop_err := expr; if __gop_err != nil { return __gop_ok, __gop_err }`
    (named returns from 03); Option `?` → `__gop_oN := expr; if __gop_oN == nil { return nil };
    name := *__gop_oN`; discard → if-init form. Composes the 03/04 signature+construction lowerings.
    Closed-E `?`, inline `?`, and stored values deferred with a located message. `go test ./...`
    passes (3/3) and all three generated packages compile + `go vet` clean.

- [x] **06-error-e** — Error type `E`: open *and* closed, one mechanism
  - Spec: §3.3, codegen §8.3 (closed-`E` fork)
  - Deps: 03-result, 05-question-prop
  - Nail down: closed error enum as `E` (e.g. `enum ParseError { ... }`); the **one-mechanism-
    one-knob** constraint (open↔closed differ only in whether `E` is constrained). Resolve §9
    **`From`-style conversion** shape for `?` across mismatched closed error enums.
  - Transpile to: closed-`E` `Result` → **sum encoding** (not native tuple); `?` over closed `E`
    → type-switch-and-return with a `From`-conversion call in the `Err` arm (§8.3).
  - Note: lint-level open-vs-closed *policy* is not a transpile concern; only the two lowerings are.
  - **Done:** `features/06-error-e/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Closed `E`
    is just an `enum` used as the `Result` error type — **no new construction/match/`?` syntax** (the
    one-knob constraint). Resolved §9: the `From`-conversion is a **`from func`** modifier (same shape
    as `pure func`; `?` auto-invokes it by `(Src)→Dst` signature; `from` erases). Lowering is the
    §8.1 **sum encoding** (injected generic `Result[T,E any]` + `Ok`/`Err`): construction →
    `Ok[T,E]{Value: …}`/`Err[T,E]{…}`, `match` → type-switch with defensive panic default, `?` →
    type-switch-and-return with the `from func` call in the `Err` arm when caller/callee error types
    differ. T,E resolved from signatures (scrutinee must be a direct call). Flat Ok/Err match (nested
    `Err`-variant patterns deferred to composing `match e {…}`). `go test ./...` passes (3/3) and all
    three generated packages compile + `go vet` clean.

- [x] **07-implements** — Explicit `implements`
  - Spec: §3.4, codegen §8.5
  - Deps: none (additive)
  - Nail down: `implements io.Writer for JSONWriter` (or annotation form) — additive assertion,
    **not** nominal typing; structural satisfaction stays the default.
  - Transpile to: **erased** (Go's structural typing satisfies it). Optionally emit the free
    `var _ io.Writer = JSONWriter{}` assertion (recommended, §8.5). The reference transpiler emits
    this assertion; it does **not** verify the methods exist (checker's job).
  - **Done:** `features/07-implements/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`. Surface
    `implements X for T` inherited from feature 01 (no new question — re-litigating a settled
    choice). Lowering per §8.5: erase the assertion, emit the free `var _ X = T{}` (or
    `var _ X = (*T)(nil)` when `T` has a pointer-receiver method, detected by scanning receivers, so
    the assertion compiles either way). Same `implements` surface, two lowerings: sealed interface
    (01) → marker method; ordinary interface (here) → erased assertion. `go test ./...` passes (3/3)
    and all three generated packages compile + `go vet` clean (the assertion compiling *is* the
    self-verifying proof).

- [x] **08-no-zero-value** — Required-field struct construction
  - Spec: §3.5, codegen §8.5
  - Deps: 01-enums (for default-valued enum fields in examples)
  - Nail down: all fields required on construction (default, not opt-in); the explicit-defaults
    form. Resolve §9 **explicit-defaults syntax** (`...defaults` placeholder is illustrative).
  - Transpile to: generates **nothing extra** — output is the ordinary struct literal with all
    fields present; `...defaults` lowers to explicit per-field default values (§8.5). The
    reference transpiler passes complete literals through and expands the defaults form; it does
    **not** reject incomplete literals (checker's job).
  - **Done:** `features/08-no-zero-value/{SYNTAX,TRANSPILE}.md` + `transpiler/` + `examples/`.
    Resolved §9: chose **`...defaults`** as the explicit-defaults form (over `_`, bare `default`,
    `..Default`) — names the intent + leans on Go's `...` reading. Field-completeness is the
    **erased** static guarantee, so complete literals pass through **verbatim** (generates nothing
    extra); the only rewrite is expanding `...defaults` to explicit per-field zeros, recovered
    syntactically from each declared type (`nil`/`""`/`false`/`0`, `T{}` for a named struct;
    in-file alias/defined types resolved). "Defaults" = Go zero values written explicitly; **no**
    per-field declared-default syntax invented (spec defines none). Transpiler does **not** reject
    incomplete literals or judge default appropriateness (checker's job). `go test ./...` passes
    (3/3) and all three generated packages compile + `go vet` clean.

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
