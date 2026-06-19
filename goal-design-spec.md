# Go+ Language Design Specification (Working Draft)

> A correctness-oriented dialect inspired by Go, designed so that AI coding agents
> (and humans) get **fast, located, machine-checkable feedback**. Transpiles to Go
> for standalone/binary use. Shares a semantic core with **goscript**, an embeddable
> scripting language that replaces Lua-style runtimes and offers a smooth
> *script → binary* upgrade path.

---

## 0. How to read this document

Each feature below follows a fixed structure:

- **What it is** — the feature in one or two sentences.
- **Why we chose it** — the reasoning, tied to our priorities.
- **What it gains us** — the concrete error class caught or capability added.
- **Refused / denied** — adjacent ideas we explicitly rejected, and why.
- **Sample** — illustrative syntax (not final; syntax is the cheapest thing to change).
- **Implementation notes** — anything future-us needs to not trip over.

Two recurring principles referenced throughout:

- **The feedback principle.** A language feature's value = how much correctness signal
  it produces, how cheaply, how early, how structured. The empirical ranking of feedback
  types is **tests > compiler errors > human prose**. We bias every decision toward moving
  errors out of "silent runtime failure" or "human judgment" and into "fast, located,
  structured compiler (or test) feedback."
- **The familiarity principle.** The model is strong at Go because the training corpus is
  full of Go. Every divergence from Go-shape spends some of that advantage. So: stay
  Go-shaped by default; when a feature Go *lacks* forces divergence, **land on another
  widely-seen idiom** (Rust/Swift/Scala/OCaml/TS) rather than inventing novel syntax.
  "Look like *something the model has seen a lot*" — Go-shaped is just the cheapest such thing.

**Our stated priority order:** (1) catch the model's errors; (2) help the model reason
about code, but only where it's possible without too much friction.

---

## 1. Project framing & non-negotiables

### 1.1 Two languages, one shared semantic core

- **Go+**: transpiles to Go. Inherits Go's runtime, GC, scheduler, cross-compilation,
  stdlib, and toolchain "for free" by being a source-to-source layer. We spend effort
  only on the front-end correctness features.
- **goscript**: an embeddable scripting language (Lua/Wren/Starlark niche). Runs in an
  engine we own (sandboxing, in-process, no host toolchain at runtime). It is an **exact
  semantic subset** of Go+ — *less* (fewer features) and *capability-restricted*, never
  *different*. This is what makes the **upgrade path** real: a goscript file graduates to
  a Go+ module as (near) a no-op, carrying its correctness properties across the boundary.

**The load-bearing invariant: `goscript ⊆ Go+`, semantically exact.** goscript may be
smaller and sandboxed, but any construct it shares with Go+ must *mean the same thing*.
The instant semantics diverge, "graduate" becomes "port," and the entire thesis dies.
goscript is specified as a **diff against Go+** ("Go+ minus {…}, plus sandbox"), never as
a fresh smaller spec — one source of truth prevents drift.

### 1.2 The "never widen the checker" build rule

The shared type checker / front-end is built **once, against the full Go+ semantics**.
goscript *consumes* it with a restricted feature set; Go+ *consumes* it and adds the
transpile backend. At no point do we "widen" a checker that was built small. Mental test:
**any build step that makes the checker handle *more* than before means the ordering is
wrong.** Every step builds the full checker, consumes it, or restricts it.

### 1.3 Build order (committed)

1. Go+ full semantics spec (thorough enough to build the checker; expect revision as
   implementation surfaces holes — spec and checker co-evolve).
