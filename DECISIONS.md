# Decision Ledger

The single running tally of every choice made building **goal** — decisions (with refused
alternatives and why), assumptions made without explicit discussion, and outright refusals.

This is **history, not current state**: superseded entries stay, with a new entry pointing back.
The feature-audit loop appends to this file every iteration (see `FEATURE-AUDIT-PROMPT.md` Step 4).

Entry kinds:
- **decision** — real alternatives existed; records chosen + refused + why for both.
- **assumption** — decided without explicit discussion or spec mandate; flagged so it can be vetoed.
- **refusal** — considered and rejected with no replacement.

---

## 00-setup — project workspace & process

### Project name "goal" (formerly "Go+" / "go+")
- **Kind:** decision
- **Chose:** "goal" as the working name.
- **Over:** keeping "Go+"/"go+".
- **Why:** user direction. The design spec still says "Go+" internally; treated as the same
  language. (Spec file left unedited — see assumption below.)

### Reference transpiler implementation language: Go
- **Kind:** decision
- **Chose:** Go for the per-feature reference transpilers.
- **Over:** Rust, TypeScript, Zig.
- **Why:** matches the transpile target, so `go/ast` + `go/printer`/`go/format` produce
  gofmt-clean Go directly; fits the user's ecosystem. The alternatives would emit Go as text with
  no native AST tooling. (User chose via AskUserQuestion.)

### Transpiler structure: per-feature standalone
- **Kind:** decision
- **Chose:** each feature gets its own self-contained Go transpiler (own `go.mod`, no shared infra).
- **Over:** one incremental shared parser+AST front-end that each feature extends.
- **Why:** simplest path, matches "a reference transpiler *for that feature*", lets features be
  built/run out of order with no integration coupling. A unified front-end is a deliberate later,
  separate effort. Cost accepted: small helper duplication across feature dirs. (User chose via
  AskUserQuestion.)

### Syntax selection is user-driven, via AskUserQuestion
- **Kind:** decision
- **Chose:** Step 1 of each iteration presents 2–4 distinct candidate syntaxes (with code-preview
  mockups) and the user picks; nothing is written until they choose.
- **Over:** the loop auto-deciding final syntax from the spec's illustrative samples.
- **Why:** user direction — syntax is the cheapest thing to change and the user wants control of
  it. Consequence: the syntax step must run in the interactive main loop, not a detached
  sub-agent (AskUserQuestion can't prompt from background).

### Per-feature commit at end of each turn
- **Kind:** decision
- **Chose:** every iteration commits its own artifacts (one commit per feature) via the
  `/commit-message` skill.
- **Over:** batching multiple features per commit; committing only at the end.
- **Why:** user direction — keeps each feature's audit a reviewable, revertible unit of history.

### Running decision ledger (this file)
- **Kind:** decision
- **Chose:** maintain `DECISIONS.md` as a central running tally; each iteration appends.
- **Over:** keeping decisions only inside each feature's `SYNTAX.md` (local, not aggregated).
- **Why:** user wants one tally of all options, why chosen, why others refused — including
  undiscussed assumptions. Local-only records don't give a project-wide view.

### TODO ordered by the closed sum-type dependency spine
- **Kind:** decision
- **Chose:** enums → match → Result → Option → `?` → closed-`E` → implements → no-zero-value →
  pure → assert → doctests.
- **Over:** following the spec's section order, or grouping by tier.
- **Why:** the sum-type encoding (§2/§8.1) is the spine every error-catching feature reuses, so it
  must be pinned first; `match` needs enums; `Result`/`Option` need both; `?` needs them. Additive
  features (implements, pure, assert, doctests) have no deps and sort after.

---

## Assumptions to revisit (made without explicit discussion)

### `goal-design-spec.md` left unedited despite the rename
- **Kind:** assumption
- **Chose:** treat the spec's "Go+" / "goplus" / `__gop_` naming as referring to "goal"; do not
  rewrite the spec.
- **Over:** find-and-replacing the spec to the new name.
- **Why:** the spec is the design record; mass-renaming risks churn and the user only said "renamed
  *for now*". **Revisit** if the name sticks — may want spec + `__gop_` hygiene prefix updated.

### Output layout `features/<NN-name>/{SYNTAX,TRANSPILE}.md + transpiler/ + examples/`
- **Kind:** assumption
- **Chose:** this directory shape and the `*.goal` / `*.go.expected` example-pair convention.
- **Over:** other layouts (flat files, a single docs tree, `.gop` extension).
- **Why:** a judgment call to keep each feature self-contained and testable; `.goal` extension
  picked to match the new name. **Revisit** if a different extension or layout is preferred before
  the loop generates all 11 features.

---

## 01-enums — closed sum types (real enums)

### Payload field syntax: Rust/Swift named braces `Variant { field: Type }`
- **Kind:** decision
- **Chose:** brace block of `name: Type` fields (`Active { since: Time }`).
- **Over:** Go-style named parens `Active(since Time)` (the spec §2.5 sample); positional tuple
  `Active(Time)` (Rust tuple-variant / Scala / OCaml).
- **Why:** user chose via AskUserQuestion. Lands on the Rust struct-variant / Swift
  associated-value idiom; keeps field names (so construction can use labels), converting "which
  positional field is which" into named access. Positional was refused for losing field names
  (weaker at catching wrong-order errors); Go-parens was refused in favor of the visually distinct
  brace form. Spec sample diverged from but lowers to the same §8.1 encoding.

### Construction syntax: qualified call with labeled args `Status.Active(since: now())`
- **Kind:** decision
- **Chose:** enum-qualified, call syntax, labeled arguments.
- **Over:** qualified composite literal `Status.Active{since: now()}` (Go literal shape);
  unqualified call `Active(since: now())` (Rust/Swift post-import).
- **Why:** user chose via AskUserQuestion. The `Status.` qualifier disambiguates same-named
  variants across enums and ties the variant to its enum; labels catch wrong-argument-order.
  Unqualified was refused (requires globally-unique variant names, loses the visual tie to the
  enum). Composite-literal form was refused in favor of call-shape, though the §8.1 lowering
  *is* a composite literal under the hood.

### Variant separator: newline-separated, no trailing punctuation
- **Kind:** decision
- **Chose:** one variant per line, Go block-decl idiom.
- **Over:** comma-separated (Rust/Swift enum idiom).
- **Why:** user chose via AskUserQuestion. Lowest visual noise; avoids the "forgot/trailing comma"
  error class Go deliberately omits from block declarations.

### Sealed-interface form: `sealed interface NAME {}` + per-variant `implements NAME for T`
- **Kind:** decision
- **Chose:** the `sealed` keyword as the closedness marker for the standalone-type form, with the
  variant set gathered from `implements` declarations (reusing feature 07's mechanism).
- **Over:** union alias `sealed NAME = A | B | C` (TS/Scala 3 union); per-variant suffix
  annotation `type A struct{...} implements NAME`. Also weighed (and refused): **dropping the
  marker entirely** and inferring closedness from "all implementors are local."
- **Why:** user confirmed `sealed` for now. The keyword is *not* redundant with `implements`:
  `implements X for Y` only asserts Y satisfies X and says nothing about closedness; the same
  `implements` is used for **open** contracts (feature 07's `implements io.Writer for JSONWriter`).
  The `sealed` marker is the one bit that distinguishes an open contract (don't exhaustiveness-
  check, allow outside implementors) from a closed enum (check, forbid them). Inference-from-
  absence was refused as non-local and silently unsound (§2.2): adding a satisfier elsewhere breaks
  exhaustiveness with no declared intent. Union-alias and suffix-annotation were refused to keep
  one shared `implements` mechanism (§2.6) rather than introducing a second closedness spelling.
  **Revisit** the keyword choice (`sealed` vs reusing `enum interface` vs `closed`) if preferred.

### Same Go encoding for both forms (§8.1)
- **Kind:** decision
- **Chose:** single-block `enum` and `sealed interface` both lower to sealed interface + one struct
  per variant + unexported `isNAME()` marker.
- **Over:** giving the standalone-type form a distinct lowering.
- **Why:** spec §8.1/§8.0 mandate one encoding ("the universal fallback"). Single-block synthesizes
  `NAME_Variant` structs; the sealed form attaches the marker to the user's own standalone types.
  No §8.7 immediate-vs-stored fork applies here — enums *are* the encoding that fork falls back to.

### Type expressions and argument exprs passed through verbatim (no stdlib resolution)
- **Kind:** assumption
- **Chose:** the reference transpiler copies variant field types and construction argument
  expressions as raw source text; it does not resolve names (e.g. it emits `Time`, not
  `time.Time`).
- **Over:** matching §8.1's sample exactly, which rewrote the bare `Time` to `time.Time`.
- **Why:** name/stdlib resolution is a checker/resolver concern, out of scope for a no-checking
  reference transpiler. The `status` example therefore declares `type Time = int64` locally so the
  output is self-contained and compiles. **Revisit** when a shared front-end with real type
  resolution exists.

### Sealed-form variant types use plain Go struct syntax, constructed as plain Go literals
- **Kind:** assumption
- **Chose:** in the `sealed interface` form, standalone variant types are written as ordinary Go
  `type T struct { Field Type }` and constructed as `T{...}` — the brace-named / labeled-call enum
  sugar applies only to the single-block `enum` form.
- **Over:** inventing a goal-specific struct-declaration syntax for standalone variants.
- **Why:** general struct declaration is not this feature's to design (it is base-Go / touches
  feature 08). The sealed form is the "I need real standalone types" escape hatch, so it speaks Go
  for the type and its construction. Documented as a deliberate asymmetry in SYNTAX.md §1.3.
  **Revisit** if a unified construction surface is wanted.

### Reference transpiler: stdlib-only lexer (`text/scanner`), span-splice + `go/format`
- **Kind:** assumption
- **Chose:** lex with `text/scanner`, recognize only enum/sealed/implements/construction, splice
  generated Go over those byte spans, pass everything else through, then `go/format` the whole.
  No CLI framework, no third-party deps.
- **Over:** a full Go-grammar parser via `go/parser` (rejects goal's non-Go tokens); a hand-written
  recursive-descent parser for all of Go; using `urfave/cli` for the CLI.
- **Why:** the prompt licenses a focused recognizer that passes the rest through; the per-feature
  standalone decision forbids shared infra and favors zero deps. `go/parser` can't parse `enum`/
  `since:`-labels. A scoped recognizer + `go/format` yields gofmt-clean output with minimal code.

### No defensive `panic` emitted by this feature
- **Kind:** assumption
- **Chose:** enum declaration/construction emit no `panic("unreachable: ...")`.
- **Over:** pre-emptively adding defensive panics.
- **Why:** the erasure-with-defensive-panic rule (§8.0) applies only where the checker *proves a
  point unreachable*; declaration/construction prove no unreachability. The first such point is
  `match`'s exhaustive default (feature 02). Recorded so the absence is deliberate.

---

## 02-match — pattern matching with exhaustiveness

### Payload binding: bind-the-value `Status.Active(a) => a.since`
- **Kind:** decision
- **Chose:** bind the narrowed variant to one name, read fields off it (`a.since`).
- **Over:** struct destructure `Active { since }` (Rust struct-variant pattern, consistent with the
  01-enums braced declaration); positional bind `Active(since, at)` (Rust tuple-variant / spec §3.1
  sample).
- **Why:** user chose via AskUserQuestion. Lands on Go's own type-switch idiom (`case T: v.Field`),
  the lowest-familiarity-spend binding form, and lowers cleanly to `__gop_v.Field`. Positional was
  refused for reintroducing the field-order dependence the braced payload removed. Struct-destructure
  was refused despite mirroring the declaration — the user preferred the Go-shaped value binding.

### Variant reference in arms: qualified `Status.Active`
- **Kind:** decision
- **Chose:** enum-qualified variant in patterns.
- **Over:** bare `Active` (spec §3.1 sample; scrutinee type is known so it is unambiguous).
- **Why:** user chose via AskUserQuestion — consistency with the qualified construction form
  `Status.Active(...)` from 01-enums, and the enum stays visible at the match site. Bare was refused
  for surface inconsistency with construction, despite being terser.

### `match` is one unified construct (statement + expression)
- **Kind:** decision
- **Chose:** a single `match` usable in statement position and in value position
  (`x := match`, `return match`, `var x T = match`).
- **Over:** statement-only `match` with a pre-declared `var` assigned inside arms.
- **Why:** user chose via AskUserQuestion. Lands on Rust expression-match; spec §8.2 already defines
  the value-position lowering (`var x T` + per-arm assignment, no IIFE). Statement-only was refused
  for losing value-returning match and forcing a mutable two-step on the user.

### Switch-coexistence (§9): plain `switch` on a closed enum is a compile error
- **Kind:** decision
- **Chose:** reject plain `switch` on a closed-enum value with a located error redirecting to
  `match` (listing the variants); plain `switch` stays legal on non-enum types.
- **Over:** allow-but-unchecked (plain `switch` compiles, no exhaustiveness); allow-only-with-
  explicit-`default`.
- **Why:** user chose via AskUserQuestion; matches spec §3.1 "option 3" and "Refused: plain switch
  on a closed enum." Allow-but-unchecked was refused as exactly the reflex-`switch` failure §3.1
  warns about (model reaches for `switch`, silently loses exhaustiveness). Require-`default` was
  refused because a present `default:` is what defeats exhaustiveness, so it legitimizes the reflex.
  Enforcement is the checker's job; the reference transpiler does not transform plain `switch`.

### `__gop_v` guard variable emitted only when a binding is used
- **Kind:** assumption
- **Chose:** emit `switch __gop_v := s.(type)` only if some arm references its binding; otherwise
  `switch s.(type)` with no guard.
- **Over:** always declaring the guard (spec §8.2 always shows `v :=`).
- **Why:** an always-declared but never-used type-switch guard risks an unused-variable complaint
  and adds noise. Gating on use keeps the output clean and compilable. `__gop_v` follows the §8
  hygiene prefix (spec used bare `v`; renamed for collision-safety).

### Reference transpiler defers the untyped `name := match` value form
- **Kind:** assumption
- **Chose:** the transpiler handles statement, `return match`, and `var name T = match`; it rejects
  `name := match` with a located message pointing to the typed forms.
- **Over:** implementing lightweight type inference to recover the result type for `:=`.
- **Why:** `:=` lowers identically to the typed `var` case but needs the result type, which only the
  checker's inference provides; inferring it here would be ad-hoc and out of scope for a no-checking
  reference transpiler (audit prompt: handle the immediate case, note the fallback). `name := match`
  remains fully valid goal *surface* (documented in SYNTAX.md); only the reference lowering defers.

### 02 transpiler omits enum *construction* (only declaration + match)
- **Kind:** assumption
- **Chose:** the standalone 02 transpiler lowers `enum` declarations (reused §8.1 path) and `match`,
  but not `Status.Active(...)` construction.
- **Over:** copying 01's construction rewrite too.
- **Why:** match examples take the scrutinee as a parameter, so no construction is needed; and a
  qualified arm pattern `Status.Active(a)` is lexically identical to a construction
  `Status.Active(a)`, so including the construction pass would risk mis-rewriting arm patterns.
  Omitting it keeps the two unambiguous. 01-enums already covers construction. Per-feature
  standalone makes this duplication/omission expected.

---

## 03-result — Result[T, error] (open-E keystone)

### Construction/arms qualified: `Result.Ok(v)` / `Result.Err(e)`
- **Kind:** decision
- **Chose:** enum-qualified construction and match arms, `Result.Ok(...)` / `Result.Err(...)`.
- **Over:** bare `Ok(...)` / `Err(...)` (spec §3.2 sample; the universal cross-language spelling and
  what §7 names as conventional).
- **Why:** user chose via AskUserQuestion — one uniform `Type.Variant(...)` construction rule across
  all sum types (matches 01-enums `Status.Active(...)`). Bare was refused despite being the
  convention §7 points at and despite spending a little conventional-name budget; the user preferred
  cross-feature consistency. `Ok`/`Err` names themselves are kept (only the `Result.` qualifier is
  added). **Revisit** if the conventional bare form is later preferred. (Note: this overrides the
  spec sample's bare spelling — recorded under "Open against spec" as a no-semantics change.)

### Type spelling always explicit: `Result[T, error]`
- **Kind:** decision
- **Chose:** both type arguments always written; no shorthand.
- **Over:** a `Result[T]` shorthand defaulting E to `error`.
- **Why:** user chose via AskUserQuestion. Matches the spec §3.2/§8.3 samples, keeps the error
  channel visible in every signature, and adds no defaulting rule the spec didn't define (audit
  guardrail: don't add unrequested features). Shorthand refused for introducing that magic and
  hiding the error type.

### Result return type lowers to NAMED Go returns `(__gop_ok T, __gop_err error)`
- **Kind:** assumption
- **Chose:** rewrite `func ... Result[T, error]` to named returns; `return Result.Err(e)` becomes
  `return __gop_ok, e` (the named zero), `return Result.Ok(v)` becomes `return v, nil`.
- **Over:** the spec §8.3 shape of unnamed `(T, error)` + a synthesized zero literal (`Config{}`);
  also over injecting a `var __gop_zero T` at function top.
- **Why:** a no-type-inference reference transpiler cannot pick the correct zero **literal**
  (`Config{}` vs `0` vs `nil` vs …) from a bare type name, and a per-`Err`-return `var` would
  collide when a function has multiple `Err` returns. Named returns give the zero for any T with no
  literal and no declaration, and remain idiomatic Go. A checker-backed compiler with full type
  info could emit the spec's literal form instead. **Revisit** when real type resolution exists.

### Ok-binding-unused → discard the success value with `_`
- **Kind:** assumption
- **Chose:** at a match site, capture `__gop_v, __gop_err := call` only when the Ok arm uses its
  binding; otherwise `_, __gop_err := call`. The error LHS is always `__gop_err` (the branch
  discriminant).
- **Over:** always binding `__gop_v` (risks an unused-variable compile error when the Ok arm ignores
  the value).
- **Why:** keeps generated Go compiling and clean. Mirrors 02-match's "emit the guard only when
  used" discipline.

### 03 transpiler scope: immediate open-E only; value-position match + stored Results deferred
- **Kind:** assumption
- **Chose:** handle Result whole-return signatures, `return Result.Ok/Err(...)`, and statement-
  position `match` on a Result-returning call. Reject value-position Result `match` (`x := match`)
  with a located message; do not handle stored Results.
- **Over:** implementing the §8.7 stored-value sum-encoding fallback and value-position match now.
- **Why:** the TODO scopes 03 to the open-E immediate keystone; §8.7 stored fallback and closed-E
  are explicitly later (feature 06). The audit prompt says handle the immediate case and note the
  fallback. Deferred forms fail loudly rather than miscompiling.

---

## 04-option — Option[T] / nil-safety (pointer strategy)

### Type spelling: `Option[T]` (not `T?`)
- **Kind:** decision
- **Chose:** Go-generics bracket `Option[T]`, single type arg always explicit.
- **Over:** postfix `T?` optional sugar (Swift/Kotlin/TypeScript/C#).
- **Why:** user chose via AskUserQuestion. Consistent with `Result[T, error]` (one uniform `Sum[...]`
  spelling) and the spec §3.6/§8.4 samples. `T?` was refused to keep `?` reserved exclusively for
  propagation (feature 05, `expr?`) — using the same glyph for an optional *type* and a propagating
  *expression* is conceptually overloaded, and the spec keeps them separate. **Revisit** if `T?` is
  later preferred.

### Construction/arms qualified: `Option.Some(v)` / `Option.None`
- **Kind:** decision
- **Chose:** qualified construction and match arms.
- **Over:** bare `Some(v)` / `None` (spec §3.6 sample; the universal spelling §7 names conventional).
- **Why:** user chose via AskUserQuestion — one uniform `Type.Variant(...)` rule across all sum types
  (matches 01-enums `Status.Active`, 03-result `Result.Ok`). Bare refused for cross-feature
  inconsistency despite being the convention §7 points at. `Some`/`None` names kept; only the
  `Option.` qualifier added. (Overrides the spec sample's bare spelling — see "Open against spec".)

### `Option.Some(v)`: `&v` for a bare identifier, box through a temp otherwise
- **Kind:** assumption
- **Chose:** `return Option.Some(v)` → `return &v` when `v` is a single identifier (addressable,
  matching §8.4's `Some(u) -> &u`); otherwise `__gop_some := v; return &__gop_some`.
- **Over:** always emitting `&v` (illegal Go for literals/calls/index exprs — `&5`, `&f()`); always
  boxing (correct but noisier for the common `Some(u)` case).
- **Why:** Go forbids taking the address of a non-addressable expression, so a literal/call/index
  payload must be boxed through a temp (the idiomatic `v := …; &v`); a bare identifier can be
  addressed directly for cleaner output that matches the spec. Boxing also gives the Option its own
  copy (no aliasing). Edge: a single *constant* identifier isn't addressable and would still emit
  `&c` (rare); noted as a known limitation. The temp name `__gop_some` is shared (distinct `Some`
  returns sit in distinct branches/scopes).

### Option match: deref alias `x := *__gop_o` only when the Some binding is used
- **Kind:** assumption
- **Chose:** in the `Some` (non-nil) branch emit `x := *__gop_o` only if the arm uses `x`; otherwise
  omit it. `Some` → the `if` branch, `None` → `else`, regardless of source order.
- **Over:** always emitting the deref alias (unused-variable error when the Some arm ignores the
  value).
- **Why:** keeps generated Go compiling and clean; mirrors the "emit the binding only when used"
  discipline from 02/03. `__gop_o` is always used (the nil-test), so no guard needed there.

### 04 transpiler scope: immediate pointer strategy only
- **Kind:** assumption
- **Chose:** handle `Option[T]` types, `return Option.Some/None`, and statement-position `match` on
  an Option. Reject value-position Option `match` (`x := match`) with a located message; do not
  specially handle stored Options.
- **Over:** implementing value-position match and an explicit stored/value-type sum encoding now.
- **Why:** the TODO scopes 04 to the immediate case with the same fork as Result; §8.4's
  non-allocating sum encoding for value types and §8.7 stored handling are later. The pointer rep
  `*T` is itself storable, so basic stored Options would still compile, but full handling is
  deferred. Deferred forms fail loudly rather than miscompiling.

---

## 05-question-prop — `?` propagation (open-E)

### `?` is always the RHS of an assignment; discard via `_ := expr?`
- **Kind:** decision
- **Chose:** `?` may appear only as `name := expr?` (keep the value) or `_ := expr?` (discard it,
  propagate only the failure). No bare `expr?` statement.
- **Over:** also allowing a bare `expr?` statement (implicit discard of the success value, the
  Rust/Swift form).
- **Why:** user chose via AskUserQuestion, explicitly for explicitness + consistency. goal already
  uses `_` as the single deliberate-discard marker (match rest-arm §3.1, must-use opt-out §3.2);
  requiring `_ := expr?` makes any discard visible and gives `?` one uniform `lhs := expr?` shape.
  Bare `expr?` was refused as inconsistent with that discipline (silent drop of the unwrapped value).
  Note: the failure (`Err`/`None`) is never silently dropped either way — `?` propagates it; only
  the benign success value is what `_` makes explicit.

### `?` propagation mode comes from the enclosing function's return type
- **Kind:** assumption
- **Chose:** a `?` in a `Result[_, error]` function is Result-mode (`return __gop_ok, __gop_err`); in
  an `Option[_]` function it is Option-mode (`return nil`). The transpiler maps each `?` to its
  enclosing function by source offset.
- **Over:** inferring the operand's type to decide mode (needs type inference the transpiler lacks).
- **Why:** `?` early-returns the *same kind* the enclosing function returns (the failure must have a
  compatible channel), so the return type determines the mode without any operand type inference.
  Matches Rust/Swift. A `?` outside a Result/Option function is a located error.

### Reuse `__gop_err` across Result `?`; fresh `__gop_oN` per Option `?`
- **Kind:** assumption
- **Chose:** Result `name := expr?` emits `name, __gop_err := expr` (reusing the named-return
  `__gop_err`, valid because `name` is new); the discard form uses an if-init to scope `__gop_err`.
  Option `?` uses a monotonic `__gop_o1`, `__gop_o2`, … per occurrence.
- **Over:** unique error temps per Result `?`; reusing one `__gop_o` for Option (which would
  redeclare with `:=`).
- **Why:** Go's `:=` redeclaration rule lets `name, __gop_err :=` reuse `__gop_err` when `name` is
  new (the spec's `cfg, err := ...` pattern), but an Option `__gop_o := ...` with no new LHS var
  would be an error on repeat — so Option temps must be unique. Keeps all generated Go compiling.

### 05 transpiler scope: open-E `?` at statement level; bundles the 03/04 lowerings it needs
- **Kind:** assumption
- **Chose:** the standalone 05 transpiler lowers Result signatures + `return Result.Ok/Err`, Option
  `[T]` types + `return Option.Some/None`, and the `?` operator (statement-level, open-E). It does
  not handle inline `?` (`g(f()?)`), closed-E `?`, or stored Result/Option.
- **Over:** a minimal `?`-only transpiler (couldn't produce compilable output without the
  Result/Option forms) or implementing inline/closed-E `?` now.
- **Why:** `?` composes 03 and 04, so the standalone transpiler must duplicate those lowerings (the
  per-feature-standalone rule expects this). Closed-E `?` + From-conversion is feature 06 (§3.7
  fast-follow); inline `?` and stored values are deferred. Deferred/unsupported `?` forms fail with a
  located message rather than miscompiling.