2. goscript spec **as an explicit restriction diff** of Go+.
3. **Shared checker / front-end** built for *full* Go+ semantics. Great diagnostics live here.
4. goscript **execution engine** (executes the restricted set; reuses checker from #3).
5. Go+ **transpile backend** (reuses checker from #3; adds Go codegen).

Build **Go+ first** — it has the cheaper execution path (transpile, inherit Go's backend,
no runtime to own), so we nail semantics and diagnostics against a real, fast, correct
target without simultaneously debugging an interpreter. goscript is then "Go+ minus
features, plus sandbox," a constrained problem against a stable spec.

### 1.4 Diagnostics are the product (cross-cutting requirement)

Every check emits a **located, specific, machine-parseable** error designed to be injected
back into a model's context. This is not per-feature; it's a property of the whole front-end.

- Coarse "compilation failed" is near-useless; "missing variants `Pending`, `Cancelled` at
  line 12" is the asset.
- **Critical for the transpile path:** errors must be reported against **Go+ source**, not
  the generated Go. Essentially all meaningful errors must be caught by *our* front-end
  *before* transpilation. A raw Go-toolchain error about generated code (referencing line
  numbers/constructs that don't exist in the Go+ source) is the worst possible feedback and
  must never leak to the user.

### 1.5 Typing stance (decided)

Types like Go has them: **static, inferred at compile time where possible** (Go's `:=`
familiarity). Not mandatory-annotate-everything. goscript is **also typed** (same stance) —
this is what keeps the upgrade path frictionless (no "add all the types" wall at graduation)
and gives the model the type-feedback ROI. The shared checker means "typed goscript" is
mostly *checker reuse*, not net-new work; only *execution* of typed code is engine-specific,
and types can be largely erased at runtime (as Go does).

---

## 2. The foundation: closed sum types (real enums)

Everything in §3 is an application of this one mechanism. Get this right and exhaustiveness,
`Result`, and `Option` largely build themselves.

### 2.1 What it is

A **closed tagged union** with **per-variant data**. Variants may carry different payloads.
The complete variant set is known to the compiler at check time and is **not externally
extensible** — the declaring module owns the full set, permanently.

### 2.2 Why we chose it

- Go's `iota`-based enums are open (any int is a valid value) and carry no per-variant data.
  That openness is *why Go can't exhaustiveness-check*, and the lack of payload is why Go
  enums can't model "Cancelled, with a reason."
- A closed set is the precondition for exhaustiveness (you can't check "all cases handled"
  unless you know all cases). Closedness across the package boundary is required for
  soundness — if a consumer could add a variant, exhaustiveness in the declaring package
  breaks the moment they do.
- We express closedness via the **same mechanism as explicit `implements`** (§3.4): "a
  contract whose complete set of satisfiers the compiler knows." Two listed features, one
  underlying compiler capability.

### 2.3 What it gains us

Real enums, *plus* it is the spine for exhaustiveness, `Result`, `Option`, and richer errors —
five list items off one mechanism.

### 2.4 Refused / denied

- **`iota` int-const enums as the primary enum.** Open, payload-free — the thing we're fixing.
  (May remain available as plain typed constants for non-enum uses; they are simply not
  "enums" in the exhaustiveness sense.)
- **Externally extensible / open enums** for anything we want exhaustiveness on. Open unions
  defeat exhaustiveness; refused for that role.
- **Novel symbolic syntax** for variants. BPE penalizes rare glyphs and it spends familiarity
  for nothing (see §6).

### 2.5 Sample

```goplus
// Single-block declaration is the default form: the whole set is in one place,
// trivial for the compiler to know completely and easy for the model to read.
enum Status {
    Pending
    Active(since Time)
    Cancelled(reason string, at Time)
}

// Construction
s := Status.Active(since: now())
```

### 2.6 Implementation notes

- **Two closedness forms.** Default: **single-block declaration** (easiest to check, whole
  set co-located). Optional: **sealed-interface form** (variants are independently-existing
  types implementing a sealed interface; compiler gathers all implementors in the module)
  for cases where variants must be standalone types. Prefer the single-block form as the
  common path.
- Closedness is a real semantic commitment: enums are **not** externally extensible. Decide
  and document this at the spec level.
- The "compiler knows the complete satisfier set" capability is shared with `implements`
  (§3.4). Build it as one capability.

---

## 3. Tier 1 — error-catchers (the core of priority #1)

### 3.1 Pattern-matching `match` with exhaustiveness

#### What it is

A **syntactically distinct** match expression (distinct from Go's C-style `switch`) with
`=>` arms, **payload binding in the arm**, exhaustiveness checking, and a deliberate `_`
rest-arm. On a closed enum, every variant must be handled or explicitly dismissed via `_`.

#### Why we chose it

- Exhaustiveness moves "forgot a case" from silent-omission (or distant runtime failure) into
  a **located compile error that enumerates the missing variants** — the feedback principle
  in its purest form.
- **Syntactic distinctness is deliberate.** Go's `switch` has `default:`, and a model trained
  on Go reaches for `default` by reflex — which would silently defeat exhaustiveness (every
  case "handled" by the catch-all). A distinct construct with `_` (the cross-language idiom
  for "rest" in exhaustive matches) means the escape hatch is a *conscious* choice, not Go
  muscle memory. `_` also "looks like how other languages with this feature do it," so it's a
  familiarity *win* despite diverging from Go (familiarity principle: land on another
  well-seen idiom).
- Binding-in-arm is required so exhaustiveness and data extraction compose; otherwise the
  model writes around an awkward two-step and the safety is lost.

#### What it gains us

Unhandled-case errors become compile errors. Plus the destructuring needed to make `Result`/
`Option` pleasant.

#### Refused / denied

- **Reflexive `default:` on enum matches.** Refused — it's exactly how omitted-case errors
  hide. `_` is allowed but is a distinct, deliberate "I'm intentionally not handling the rest."
- **Plain `switch` on a closed enum.** Refused — must be a **compile error that redirects to
  `match`**. If both constructs accept enums, the model reaches for `switch` by reflex and
  gets no exhaustiveness. The unsafe form must be *unreachable* on the types that need safety.

#### Sample

```goplus
match s {
    Pending             => startOnboarding()
    Active(since)       => render(since)
    Cancelled(reason, at) => log(reason, at)
}

// Deliberate partial handling — `_` is a conscious opt-out, not a reflex:
match s {
    Active(since) => render(since)
    _             => showPlaceholder()
}
```

#### Implementation notes

- **Two switch-like constructs coexist (chosen: option 3).** Plain `switch` survives for
  non-enum switching (ints, strings) — Go-shaped and harmless there. `match` is **required**
  the moment you switch on a closed enum; the compiler rejects plain `switch` on enums and
  redirects. Rationale: spend the familiarity budget on divergence *only where it catches
  errors* (enums), nowhere else.
- Exhaustiveness errors **must list the missing variants by name**. A vague "non-exhaustive
  match" throws away most of the value.
- Exhaustiveness checking is **one algorithm** serving enums, `Result`, and `Option` alike.

---

### 3.2 `Result[T, E]` as the error channel

#### What it is

Fallible functions return `Result[T, E]` as the **whole** return value (a sum of `Ok(T)` /
`Err(E)`). The success value lives *inside* the sum, so it cannot be reached without going
through the error path. `Result` is **must-use**.

#### Why we chose it

- Go's `(T, error)` convention makes fallibility a *convention*, not an enforced property:
  `cfg, _ := parse(s)` silently discards the error. This is Go's single biggest silent-failure
  class and the primary thing `Result` exists to kill.
- With `Result`, there is no separate error return to `_`-discard — the error is a *branch of
  the value you hold*, so you must `match` or `?` through it. That is the enforcement.
- Fallibility becomes **visible at the call site by construction** — the return type *is* the
  marker. No separate `throws` clause needed (a reasoning aid, priority #2, gained for free).

#### What it gains us

The ignored-`err` class is eliminated. Fallibility is self-documenting in the type.

#### Refused / denied

- **Mixing `Result` into a tuple** (e.g. `(Result[T,E], Other)`). Refused — the moment
  fallibility is one slot in a tuple, the model can destructure around it (back to Go's
  problem). Fallible functions return `Result` as the *entire* return. (Genuine
  multiple-infallible returns may still use Go-style multi-return — harmless, Go-familiar.)
- **Silent discard of a `Result`.** Refused — ignoring a `Result` is a compile error unless
  explicitly discarded (Rust `#[must_use]` / `let _ =` spirit). Must-use is what makes
  unignorability *complete*; without it the model can compute a `Result` and drop it.
- **Renaming `Ok`/`Err` to a Go-flavored aesthetic.** Refused — the conventional names are
  where the model's cross-language knowledge lives. Familiarity principle.

#### Sample

```goplus
func parse(s string) Result[Config, error] {
    if s == "" {
        return Err(ParseError.Empty)
    }
    return Ok(Config{ ... })
}

// Must handle — can't reach Config without going through the Result:
match parse(input) {
    Ok(cfg)  => run(cfg)
    Err(e)   => report(e)
}
```

#### Implementation notes

- See §3.3 for `E` open-vs-closed (deliberately *not* fixed; supported both ways).
- "Must-use" needs the checker to track `Result`-typed expression results and flag unused
  ones, with an explicit discard form to silence intentionally.

---

### 3.3 Error type `E`: open *and* closed, policy via lint

#### What it is

`E` may be the **open `error` interface** (Go-shaped, flexible) *or* a **closed error enum**
(exhaustively matchable, the failure set visible in the signature). Both are supported by
**one mechanism**; which is the *norm* is a **lint-level policy**, not a language constraint.

#### Why we chose it

- Whether open-default becomes "opt-out-by-neglect" (the same failure mode as reflexive
  `default`) is an **ergonomics question answerable only by use**, not by armchair reasoning.
  So we don't hard-pick; we build both and decide the default *posture* empirically.
- Supporting both lets the **same language** serve both audiences: max-strictness for
  LLM-generated correctness-critical code, lighter friction for scripts/prototyping — selected
  per project by lint level.
- **Lint level is itself a model-feedback channel.** "Closed errors required" turns an open
  error into a *located compiler error telling the model to close it* — purism becomes
  mechanical enforcement, tunable per project, rather than reliance on good behavior.
- This decision is **cheap to reverse** (policy over mechanism), unlike the foundational
  decisions — so "try it and see" is correct empiricism here, not decision-dodging.

#### What it gains us

Per-project strictness as a real feature; an empirical path to the open-vs-closed answer;
the ability to be "purist via linter" without forcing friction on every user.

#### Refused / denied

- **Hard-coding one choice (all-open or all-closed) into the language.** Refused — it would
  impose one population's tradeoff on the other and pre-commit to an unanswered empirical
  question.
- **Two *different* error systems.** Refused — open and closed must be the **same mechanism**
  (`Result`, `match`, `?`, must-use) differing **only** in whether `E` is constrained to a
  closed enum or the open `error` interface. A function moves open→closed by *narrowing `E`*,
  nothing else. If they drift into two mechanisms, the lint-as-policy approach breaks. **This
  one-mechanism-one-knob constraint is the line to protect.**

#### Sample

```goplus
// Open E — ergonomic default, trivial propagation:
func load(p string) Result[Bytes, error] { ... }

// Closed E — failure set is visible and exhaustively matchable:
enum ParseError { Empty; BadKey(key string); BadValue(key string, got string) }

func parse(s string) Result[Config, ParseError] { ... }

match parse(input) {
    Ok(cfg)              => run(cfg)
    Err(Empty)           => useDefaults()
    Err(BadKey(k))       => report("bad key", k)
    Err(BadValue(k, g))  => report("bad value", k, g)   // compiler enforces all cases
}
```

#### Implementation notes

- The **experiment**: ship with open-as-default lint, use it heavily (LLM-generated scripts,
  our own Go+), and watch whether we wish the linter were stricter or reach for closed errors
  unprompted. If closed gets used naturally where it matters and open suffices elsewhere,
  open-default was right; if errors stay vague because open let them, flip the lint default to
  closed and measure the friction. Changing it is a **default flip, not a redesign**.

---

### 3.4 Explicit `implements`

#### What it is

A struct may **declare** that it implements an interface. The declaration does **not** preclude
satisfying other interfaces structurally — it asserts "this type satisfies *at least* this
contract," checked at the declaration site.

#### Why we chose it

- Go's implicit interface satisfaction is the most-cited source of Go confusion for humans
  *and* models: a struct satisfies an interface invisibly, nothing is checked at the
  declaration, and a subtly-wrong signature surfaces only at a distant call site (or never).
- Explicit `implements` converts that into a **located compile error at the struct**: "you
  declared you implement `Writer`; method `Write` is missing/has wrong signature."
- It's **additive, not a nominal-typing conversion** — structural satisfaction still works
  everywhere else. So it's near-zero friction and near-zero divergence: it *adds* a checkable
  assertion without changing what existing Go constructs mean. One of the best value/friction
  ratios on the list.
- Shares its core capability ("compiler knows the satisfier relationship at a declaration
  site") with the closed-enum mechanism (§2). Build once.

#### What it gains us

"I thought this implemented the interface but the signature's wrong" becomes a located,
declaration-site compile error.

#### Refused / denied

- **Converting Go to nominal typing** (requiring `implements` to satisfy an interface).
  Refused — structural satisfaction stays the default; `implements` is an *additive* assertion,
  not a gate.

#### Sample

```goplus
type JSONWriter struct { ... }

// Asserts at least io.Writer; still free to satisfy others structurally elsewhere.
implements io.Writer for JSONWriter

func (w JSONWriter) Write(p []byte) (int, error) { ... }
// If Write were missing or mis-signed, the error points HERE, at the declaration.
```

#### Implementation notes

- Compile-time only; **erased after checking**, zero runtime cost.
- Syntax above (`implements X for T`) is illustrative; could equally be an annotation on the
  type. The semantic requirement is "declaration-site, additive, located error."

---

### 3.5 No-zero-value-surprises (required field construction)

#### What it is

Constructing a struct **requires all fields be set explicitly** (or explicitly defaulted).
Forgetting a field is a compile error, not a silent zero value. **This is a default, not
opt-in.**

#### Why we chose it

- Go's zero values are a notorious silent-error source: a struct with an unset field is
  silently the zero value, not an error. This is a *named, Go-specific footgun* — we're
  attacking Go's actual weakness, not adding generic rigor.
- **Default, not opt-in** (decided): an opt-in strictness the model forgets to opt into
  catches nothing — "default is what makes it actually catch." This is the same
  safe-path-as-default logic that decided `_` over reflexive `default`.

#### What it gains us

"Forgot to set a field" moves from silent-zero-value into a located compile error.

#### Refused / denied

- **Opt-in field-completeness.** Refused for the *default* posture — it would be forgotten.
  (An explicit "use defaults" form is provided so the strictness is satisfiable without
  ceremony where defaults are genuinely wanted.)

#### Sample

```goplus
type User struct { name string; email string; role Role }

// Error: missing field `role`.
u := User{ name: "a", email: "b@c" }

// OK — explicit:
u := User{ name: "a", email: "b@c", role: Role.Member }

// OK — explicitly opting into defaults for the rest:
u := User{ name: "a", email: "b@c", ...defaults }
```

#### Implementation notes

- This is **pervasive, not additive** — it changes how every struct is constructed and is a
  real divergence from Go. We accept it because it's high-value pure error-catching on a named
  Go footgun, and we make it a default deliberately.
- Provide a low-ceremony explicit-defaults form so the common "I really do want zero/defaults"
  case isn't painful (or the model routes around the whole feature).

---

### 3.6 `Option[T]` / nil-safety

#### What it is

`Option[T]` (a sum of `Some(T)` / `None`) that **must be destructured** to access the inner
value. Kills nil-dereference.

#### Why we chose it

- Nil-deref is Go's *other* great silent-failure class (alongside ignored-`err`).
- It's **nearly free** given the sum-type machinery (§2) is already built — `Option` is just a
  library enum, same as `Result`.

#### What it gains us

Nil-dereference becomes impossible-by-construction for `Option` values; "is this nil?" is
forced into a visible match.

#### Refused / denied

- **Pervasive nil for reference types** in the places where `Option` is the right tool. (How
  far to push nil-elimination across *all* pointer types is a larger, more pervasive decision
  deferred to §5; `Option` itself is in Tier 1 because it's cheap and high-value.)

#### Sample

```goplus
func find(id ID) Option[User] { ... }

match find(id) {
    Some(u) => greet(u)
    None    => prompt()
}
```

#### Implementation notes

- Same exhaustiveness algorithm, same must-use considerations as `Result`. Library enum over
  the §2 mechanism.

---

### 3.7 `?` propagation

#### What it is

Postfix `?`: if the value is `Err`/`None`, early-return it from the enclosing function;
otherwise unwrap the `Ok`/`Some`.

#### Why we chose it

- `Result` without propagation sugar is a pyramid of matches — *more* verbose than Go's
  `if err != nil` — and the model will route around it, defeating the safety. `?` is what
  **earns** the divergence from `if err != nil`: it makes the safe path the *short* path.
- `?` is a widely-seen idiom (Rust/Swift). Familiarity principle: when diverging from Go, land
  on a known shape.

#### What it gains us

Error handling that's *shorter* than Go's, so the model uses it instead of avoiding it.

#### Refused / denied

- **`Result` with no propagation sugar.** Refused — it fails both priorities (more verbose
  than Go → routed around → catches nothing).

#### Sample

```goplus
func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?        // if Err, returns it; else unwraps to Bytes
    cfg := parse(raw)?         // same
    return Ok(cfg)
}
```

#### Implementation notes — the conversion problem

- **Open `E`:** `?` is trivial — any error flows up as `error`, no conversion. **Ship this in
  v1.** Covers the common case frictionlessly.
- **Closed `E`:** `?` must convert callee's error enum into the caller's error enum. Use a
  **`From`-style conversion mechanism** (auto-invoked by `?` when a conversion is defined).
  This is the friction cost of typed errors. **Fast-follow, not v1-critical** — consistent with
  "build the cheap high-value thing first."
- The two modes are the same `?` with/without a conversion step — keep them one mechanism
  (§3.3 line-to-protect).

---

## 4. Tier 1.5 / Tier 2 — supporting features

### 4.1 Runnable doctests

#### What it is

Tests co-located with code (in doc comments or inline blocks) that are **extracted and
executed** as part of the standard build. Failures surface as **located errors**.

#### Why we chose it

- Tests are the **top feedback band** (tests > compiler > prose). Doctests put executable
  tests right next to the code, co-locating intent with verification.
- **Conditional on execution.** An executable doctest is top-tier test feedback. A
  doctest-shaped comment that *doesn't* run is unverified natural language — the *lowest*
  feedback band, and actively misleading when it drifts from the code.

#### What it gains us

Cheap behavioral verification in the highest-value feedback band, adjacent to the code it
checks.

#### Refused / denied

- **Doctests that can silently not-run.** Refused — the hard design requirement is *no way*
  for a doctest to silently skip. Non-running doctests are worse than none.

#### Sample

```goplus
/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int { return a + b }
```

#### Implementation notes

- This is **tooling/engine work** (extraction + run wiring), a somewhat separate workstream
  from the type-system features. Build the wiring **once**; both Go+ and goscript use it.
- Reconsider the *form* for goscript depending on who writes those scripts (ephemeral user
  content vs. persistent plugins/mods) — the doctest value may not transfer unchanged from the
  codegen context.

---

### 4.2 Lightweight `pure` annotation

#### What it is

A **declarable-and-checked** "this function has no side effects" marker. *Not* a full granular
effect system.

#### Why we chose it

- It's the reasoning aid (priority #2) that best fits the "where it's low-friction" clause:
  **opt-in** (annotate where useful, silent elsewhere) and the check is **cheap**.
- Enables reasoning ("this can't mutate anything") and dependency analysis / parallelization
  (cf. Pel's AST-dependency auto-parallelization).

#### What it gains us

Cheap, checkable purity facts that aid both the model's reasoning and runtime analysis.

#### Refused / denied

- **Full granular effect systems** (declaring *which* I/O, *which* tables, etc.). Refused for
  now — best theory, **near-zero empirical evidence**, high annotation cost. The single
  most-overrated feature in the design discourse. The *light* version only.

#### Sample

```goplus
pure func square(x int) int { return x * x }

// Compile error if a `pure` function performs I/O or mutation.
```

#### Implementation notes

- Checker verifies the absence of effects within `pure` bodies. Keep the definition of
  "effect" simple and conservative for v1.

---

### 4.3 Asserts

#### What it is

Inline assertions checked **at runtime** (panic/error on failure). Syntax reserved for a
possible future **statically-checkable** subset.

#### Why we chose it

- Runtime asserts are cheap, familiar, and *do* serve priority #1 — they catch errors (at
  runtime, the next-best band after compile time).

#### What it gains us

A low-cost way to encode and check invariants the type system doesn't capture.

#### Refused / denied

- **General static verification of asserts** (Dafny-style proving). Refused for v1 —
  expensive, undecidable in general, and slow feedback violates the "feedback must be fast"
  rule. We ship runtime asserts and **reserve syntax** for a later static-checkable subset
  (e.g. asserts the checker *can* discharge, like simple ranges or enum-exhaustiveness facts).

#### Sample

```goplus
func withdraw(amount int) {
    assert amount > 0
    ...
}
```

#### Implementation notes

- Same fork applies to **pre/postconditions / contracts** (§5): runtime-checked is easy and
  fine; statically-verified is the slow/undecidable trap. Treat contracts, if added, as
  runtime-checked in v1 with syntax reserved — same posture as asserts.

---

### 4.4 Design-in-now, build-later

These cost ~nothing if designed into the core and are expensive to retrofit. **Reserve for
them now even if not built in v1.**

- **Queryable structure for LSP.** Make the checker expose type-at-point, go-to-definition,
  completion. The "typed holes" result is clear: hallucination reduction happens via tight
  integration with the language's type/binding structure ("AIs need IDEs too"). Reasoning aid
  (priority #2), low-friction *only if reserved for now*.
- **Capability shape in the shared core.** goscript needs sandbox/capability control
  regardless (it's the main reason people pick Lua). Design the *shape* of capability-scoped
  effects into the shared core now (Go+ may not surface it), so goscript doesn't force a
  retrofit. Capabilities are how goscript "removes" features like concurrency: not a
  different language, just authority the host doesn't grant.

---

## 5. Pervasive features deferred (decide consciously, not by neglect)

These are **pervasive, not additive** — they change the whole language's feel and spend real
familiarity. Default-vs-opt-in is the key call, and it determines how Go-shaped Go+ stays.

- **Immutability / `const`-by-default.** Shrinks what the model must track (no aliasing/hidden
  mutation), enables dependency analysis. **Decision: keep mutability open by default,
  immutability opt-in** (`const`/`let` where wanted). Rationale: immutability-by-default is the
  single most pervasive divergence from Go-shape and is more reasoning-aid (#2) than
  error-catch (#1) — and #2 is our "only where low-friction" tier. (Contrast: no-zero-value
  §3.5 *is* default, because it's pure #1 error-catching on a named footgun.)
- **Full nil-elimination across all pointer types** (beyond `Option`). Pervasive; deferred.
- **Granular effect systems / per-function error sets (`throws X, Y`).** Annotation burden;
  deferred past v1 unless they fall out of the `Result` design naturally.
- **General contracts as proofs / dependent types / static verification.** Research-grade,
  slow feedback; refused for v1 (runtime-checked variants only, syntax reserved).

---

## 6. Explicitly refused, language-wide

Named so they stay named. The failure mode is a language so feature-dense it leaves the
training distribution and the model gets *worse* at it than at plain Go.

- **Novel symbolic / mathematical-notation syntax** (∀, ∈, ⊕, custom glyphs for keywords).
  Refused. Empirically, BPE tokenizers penalize rare glyphs (APL's symbols cost *more* tokens
  than ASCII; J beats APL precisely by using ASCII). Conceptual density ≠ token density on
  current tokenizers, and exotic Unicode hits byte-fallback (multiple feed-forward passes per
  glyph). It also spends Go-familiarity for nothing. *Token efficiency, where pursued, comes
  from deleting human-only formatting and leaning on type inference — not from a symbol
  alphabet.*
- **Macros / heavy metaprogramming.** Refused — expands what the model must simulate, breaks
  the "one obvious way" property we get from Go. Go omits these deliberately; we keep it that
  way.
- **Removing features enthusiastically.** Each removal is a small distribution shift that makes
  the model *worse* at the language. Subtract surgically (concurrency: yes), not broadly.

---

## 7. The familiarity budget (decision procedure for future features)

A running tally framing: every feature past the *additive* ones spends familiarity. Spend it
where it buys the most error-catching.

- **Additive (cheap — do freely):** explicit `implements`, exhaustiveness/`match`, asserts,
  `pure`, doctests. They *add* a checkable assertion without changing what existing Go
  constructs mean.
- **Replacing (costly — justify each):** `Result` over `if err != nil`, strict zero-values,
  any immutability default, `?`. Each spends real familiarity; do it only when the
  caught-error-class is big enough.

**Decision procedure for any *new* feature idea:**

1. Does it convert a currently-silent error class into a located compile-time (or
   runtime-asserted) error? (If no → probably skip; it's not serving priority #1.)
2. Is it **additive** (→ cheap, do it) or **replacing** (→ only if the caught class justifies
   the familiarity cost)?
3. If it diverges from Go-shape, does it land on **another widely-seen idiom**, or is it novel
   (→ avoid novel)?
4. Is the feedback it produces **fast**? (Slow/undecidable checks violate the feedback-latency
   constraint — defer or make runtime-checked.)

When forced off Go-shape, prefer the prevailing cross-language idiom and **conventional names**
(`Some`/`None`, `Ok`/`Err`, `?`, `=>`, `_`) — do **not** Go-ify them; that spends familiarity
for nothing.

---

## 8. Transpilation to Go (codegen reference)

This section gives the target Go for each feature. It is the contract the transpiler must meet.

### 8.0 Governing principle: checks erased, guarantees compiled in

By transpile time the Go+ checker has **proven** exhaustiveness, must-use, field-completeness,
`implements`, and `pure`. The generated Go therefore does **not** re-check those — they
constrained *which source was legal*, and re-litigating proven facts only bloats the output.
But there is a sharp distinction:

- **Static guarantees** (exhaustiveness, `implements`, field-completeness, must-use, `pure`)
  → **erased entirely.** They produced no runtime behavior; generated Go simply omits them.
- **Runtime semantics** (the *value* a `match` produces, `Result`/`?` control flow, `Option`
  branching, runtime `assert`) → **preserved exactly.** They affect what the program *does*.

**The erasure-with-defensive-panic rule.** Where the checker *proved* control cannot reach a
point (e.g. the impossible default of an exhaustive `match`), we erase the *check* but must
**not** leave undefined behavior. We emit `panic("unreachable: ...")` there — not for
correctness (the proof covers that) but so that if the proof is ever violated (unsafe escape,
checker bug), the failure is **loud**, not a silent wrong-zero-value. This is how "we don't
need to check in the Go" reconciles with "it must behave as expected": we don't *re-check the
proof*, but we don't *trust the universe* to honor it silently either.

Three cross-cutting strategies govern the whole transpiler:

1. **The open-`E` native-tuple strategy is the keystone.** `Result[T, error]` consumed
   immediately (returned + `?`/`match`) lowers to Go's native `(T, error)`, making the output
   *idiomatic* Go with full stdlib interop. Without it, every fallible function returns an
   interface and the output looks like nothing any Go dev wrote. Highest-leverage lowering.
2. **The sum-type encoding is the universal fallback.** Domain enums, closed-`E` `Result`,
   `Option[valuetype]`, and any `Result`/`Option` used as a stored *value* all use the
   sealed-interface-plus-variant-structs encoding. One encoding, reused.
3. **Erasure + defensive panic** for all proven invariants (exhaustiveness now; static
   asserts/contracts later).

Hygiene: all generated temporaries use an unlikely-to-collide prefix (e.g. `__gop_`).

### 8.1 Closed sum types (real enums)

Encoding: **sealed interface + one struct per variant.** The unexported marker method keeps the
set closed *in the generated Go too*, mirroring the Go+ guarantee.

```goplus
enum Status { Pending; Active(since Time); Cancelled(reason string, at Time) }
s := Status.Active(since: now())
```
```go
type Status interface{ isStatus() }

type Status_Pending struct{}
type Status_Active struct{ Since time.Time }
type Status_Cancelled struct{ Reason string; At time.Time }

func (Status_Pending) isStatus()   {}
func (Status_Active) isStatus()    {}
func (Status_Cancelled) isStatus() {}

s := Status(Status_Active{Since: now()})
```

Notes:
- Unexported `isStatus()` is what makes the encoding genuinely closed in Go — no other package
  can add a variant. Keep it even though the checker already proved closedness; it keeps the
  output honest against hand-edits or separate packages.
- Per-variant structs (not one all-fields-nullable struct) — each carries exactly its payload.
- Data-less variants (`Pending`) → empty structs; may intern a shared value later (optimization).
- Both Go+ closedness forms (single-block, sealed-interface) target this **same** Go encoding.

### 8.2 `match` (type switch + erased exhaustiveness)

```goplus
match s {
    Pending               => startOnboarding()
    Active(since)         => render(since)
    Cancelled(reason, at) => log(reason, at)
}
```
```go
switch v := s.(type) {
case Status_Pending:
	startOnboarding()
case Status_Active:
	render(v.Since)        // payload binding -> field access on narrowed type
case Status_Cancelled:
	log(v.Reason, v.At)
default:
	// Erased exhaustiveness check; proven unreachable. Loud failure, not silent fall-through.
	panic("unreachable: non-exhaustive Status (compiler invariant violated)")
}
```

Explicit `_` rest-arm → a **real** default (distinct from the panic default above):
```go
switch v := s.(type) {
case Status_Active:
	render(v.Since)
default:
	showPlaceholder()
}
```

Notes:
- Payload binding maps onto type-switch narrowing: `v` narrows to the variant struct; bound
  names rewrite to field accesses (`since` → `v.Since`).
- **`match` as an expression** is the main impedance mismatch (Go `switch` is a statement).
  When `match` yields a value, lower via `var x T` before the switch + assignment in each arm
  (preferred — clean Go, no closure) rather than an IIFE. Most mechanically involved rewrite.
- The checker knows "proven-exhaustive → panic default" vs "explicit `_` → real default";
  the transpiler must carry that bit through lowering and emit accordingly.

### 8.3 `Result` and `?` — dual strategy

**Open `E` (`Result[T, error]`), consumed immediately → native `(T, error)`.** The keystone.

```goplus
func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?
    cfg := parse(raw)?
    return Ok(cfg)
}
```
```go
func loadConfig(p string) (Config, error) {
	raw, err := readFile(p)
	if err != nil { return Config{}, err }   // `?` lowering
	cfg, err := parse(raw)
	if err != nil { return Config{}, err }
	return cfg, nil                            // Ok(cfg) -> (value, nil)
}
```
This is textbook idiomatic Go: `?` becomes the `if err != nil` block the model was trained on,
and the result interoperates with the entire Go stdlib.

**Closed `E` → sum encoding;** `?` lowers to type-switch-and-return:
```go
// cfg := parse(raw)?   where parse returns Result[Config, ParseError]
__gop_r := parse(raw)
switch __gop_e := __gop_r.(type) {
case Err_ParseError:
	return Result_X_ParseError(Err_ParseError{Value: __gop_e.Value})  // or via From-conversion
case Ok_Config:
	cfg = __gop_e.Value
}
```

Notes:
- **Two lowering strategies keyed on open-vs-closed `E`** — a deliberate transpiler fork. Open
  is the common case and yields clean Go; closed uses the sum encoding.
- `From`-conversion (§3.7 fast-follow) lowers to a conversion call in the `Err` arm. Until
  built, closed-`E` `?` across mismatched error types is a checker error (unsupported in v1).
- See §8.7 for the critical **immediate-vs-stored** distinction that gates the native strategy.

### 8.4 `Option` — pointer strategy where possible

```goplus
func find(id ID) Option[User] { ... }
match find(id) { Some(u) => greet(u); None => prompt() }
```
```go
func find(id ID) *User { ... }   // None -> nil, Some(u) -> &u

if __gop_o := find(id); __gop_o != nil {
	u := *__gop_o
	greet(u)
} else {
	prompt()
}
```

Notes:
- `Option` exists to *eliminate* nil-deref yet transpiles *to* nil — correct and fine: the
  checker proved every access goes through a `match`, so the generated nil-check is provably
  present. Safety lives in the source; the output is its proven-safe shadow.
- Pointer strategy works for reference types. Value types (`Option[int]`) can't use `nil`:
  box to `*int` (simpler codegen — v1) or use the sum encoding (no allocation — later).

### 8.5 Pure-erasure features

- **Explicit `implements`** → erased (Go's structural typing already satisfies it). Optionally
  emit the free, runtime-cost-free assertion `var _ io.Writer = JSONWriter{}` — keeps the
  output self-verifying and documents intent. Recommended.
- **No-zero-value-surprises** → generates *nothing extra*; the feature only ever rejected
  source. Output is the ordinary struct literal with all fields present. `...defaults` lowers
  to explicit per-field default values so the generated literal is also complete.
- **`pure`** → erased to a plain `func`. (Later, the backend *may* exploit purity for
  memoization/reordering/parallelization — an optimization pass, not a codegen requirement.)

```go
// implements io.Writer for JSONWriter  ->
var _ io.Writer = JSONWriter{}     // optional, free, recommended
```

### 8.6 Runtime-preserved features

- **Runtime `assert`** → `if !(cond) { panic("assertion failed: <expr text>") }`. Include the
  source expression text (located-feedback principle for runtime failures). Design the lowering
  to be **toggleable** via a build tag so release builds can strip asserts (not v1-critical).
- **Runnable doctests** → generated **`_test.go` files** running under `go test` (idiomatic,
  free runner, satisfies "no silent non-run" for the transpile path). goscript needs its own
  doctest runner (no `go test` there) — the separate workstream noted in §4.1.

```goplus
/// >>> add(2, 3)
/// 5
func add(a int, b int) int { return a + b }
```
```go
// add_doctest_test.go
func TestDoctest_add_1(t *testing.T) {
	got := add(2, 3)
	want := 5
	if got != want { t.Errorf("doctest add: got %v, want %v", got, want) }
}
```

### 8.7 The immediate-vs-stored `Result`/`Option` distinction (transpiler analysis)

The native-tuple (`Result`) and pointer (`Option`) strategies apply **only when the value is
consumed immediately** — returned and `?`/`match`-ed at the use site. The moment a `Result` or
`Option` is used as a **first-class value** — stored in a slice/map/struct field, passed
around, e.g. `results := []Result[int, error]{...}` — there is no tuple to hold, so it **must**
fall back to the **sum encoding**.

The transpiler therefore needs an analysis: *is this `Result`/`Option` consumed immediately
(optimize to tuple/pointer) or stored as a value (box into the sum)?* Easy to miss until a
stored `Result` produces wrong Go. See §9 open questions.

---

## 9. Open questions / to resolve during implementation

- **Switch coexistence enforcement** (§3.1): finalize the exact rule and error message for
  "plain `switch` used on a closed enum → use `match`."
- **`E` lint default** (§3.3): ship open-default, run the experiment, decide whether to flip.
- **`From`-style conversion** (§3.7): design and schedule as the fast-follow after open-error
  `?`.
- **Explicit-defaults form** (§3.5): exact syntax for "set the rest to defaults" so strict
  construction isn't painful.
- **Doctest form for goscript** (§4.1): depends on who authors scripts; revisit.
- **goscript restriction diff** (§1.1): enumerate exactly what goscript removes and which
  capabilities it gates (concurrency via ungranted capability, etc.).
- **Immediate-vs-stored `Result`/`Option` analysis** (§8.3, §8.4, §8.7): the transpiler must
  decide per use whether a `Result`/`Option` is consumed immediately (→ native tuple / pointer)
  or stored as a first-class value (→ sum encoding). Define this analysis precisely; it's easy
  to miss until a stored `Result` (e.g. `[]Result[int, error]{...}`) generates wrong Go.
- **Spec/checker co-evolution** (§1.3): expect the Go+ semantics spec to revise as the checker
  surfaces ambiguities; freeze the *shape* (feature set, subset relationship) early, let
  details firm up against implementation.

---

## 10. One-paragraph summary

Go+ is a Go-inspired dialect (not a superset) that transpiles to Go, built so AI agents get
located, structured, fast feedback. Its spine is a **closed tagged-union mechanism**, from
which **real enums, exhaustive `match`, `Result`, `Option`, and richer errors** all derive —
five features, two real pieces of engineering (the closed union + the matching/exhaustiveness
checker). Errors are unignorable (`Result`-as-whole-return, must-use) with **open-or-closed
`E` selected by lint policy over one shared mechanism**, made ergonomic by `?`. Additive
correctness assertions (**explicit `implements`**, strict **no-zero-value** construction,
runtime **asserts**, lightweight **`pure`**, runnable **doctests**) round out v1. We refuse
novel symbolic syntax, macros, full effect systems, and static verification — anything that's
slow, evidence-free, or drags the language out of the model's training distribution. A sibling
language, **goscript**, is an exact semantic subset with a sandbox and an engine, giving a
smooth **script → binary upgrade path**. The whole design is governed by two principles:
**maximize cheap located feedback**, and **stay in a shape the model has seen a lot**.
