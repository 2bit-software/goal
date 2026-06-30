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
  assert → doctests.
- **Over:** following the spec's section order, or grouping by tier.
- **Why:** the sum-type encoding (§2/§8.1) is the spine every error-catching feature reuses, so it
  must be pinned first; `match` needs enums; `Result`/`Option` need both; `?` needs them. Additive
  features (implements, assert, doctests) have no deps and sort after.

---

## Assumptions to revisit (made without explicit discussion)

### Hygiene prefix renamed `__gop_` → `__goal_`
- **Kind:** decision
- **Chose:** the synthesized-temporary / named-return hygiene prefix is now `__goal_` (e.g.
  `__goal_ok`, `__goal_err`, `__goal_v`), matching the project name. Compiler source, golden
  fixtures, reference transpilers, and docs were updated together.
- **Over:** keeping the legacy `__gop_` prefix (a leftover from the "Go+" / "goplus" name).
- **Why:** the name "goal" has stuck, so the earlier "treat `__gop_` as referring to goal *for now*"
  assumption was resolved by doing the rename. The `goal-design-spec.md` prose still uses "Go+" /
  "goplus" framing as the design record; only the `__gop_` identifier prefix was changed.

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
  the lowest-familiarity-spend binding form, and lowers cleanly to `__goal_v.Field`. Positional was
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

### `__goal_v` guard variable emitted only when a binding is used
- **Kind:** assumption
- **Chose:** emit `switch __goal_v := s.(type)` only if some arm references its binding; otherwise
  `switch s.(type)` with no guard.
- **Over:** always declaring the guard (spec §8.2 always shows `v :=`).
- **Why:** an always-declared but never-used type-switch guard risks an unused-variable complaint
  and adds noise. Gating on use keeps the output clean and compilable. `__goal_v` follows the §8
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
- **Partially lifted (Lowering L3, 2026-06-21):** the match pass now infers the result type for
  `name := match` in the **bounded set of shapes recoverable lexically** and lowers them like the typed
  `var name T = match` case (a new `posInferVar` resolved in `lowerMatch`, then treated as `posVar`).
  Inferable: every arm body (rest included) is a string literal (→ `string`), a bool literal (→ `bool`),
  or a construction of one and the **same** enum (→ that enum's type) — all arms must agree. **Still
  deferred** (located error pointing to the typed forms): numeric literals (too many sub-types to pin
  lexically — `int` vs `float64` vs sized), identifiers/calls/compound expressions, and any
  heterogeneous mix (incl. two different enums). A *wrong* inferred type would be a silent miscompile,
  so the inference is deliberately conservative — the general case awaits Option B (type-feedback
  re-lowering, LOWERING-TODO L5). Proof: round-trip `testdata/match_infer_value.goal` (enum- and
  string-valued); `internal/pass/match_test.go` pins enum/string/bool inference and numeric/mixed/
  two-enum deferral.

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

### Checker (`internal/check/exhaustive.go`) — exhaustiveness resolved from arm qualifiers, not scrutinee
- **Kind:** decision
- **Chose:** the checker fires `non-exhaustive-match` (Error) when a `match`'s arms name an in-file
  enum (read from `analyze.Tables.Enums[…].Variants`) but omit at least one of its variants **and**
  the match has no `_` rest-arm. The enum is resolved from the **arm qualifiers** (`Status.Pending`),
  not from the scrutinee's declared type. Arm location reuses the match pass's machinery verbatim:
  `scan.MatchBodyBrace`/`scan.MatchBrace` bound the arm block, arrows are the depth-0 `=>` tokens, and
  the `patternStart` locator (lifted from `internal/pass/result.go`, shared by every qualified match)
  finds each arm's first token — then this check **reads** the covered set instead of lowering.
- **Over:** resolving the scrutinee's type from the function signature / construction context (what
  the slot doc sketched). The arm qualifiers carry the enum name in *every* match position, so reading
  them is both simpler and strictly more position-independent.
- **Why:** the arm-qualifier read makes the check fire uniformly on **all** match positions —
  statement, `return match`, `var x T = match`, **and the untyped `x := match`** that the lowering
  defers (the lowering needs the *result* type; exhaustiveness needs only the *enum*, which the arms
  name). So the value-position deferral the CHECKER-TODO listed does **not** apply to exhaustiveness.
  No `analyze.Tables` extension was needed — `Enums[…].Variants`/`VSet` already carry the variant set.

### Defer-boundary: out-of-file enum → Warning; non-enum match → skipped silently
- **Kind:** decision
- **Chose:** two non-Error outcomes. (a) When the arms are enum-qualified but the named enum is **not
  declared in this file** (an out-of-package enum), its full variant set is unknown, so completeness is
  unprovable — emit a located **Warning** (`unresolved-match-enum`, "exhaustiveness deferred") naming
  the enum, never an Error. (b) When a match has **no** enum-qualified arm — a `Result`/`Option` match
  (`Result.Ok(v)` etc., owned by features 03/06), or any construct whose first arm qualifier is not a
  known enum — the match is skipped **silently** (no diagnostic at all).
- **Over:** assuming an out-of-file match is exhaustive (a false guarantee), or emitting a Warning on
  every Result/Option match (noise on matches this guarantee does not own).
- **Why:** "defer, never guess" — a false "exhaustive" on an unresolvable enum is worse than an honest
  deferral. Result/Option exhaustiveness belongs to their own features, so silence (not a Warning) is
  correct there; the same first-arm-qualifier key the match pass uses to *claim* enum matches is reused
  here to *recognize* them.

### File-layout / `Code` scheme + the 08-fields cross-check interaction
- **Kind:** assumption
- **Chose:** `Feature` = `"02-match"`; `Code` = `"non-exhaustive-match"` (Error) and
  `"unresolved-match-enum"` (deferral Warning). The message lists every missing variant **qualified**
  (`Status.Cancelled`) in **declaration order**, echoing the arm form the agent must add. Testdata uses
  **data-less** enum variants so a payload-binding arm (`Status.Active(a)`) — which is lexically
  identical to a variant construction and so trips the **08-fields** check (`missing-field`) when the
  harness runs all checks together — does not contaminate exhaustiveness cases. (The 02↔08 lexical
  ambiguity is the same one recorded in "02 transpiler omits enum construction" above.)
- **Over:** no naming scheme was fixed by the spec; payload-binding arms in testdata were avoided
  rather than worked around in another slot.
- **Why:** stable greppable codes per the slot doc. The `patternStart` `(binding)` branch is lifted
  verbatim from the proven match-pass locator (exercised by the front-end round-trip suite), so the
  exhaustiveness testdata can stay data-less and focused on variant *coverage* without re-proving arm
  binding parsing.
- **Resolved (08-fields fix):** the 02↔08 interaction is now fixed in `checkFields` — payload-binding
  arms (`Status.Active(a)`) inside a `match` are recognized as bindings (via `matchPatternSpans`) and
  no longer trip the 08 `missing-field` check. The note above stands as the historical reason the 02
  testdata uses data-less variants; data-carrying payload-binding arms are now safe under the shared
  harness (proven by `testdata/check/08-no-zero-value/match_binding_arm.goal`). See §08 "Fix:
  `...derive(src)` spread and match payload-binding arms no longer false-flag."

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

### Result return type lowers to NAMED Go returns `(__goal_ok T, __goal_err error)`
- **Kind:** assumption
- **Chose:** rewrite `func ... Result[T, error]` to named returns; `return Result.Err(e)` becomes
  `return __goal_ok, e` (the named zero), `return Result.Ok(v)` becomes `return v, nil`.
- **Over:** the spec §8.3 shape of unnamed `(T, error)` + a synthesized zero literal (`Config{}`);
  also over injecting a `var __goal_zero T` at function top.
- **Why:** a no-type-inference reference transpiler cannot pick the correct zero **literal**
  (`Config{}` vs `0` vs `nil` vs …) from a bare type name, and a per-`Err`-return `var` would
  collide when a function has multiple `Err` returns. Named returns give the zero for any T with no
  literal and no declaration, and remain idiomatic Go. A checker-backed compiler with full type
  info could emit the spec's literal form instead. **Revisit** when real type resolution exists.

### Ok-binding-unused → discard the success value with `_`
- **Kind:** assumption
- **Chose:** at a match site, capture `__goal_v, __goal_err := call` only when the Ok arm uses its
  binding; otherwise `_, __goal_err := call`. The error LHS is always `__goal_err` (the branch
  discriminant).
- **Over:** always binding `__goal_v` (risks an unused-variable compile error when the Ok arm ignores
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

### Lowering L4a (§8.7): guard rejects a STORED open-E Result with a located error
- **Kind:** decision (+ refusal-with-reason for the un-built half)
- **Did:** added the `storedresult` guard pass (`internal/pass/storedresult.go`), first in the
  pipeline. It refuses, with a located §8.7 error, an **open-E `Result[T, error]`** appearing as a
  **slice/array/map element** (`[]Result[...]`, `map[K]Result[...]`) or a **struct/enum field** — the
  two positions where it is unambiguously stored and silently miscompiles today (open-E lowers to a
  bare `(T, error)` tuple, which has no type name; the stored spelling `Result[int, error]` is
  undefined in the lowered Go). This turns a silent miscompile into an honest, located refusal — the
  prime directive. Running first, a rejected program never reaches the Result/Option/closed passes.
- **Scope nailed down by probing (what is and isn't guarded):**
  - **Open-E stored → rejected.** Always broken; the guard's target.
  - **Closed-E `Result[T, E]` (E≠error) → NOT guarded.** It lowers to the first-class Ok/Err **sum
    interface**; a closed-E Result field is a *supported, lowerable* pattern — the must-use depth check
    (`CheckMustUse` `dropped-stored-result`) relies on it, and guarding it broke that test. `isOpenEResult`
    distinguishes the two by the error parameter (bare `error` → open-E). *(A narrower real gap —
    constructing a closed-E Result `Result.Ok(...)` **in** a stored position is NOT lowered today, probe
    confirmed — is left to L4b, not conflated with the common declared/assigned field, to avoid a false
    rejection.)*
  - **`Option[T]` → NOT guarded.** Lowers to `*T`, which is first-class; stored Options are fine.
  - **A Result *parameter* matched in-body → NOT guarded.** The lexical checker treats it as valid
    surface (`testdata/check/02-match/result_match_ignored.goal`); whether it is "immediate" or "stored"
    is the open §8.7 question — not pre-judged. Likewise a bare local `var x Result[...]` (rare).
- **Refusal-with-reason (L4b deferred):** the actual sum-encoding fallback — open-E boxing, the
  immediate-vs-stored analysis (§9 open question), value-position constructor lowering, and store/read
  boundary conversions — is a deliberate design effort needing the rule signed off first. Probes showed
  "just emit the closed-E prelude for stored Results" is insufficient (constructors in stored position
  don't lower → a new silent miscompile), so the safe deliverable now is the diagnostic, not the boxing.
- **Proof:** `internal/pass/storedresult_test.go` — open-E slice/map/array/field rejected (located);
  function return, multi-return, Result param, closed-E field/slice, Option field, and Result-free
  structs all pass. Full suite green (B3 unbroken by the open-E narrowing).

### (checker) Must-use checks the unconsumed *call site*, not the fate of a bound value
- **Kind:** decision
- **Chose:** `internal/check/mustuse.go` rules **only** on a Result-returning direct call that
  stands alone as an expression statement (`parse(input)` on its own line). That is an Error
  `dropped-result`. Every consuming/nested context — `name := f(...)`, `match f(...) {…}`,
  `f(...)?`, `return f(...)`, `g(f(...))` — is recognized as a use by reading the token immediately
  before the callee (statement-start delimiter `{`/`}`/`;` vs `=`/`,`/`(`/`return`/`match`/…) and
  the token immediately after the matching `)` (a trailing `?` = consumed).
- **Over:** intra-function use-tracking of an assigned-then-unused Result (`r := parse(x)` where `r`
  is never read), which the slot doc lists as a candidate coverage target.
- **Why:** the statement-leading drop is provable lexically with zero false-positive risk. Proving
  "this bound variable is never used" needs flow analysis the lexical model cannot do soundly
  (closures, shadowing, reassignment, use in a later block); a false `dropped-result` would be a
  false guarantee, which the loop forbids. The obligation attaches to both `ModeResult` (open-E) and
  `ModeResultClosed` (closed-E) callees read from `Tables.FuncSignatures`. **Defer-boundary recorded
  as the go/types graduation point below.**

### (checker) Defer-boundary: `_ :=` discard and chained continuations → Warning
- **Kind:** decision
- **Chose:** two located deferrals instead of a verdict. `_ := f(...)` / `_ = f(...)` (a whole-Result
  discard) → Warning `unresolved-result-discard`, because the sanctioned explicit-discard surface for
  a Result is **not yet defined** (feature 03 SYNTAX.md §5 defers it together with this check) and its
  `(T, error)`-tuple lowering is unsettled — neither a use nor a sound drop can be asserted yet. A
  statement-leading Result call followed by an expression continuation (`f(...).x`, `f(...) + …`) →
  Warning `unresolved-result-use` (a Result has no usable surface besides match/?/bind, so this is
  unusual; rather than guess whether the continuation consumes the value, defer).
- **Over:** ruling `_ := f(...)` an Error (it is the natural Go discard and may become the sanctioned
  form) or a silent pass (would let a Result be dropped through `_`); and over guessing at chained
  uses.
- **Why:** "defer, never guess" — a false `dropped-result` is worse than an honest Warning. When the
  explicit-discard surface lands (feature 03 follow-up), this Warning is where the rule attaches.

### (checker) The assigned-then-unused class is the go/types graduation boundary for 03
- **Kind:** refusal (with reason)
- **Refused:** implementing assigned-then-dropped detection (`r := parse(x)` never read; a Result
  stored in a field/slice and never consumed; a Result passed onward and dropped by the callee) in
  this lexical loop.
- **Why:** these need real dataflow / type information — exactly what CHECKER-TODO.md flags as the
  point where 03 "graduates onto `go/types`." Doing it lexically would mean either false positives
  (unsound "unused" detection) or unbounded special-casing. Left **deferred**; the statement-level
  drop (the common, high-value case) is covered now, and the residue waits for the planned
  `go/ast` + `go/types` workstream — not started inside this loop.

### (checker) No `analyze.Tables` extension for 03
- **Kind:** decision
- **Chose:** reused `Tables.FuncSignatures` (the `Mode` of each in-file callee) as the only fact the
  must-use check needs; no table extension.
- **Why:** identifying a Result-returning callee is a name → Mode lookup the existing table already
  serves. Per-function spans (used by 06/02) weren't needed here — the obligation is decided from the
  call's immediate lexical neighbours, independent of which function encloses it.

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
  matching §8.4's `Some(u) -> &u`); otherwise `__goal_some := v; return &__goal_some`.
- **Over:** always emitting `&v` (illegal Go for literals/calls/index exprs — `&5`, `&f()`); always
  boxing (correct but noisier for the common `Some(u)` case).
- **Why:** Go forbids taking the address of a non-addressable expression, so a literal/call/index
  payload must be boxed through a temp (the idiomatic `v := …; &v`); a bare identifier can be
  addressed directly for cleaner output that matches the spec. Boxing also gives the Option its own
  copy (no aliasing). Edge: a single *constant* identifier isn't addressable and would still emit
  `&c` (rare); noted as a known limitation. The temp name `__goal_some` is shared (distinct `Some`
  returns sit in distinct branches/scopes).

### Option match: deref alias `x := *__goal_o` only when the Some binding is used
- **Kind:** assumption
- **Chose:** in the `Some` (non-nil) branch emit `x := *__goal_o` only if the arm uses `x`; otherwise
  omit it. `Some` → the `if` branch, `None` → `else`, regardless of source order.
- **Over:** always emitting the deref alias (unused-variable error when the Some arm ignores the
  value).
- **Why:** keeps generated Go compiling and clean; mirrors the "emit the binding only when used"
  discipline from 02/03. `__goal_o` is always used (the nil-test), so no guard needed there.

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
- **Chose:** a `?` in a `Result[_, error]` function is Result-mode (`return __goal_ok, __goal_err`); in
  an `Option[_]` function it is Option-mode (`return nil`). The transpiler maps each `?` to its
  enclosing function by source offset.
- **Over:** inferring the operand's type to decide mode (needs type inference the transpiler lacks).
- **Why:** `?` early-returns the *same kind* the enclosing function returns (the failure must have a
  compatible channel), so the return type determines the mode without any operand type inference.
  Matches Rust/Swift. A `?` outside a Result/Option function is a located error.

### Reuse `__goal_err` across Result `?`; fresh `__goal_oN` per Option `?`
- **Kind:** assumption
- **Chose:** Result `name := expr?` emits `name, __goal_err := expr` (reusing the named-return
  `__goal_err`, valid because `name` is new); the discard form uses an if-init to scope `__goal_err`.
  Option `?` uses a monotonic `__goal_o1`, `__goal_o2`, … per occurrence.
- **Over:** unique error temps per Result `?`; reusing one `__goal_o` for Option (which would
  redeclare with `:=`).
- **Why:** Go's `:=` redeclaration rule lets `name, __goal_err :=` reuse `__goal_err` when `name` is
  new (the spec's `cfg, err := ...` pattern), but an Option `__goal_o := ...` with no new LHS var
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

---

## 06-error-e — closed error type E (sum encoding + From)

### §9 From-conversion shape: `from func` modifier
- **Kind:** decision
- **Chose:** declare the `?`-conversion as `from func name(e Src) Dst { ... }` — an ordinary function
  with a `from` modifier; `?` auto-invokes it, resolved by its (Src)->Dst signature. `from` erases.
- **Over:** a dedicated `from Src to Dst { ... }` block (more novel); an unmarked function discovered
  by signature (zero syntax but implicit/magic and ambiguous if multiple match).
- **Why:** user chose via AskUserQuestion, weighing "obvious / not too foreign / explicit." The
  modifier sits in the established modifier-before-`func` slot (shared with `derive func`), so it's
  consistent and Go-shaped; the
  marker keeps it explicit (the conversion `?` reaches for is on the page), and `from` lands on
  Rust's `From`. The dedicated block was refused as the most foreign; signature-discovery refused as
  implicit/ambiguous. Open-E `?` needs no conversion — same `?` mechanism, with/without a conversion
  step (§3.3 line-to-protect). **Revisit** if a different conversion surface is later preferred.

### Closed-E uses no new construction/match/? syntax (one knob = E only)
- **Kind:** decision
- **Chose:** a closed error enum is just an `enum` used as the `E` of `Result[T, E]`; `Result.Ok/Err`,
  `match`, `?` are unchanged from 01-05. Open↔closed differ only in whether E is `error` or an enum.
- **Over:** any closed-specific construction/match/propagation syntax.
- **Why:** §3.3's one-mechanism-one-knob is the explicit line to protect — a second error system
  would break lint-as-policy. Reuses existing surface entirely.

### Closed-E Result encoding: injected generic `Result[T,E any]` + `Ok`/`Err`
- **Kind:** assumption
- **Chose:** inject one generic sum encoding (`type Result[T, E any] interface{ isResult() }`, generic
  `Ok[T,E]`/`Err[T,E]` structs with the marker) once per file; the `Result[T, E]` return type stays
  as written; construction/match/? use `Ok[T,E]`/`Err[T,E]`.
- **Over:** per-instantiation monomorphized types (`Ok_Config_ParseError`, …, §8.3 sketch style);
  an `any`-typed non-generic `Ok`/`Err` (loses payload typing).
- **Why:** the generic encoding keeps the signature unchanged and the output clean, and Go's type
  switch accepts concrete `case Ok[Config, ParseError]:`. Monomorphized names are verbose; `any`
  loses the typed payload the §8.1 encoding intends. Unused type param `E` in `Ok[T,E]` is legal Go.
  A real front-end with type resolution could choose either; this is the reference choice.

### T, E resolved from signatures; match/? scrutinee must be a direct call
- **Kind:** assumption
- **Chose:** construction takes (T, E) from the enclosing function's Result return; `match`/`?` take
  the callee's (T, E) from its signature (scrutinee must be a direct call `f(args)`) and the `?`
  early-return type from the enclosing function.
- **Over:** full type inference of arbitrary scrutinee expressions.
- **Why:** the reference transpiler has no type system; signatures are the only types it can read.
  The immediate case (match/? on a direct call) is what §8.3 targets. Stored Results, value-position
  match, and nested `Err`-variant patterns are out of scope and noted.

### Flat Ok/Err match only (nested error-variant patterns deferred)
- **Kind:** assumption
- **Chose:** `match` on a closed-E Result handles flat `Result.Ok(x)` / `Result.Err(e)`; to branch on
  the error enum's variants, compose `match e { ... }` (feature 02).
- **Over:** supporting the spec §3.3 nested form `Err(BadKey(k))` directly.
- **Why:** flat + composition covers the same ground and keeps the match lowering tractable; nested
  destructuring is an explicit later extension. Recorded so the divergence from the spec sample is
  deliberate (see SYNTAX.md "Open against spec").

### (checker) Defer-boundary: closedness on `Result.Err(E.Variant)`; From-totality on direct-call `?`
- **Kind:** decision
- **Chose:** the `internal/check/closed.go` slot proves two things lexically. (1) *Closedness:* every
  `Result.Err(X)` inside a closed-E function — `Result.Err(E.Variant)` or `E.Variant(payload…)` — must
  name a variant of *that function's* declared error enum E (`Tables.Enums[E].VSet`). A foreign enum is
  `err-outside-closed-enum`; a bad variant name is `unknown-error-variant`. (2) *From-totality:* every
  `?` in a closed-E function whose direct-call callee returns a *different* closed E must have a
  registered `from func` (`Tables.FromRegistry[[2]string{callee.E, caller.E}]`); a missing one is
  `missing-from-conversion`. Each diagnostic is located at the offending construct (the `?` token / the
  `Result` of `Result.Err`).
- **Over:** flow-tracking an `Err` value or a propagated error type through assignments and calls;
  resolving non-direct-call scrutinees.
- **Why:** mirrors exactly what the closed pass (`lowerClosedQuestions`/`lowerClosedCtors`) resolves
  lexically — caller E from the enclosing function span, callee E from the callee's signature, the Err
  value read directly at the construction site. That is the in-scope set the front-end already lowers
  (§8.3: match/? scrutinee is a direct call); a false "closed/total" is worse than an honest deferral.
- **Defers (located Warning, no Error):** a `?` whose callee is not an in-file closed-E Result function
  (out-of-package or non-direct-call scrutinee) → `unresolved-question-error`; a closed-E function whose
  error enum E is not declared in this file → `unresolved-error-enum`; a `Result.Err(X)` whose X is not
  a lexically-resolvable `E.Variant` construction (a bound var, a call, a larger expression) →
  `unresolved-err-value`. These are the points where the concrete error type genuinely needs type
  resolution; deferred to the planned `go/types` workstream, not faked here.

### (checker) No `analyze.Tables` extension for 06
- **Kind:** assumption
- **Chose:** read closedness/From-totality entirely from existing tables — `FuncSignatures` (Mode +
  T/E), `FromRegistry`, and `Enums[E].VSet`. The per-function body spans the closed pass uses
  (`funcSpans`/`sigAt`) are not importable from `check`, so the slot re-derives them locally
  (`closedSpans`/`sigAtOffset` over `scan.ScanFuncs`).
- **Over:** adding a precomputed `?`-site or Err-site index to the tables.
- **Why:** the facts already exist name-keyed; the only thing missing was the offset→enclosing-sig map,
  which is a trivial re-scan (same logic the pass package keeps private). Keeps the tables minimal.

---

## 07-implements — explicit interface assertion

### Surface inherited from feature 01: `implements X for T` (no new question)
- **Kind:** decision
- **Chose:** reuse the `implements X for T` standalone declaration form (pinned in feature 01) for
  the general additive assertion over any interface; did not open a new syntax question.
- **Over:** re-asking `implements X for T` vs a type-suffix annotation (`type T struct{…} implements
  X`) — the §3.4 "could equally be an annotation" alternative.
- **Why:** the user already chose the standalone form over the suffix annotation in feature 01 (Q4,
  sealed-interface form). Re-asking would re-litigate a settled choice. One `implements` spelling
  serves both roles (sealed-enum membership and ordinary-interface assertion), per §2/§3.4's shared
  capability. **Revisit** only if the user wants to reconsider the form globally.

### Emit the assertion; value form `T{}` vs pointer form `(*T)(nil)` by receiver scan
- **Kind:** assumption
- **Chose:** lower `implements X for T` to `var _ X = T{}`, or to `var _ X = (*T)(nil)` if any method
  of `T` in the file uses a pointer receiver. The reference transpiler emits the assertion but does
  not verify the methods (checker's job).
- **Over:** always emitting `T{}` (spec §8.5's literal form); emitting nothing (pure erasure).
- **Why:** §8.5 recommends emitting the free assertion, and the TODO says the reference transpiler
  emits it. Always-`T{}` would fail to compile for pointer-receiver method sets (only `*T` satisfies
  X then), so the receiver scan keeps the emitted Go compiling for both cases. Emitting nothing would
  drop the self-verifying property §8.5 values. The assertion's own compilation is the shadow of the
  checker's proof — a fortunate consequence (a wrong signature makes Go reject the assertion).

---

## 08-no-zero-value — required-field struct construction

### Explicit-defaults form spelled `...defaults`
- **Kind:** decision
- **Chose:** `...defaults` as the trailing explicit-defaults element ("set every unlisted field to
  its zero/default").
- **Over:** `_` (rest marker, reusing goal's `_` discard/rest convention); a bare `default` member;
  Rust's `..Default` struct-update tail.
- **Why:** user-selected via `AskUserQuestion`. `...defaults` leans on Go's existing `...` (spread)
  reading and names the intent ("defaults") so it is greppable. Refused `_` because `_` elsewhere
  in goal means "discard / don't care," which mismatches "give me the documented defaults"; refused
  bare `default` because it costs a new contextual keyword and reads like a field named `default`;
  refused `..Default` because `..` appears nowhere else in goal and `Default` implies a Default
  typeclass the language does not have. Closes the §9 "explicit-defaults form" open question.

### "Defaults" = Go zero values written explicitly; no per-field declared-default syntax
- **Kind:** decision
- **Chose:** `...defaults` fills each unset field with its *type's zero value*, emitted explicitly
  (`""`/`false`/`0`/`nil`, `T{}` for a named struct).
- **Over:** introducing a per-field declared-default syntax (e.g. `name string = "anon"`).
- **Why:** the spec frames the hatch as "I really do want **zero/defaults**" (§3.5) and §8.5 says it
  "lowers to explicit per-field default values." The spec defines **no** field-default declaration
  form, so inventing one would add a feature beyond the audit's scope ("Do NOT add features"). A
  declared-default facility is flagged in SYNTAX.md as a possible separate additive feature.

### Field zero recovered syntactically from the declared type (no type system)
- **Kind:** assumption
- **Chose:** `zeroLit` maps declared type text → zero: `*T`/`[]T`/`map`/`chan`/`func`/`interface`/
  `any`/`error` → `nil`; `[N]T` → `[N]T{}`; `string`→`""`; `bool`→`false`; numerics→`0`; in-file
  `type X struct` → `X{}`; in-file `type X interface` → `nil`; in-file alias/defined type → resolve
  underlying and recurse; unknown out-of-file named type → `T{}` (best-effort).
- **Over:** real type inference / resolving imported types.
- **Why:** the reference transpiler has no type system; declared types in the file are the only
  types it can read. Untyped constants (`0`/`""`/`false`) are assignable to defined types, so
  `type Role int` defaults correctly to `0`. The one unsafe spot — an out-of-file named *interface*
  wanting `nil` but getting `T{}` — is not syntactically recoverable, so it is deferred and noted
  rather than guessed. Field-type forms with internal spaces (func/chan-with-space) and grouped
  `type ( … )` decls are out of scope.

### `...defaults` rejects unsafe / no-safe-zero fields (located error)
- **Kind:** decision
- **Chose:** `...defaults` fills a field only when its zero is *safe* (usable as-is). When a defaulted
  field's zero is *unsafe* — `nil` map (panics on write), `nil` pointer (panics on deref), `nil`
  chan/func, a method-bearing named interface, or a sum type with no valid variant — the defaults
  pass raises a **located** compile error naming the first offending field. Safe zeros still fill
  silently: primitives, named structs, `[N]T`, **nil slices** (`range`/`len`/`append` all work),
  `error` (nil = success), `any`/bare `interface{}`, and int-backed `type Role int`. The check is
  **type-directed and scoped to defaulted fields only** — an explicitly-written value (even
  `x: nil`) is taken at the author's word.
- **Over:** (a) the original "`...defaults` = plain Go zero values, no judgement" behavior — rejected
  because it lets the escape hatch silently reintroduce the exact silent-zero footgun the feature
  exists to close; (b) the broader "reject every nil-valued zero" — rejected because nil slices and
  nil `error` are *safe, usable* values and flagging them would force noise (`tags: []string{}`) with
  no safety gain; (c) auto-allocating safe defaults (e.g. empty maps) — rejected because it makes
  `...defaults` mean something other than "zeros" and hides the decision the feature works to surface;
  (d) pervasive nil-elimination across all pointers — out of scope, still the deferred §5 decision.
- **Why:** user-driven ("the *goal* is to reject anything unsafe; all zero-valued pointers should be
  rejected"). The escape for a genuinely-optional reference is `Option[T]` (§3.6), which the language
  already ships — so "I want maybe-absent" has a safe home and "I just didn't set it" is what
  `...defaults` now refuses. Goes slightly beyond §3.5's "`...defaults` = zero values" framing,
  narrowly and deliberately (rejects *unsafe* zeros, fills safe ones). Implemented in both the real
  pipeline pass (`internal/pass/defaults.go`, via `analyze.Sealed`/`Enums`/`TypeDecls`) and the
  standalone reference transpiler; surfaced as a playground error demo. See SYNTAX/TRANSPILE.

### Transpiler does not reject incomplete literals (checker's job)
- **Kind:** decision
- **Chose:** complete struct literals pass through verbatim; the transpiler only expands
  `...defaults` (and now rejects unsafe defaults, above). It still does **not** reject missing fields
  or verify field names against the struct declaration.
- **Over:** implementing field-completeness validation in the reference transpiler.
- **Why:** field-completeness is the erased static guarantee (§8.5: the feature "only ever rejected
  source") and per the audit's "NO error checking yet" constraint, that check is the checker's job.
  (Unsafe-default rejection is the one exception the pass owns, because an unsafe zero reaching
  codegen — unlike a merely-incomplete literal the checker will catch — would defeat the feature.)

### Checker (`internal/check/fields.go`) — field-completeness defer-boundary
- **Kind:** decision
- **Chose:** the checker fires `missing-field` (Error) only when the literal's type is **named at the
  site**: a struct literal `T{…}` where `T` is an in-file `type T struct` (read from
  `analyze.Tables.Structs`), or a variant construction `Enum.Variant(…)` where `Enum` is an in-file
  `enum` (read from `analyze.Tables.Enums[…].FieldSet`/`.Variants`). Completeness = every declared
  field appears as a keyed element at the literal's own depth, **unless** the struct literal carries
  the `...defaults` spread (which the checker treats as complete-by-construction — the defaults pass
  owns expanding and unsafe-zero-rejecting it). Present-key detection mirrors the defaults pass's
  `presentFields` (an `IDENT :` at brace depth 0).
- **Over:** chasing the type of an unnamed/inferred literal (a bare `{…}` element of a typed outer
  literal, a `:=` whose type isn't at the site, a positionally-passed literal) — and over enforcing
  variant-construction completeness only when the surface is unambiguous.
- **Why:** "defer, never guess" (checker contract). A literal whose type isn't resolvable in-file is
  surfaced as a located **Warning** (`unresolved-literal-type`, "field-completeness deferred") naming
  the unresolved type, never an Error — a false "complete" is worse than an honest "cannot tell." This
  needed **no `analyze.Tables` extension**: `Structs` and `Enums` already carry the field sets.

### Variant construction `Enum.Variant(…)` is checked for completeness, paren-form, no `...defaults`
- **Kind:** assumption
- **Chose:** treat enum variant construction as the **paren** surface `Enum.Variant(field: expr, …)`
  (what `internal/pass/enums.go` actually lowers), not the brace form `Enum.Variant{…}` the
  CHECKER-TODO line writes shorthand. A variant has **no** `...defaults` escape (the defaults pass only
  recognizes `...defaults` inside `{`-braces), so every declared field of a data-carrying variant must
  be named; a data-less variant (`Shape.Dot`) is trivially complete. The enums-pass `construct`/
  `parseArgs` lowering silently zero-fills omitted variant args, so this completeness check is a
  genuine erased guarantee landing here.
- **Over:** (a) reading `Enum.Variant{…}` brace-form literally — that form does not exist on the
  surface; (b) inventing a `...defaults` escape for variants — none is specified.
- **Why:** keeps the check tied to the real lowering. Flagged as an assumption (not a hard decision)
  because the TODO's `{…}` shorthand could be read either way; the user can veto routing variant
  completeness through this slot.

### File-layout / `Code` scheme for the fields slot
- **Kind:** assumption
- **Chose:** `Feature` = `"08-no-zero-value"`; `Code` = `"missing-field"` for a violation and
  `"unresolved-literal-type"` for a deferral Warning. Brace/keyword/func-body/decl-body braces that
  share the `IDENT {` shape (`func f() T {`, `enum E { … }`, `type T struct { … }`, `struct{}`,
  `interface{}`, control-flow) are excluded via `scan.ScanFuncs` body-brace indices and an
  enum/struct decl-span scan, so a return type or a variant field declaration is never misread as a
  literal.
- **Over:** no naming scheme was fixed by the spec.
- **Why:** stable, greppable codes per the slot doc; the brace-disambiguation is the one place a
  lexical literal-finder can go wrong, so it is guarded explicitly and recorded for review.

### Fix: `...derive(src)` spread and match payload-binding arms no longer false-flag (was a follow-up)
- **Kind:** decision
- **Chose:** a dedicated false-positive fix to `checkFields` closing the two shared-harness
  interactions earlier slots worked around in testdata (recorded in §02 and §12). Two changes, both
  lexical, both reusing the lowering passes' own locators (assert, not splice):
  1. **`...derive(src)` is complete-by-construction**, exactly like `...defaults`. The spread detector
     (renamed `litHasDefaults` → `litHasCompletingSpread`) now recognizes the four-token
     `.` `.` `.` `derive` form (the same shape `internal/pass/derive.go` expands) at the literal's own
     brace depth, in addition to `...defaults`. A `T{ …, ...derive(s) }` body literal of a `derive func`
     no longer reads as omitting the unnamed fields — the derive pass owns expanding/rejecting them, so
     08 must not assert completeness over them (the exact parallel of the `...defaults` carve-out).
  2. **Variant payload-binding arms in `match` are not constructions.** A `Status.Active(a) => …` arm
     binds the narrowed value to `a`; it is lexically identical to a variant construction
     `Status.Active(a)` but must NOT be field-checked. New `matchPatternSpans` collects every match
     arm-pattern token span — mirroring `internal/pass/match.go`'s `parseMatchArms` (depth-0 `=>`
     arrows) and `internal/pass/result.go`'s `patternStart` (lifted verbatim as `matchPatternStart`) —
     and `checkFields` skips any `Enum.Variant(…)` site whose qualifier token falls inside an
     arm-pattern span.
- **Defer-never-guess preserved:** neither change can manufacture a false Error. The `...derive`
  carve-out only *suppresses* a would-be Error on a recognized spread (same risk profile as
  `...defaults`). The match-arm skip only *suppresses* the variant check inside a lexically-bounded
  arm pattern (the same `=>`/`patternStart` machinery the proven lowering uses to read arms); outside
  that span nothing changes. Where the construct genuinely cannot be told apart (a literal with no
  type at the site), the pre-existing `unresolved-literal-type` Warning still defers rather than guess.
- **Over:** leaving the two interactions worked-around in sibling testdata (the prior state — honest but
  it meant a real surface shape, a `...derive` body literal or a payload-binding arm, drew a spurious
  08 Error under the shared harness); broadening the match-arm skip to suppress 08 anywhere a `match`
  appears (would risk under-checking real constructions in arm *bodies* — the skip is scoped to the
  pattern span only, not the whole arm block).
- **Why:** a false Error is worse than a deferral, and these were *known* false Errors on valid
  programs. Both reuse the owning pass's locators so the check's view of a `...derive` spread and a
  match arm stays identical to the lowering's. **No `analyze.Tables` extension**, no new diagnostic
  code. Testdata added under `testdata/check/08-no-zero-value/`: `derive_spread_optout.goal` (a real
  `derive func` body using `...derive(s)`, clean) and `match_binding_arm.goal` (a payload-binding
  `Status.Active(a) => a.since` arm, clean). All existing 08 positive/negative cases stay green, and
  the full suite (`go vet`, `./internal/check/`, `./...`) passes. Resolves the follow-ups noted in
  §02 ("File-layout / `Code` scheme + the 08-fields cross-check interaction") and §12 ("testdata avoids
  `...derive` literals that trip the 08 field-completeness check").

---

## 09-pure — CUT (not in v1)

### `pure` removed from the v1 surface — the value is gated on a deferred optimizer, and the cheap version lies
- **Kind:** decision (reversal of the earlier "adopt `pure func`" decisions)
- **Chose:** remove `pure` entirely — delete the live pass, drop it from the pipeline/docs/playground,
  and **move the full feature (spec, spike transpiler, examples) intact to `features/_cut/09-pure/`**
  (a new convention for audited-but-cut work). Feature 09's number stays vacant; nothing is renumbered.
- **Over:** keeping the erase-only marker; or building a real effect checker now.
- **Why:** an audit of how purity *checking* would actually work found a dead end on this architecture:
  - §4.2 deferred the entire definition of "effect" to a checker that was never built — there was no
    rule to implement.
  - The transpiler is a token recognizer with no type/scope/escape/cross-package analysis, so it
    cannot distinguish local mutation from aliased mutation, resolve a call's target package, or see
    through interface dispatch. **Sound** purity checking is not buildable on it.
  - "Cheap" and "sound" are mutually exclusive: a cheap syntactic denylist is unsound (a guarantee
    that lies — worse than nothing, since readers/models trust it); a sound check needs a real parse
    pass, a value-only-parameter restriction, **and** a maintained per-Go-release FFI purity manifest.
  - The only concrete payoff — auto-parallelization / memoization (§8.5) — was already "not v1," so
    cost and benefit were both deferred and bound to the expensive version. The residual
    documentation value is marginal (an LLM already infers leaf-function purity).
- **Revisit:** reconsider **together with** the §8.5 optimizer — the only consumer that turns purity
  into real value. Full rationale + preserved spike: `features/_cut/README.md`.

---

## 10-assert — runtime assertions

### Message form: printf-style with a bare fallback
- **Kind:** decision
- **Chose:** `assert cond [, "fmt", args...]` — bare `assert cond` is valid (auto expr-text message
  only); an optional printf-style format string + args appends a formatted explanation.
- **Over:** "bare only" (no message argument ever); "single optional string message"
  (`assert cond, "msg"`, no interpolation).
- **Why:** user-selected via `AskUserQuestion`. The auto-included source expression text (§8.6)
  already gives located feedback for free, so the bare form is the low-ceremony common case; the
  printf form (Go's `t.Errorf` / Python `assert cond, msg` idiom) lets the failure interpolate the
  offending values — the highest-information runtime failure — without spending familiarity budget
  on novel punctuation. Single-string was subsumed by printf (a bare format string is a valid printf
  call). Closes the §9 assert question.

### Lowering: `if !(cond) { panic(...) }`; expr text quoted, never a format string
- **Kind:** decision
- **Chose:** bare → `panic(strconv.Quote("assertion failed: "+cond))`; message →
  `panic(<quoted prefix> + fmt.Sprintf(<msg>))`. The condition text is always emitted via
  `strconv.Quote` (a string literal), and the user format string is Sprintf'd separately and
  concatenated.
- **Over:** baking the expr text into the Sprintf format string (e.g.
  `fmt.Sprintf("assertion failed: <expr>: <userfmt>", args...)`).
- **Why:** §8.6's runtime-preserved lowering is `if !(cond) { panic("assertion failed: <expr>") }`.
  Concatenating a quoted expr text instead of embedding it in the format string keeps a `%` in the
  condition (e.g. `n%2 == 0`) from being misread as a printf verb — a real correctness trap the
  worked examples exercise.

### Inject `import "fmt"` when a message assert needs it
- **Kind:** assumption
- **Chose:** if any message-bearing assert is emitted and the file does not already import `"fmt"`,
  insert `import "fmt"` after the `package` clause (detected by scanning for a `"fmt"` import token).
- **Over:** requiring the source to pre-import fmt; always importing fmt.
- **Why:** the printf lowering needs fmt, and real goal code shouldn't have to import it manually for
  an assert message. Best-effort detection (a dot/named-alias fmt import could double-import) is
  acceptable for the reference and stays gofmt-valid; the real frontend tightens it. Bare asserts
  need no import.

### Build-tag strip toggle reserved, not built (v1 always emits)
- **Kind:** refusal
- **Chose:** n/a (nothing built).
- **Over:** implementing the §8.6 build-tag that strips asserts from release builds.
- **Why:** NEXT-SESSION confirms stripping is not v1-critical. The reference always emits the
  runtime check; the strip strategy is documented in TRANSPILE.md as reserved. Likewise the §4.3
  statically-checkable assert subset and §5 contracts are reserved syntax, not built.

### Checker (10-assert): the static-provable subset, minimal slice
- **Kind:** decision
- **Chose:** `checkAssert` folds only constant conditions with no free names, in exactly two shapes:
  (1) a bare boolean literal — `assert false` → Error `assert-always-false` (guaranteed panic),
  `assert true` → Warning `assert-always-true` (dead code); (2) a comparison of two integer literals
  `LIT OP LIT` for OP ∈ {`<`,`<=`,`>`,`>=`,`==`,`!=`} — folded to a constant, false → Error, true →
  Warning. Located at the `assert` keyword. The condition is bounded exactly as the assert pass does
  (keyword at `scan.IsLineStart`, statement to `scan.NextNewline`, condition = left of the first
  top-level comma).
- **Over:** any broader folding (floats, unary `!`/`-`, parens, multi-term arithmetic, identifier
  resolution).
- **Why:** §4.3 / SYNTAX.md "Reserved" deliberately scoped this to a small static-checkable subset
  and *refused* general Dafny-style proving. A bare boolean literal and a two-literal integer
  comparison are the only shapes provable purely lexically with zero risk of diverging from Go's
  runtime evaluation. Tautologies are a Warning (not an Error) per CHECKER-TODO — the program is
  valid, the check is just dead.
- **Defer-boundary (emit nothing, by design):** any non-constant condition — an identifier, call, or
  field access — draws **no diagnostic at all**, not even a Warning. This differs from the other
  checks' "located Warning on deferral": here a runtime-checked assert over a variable is the
  *intended* v1 behavior (SYNTAX.md), so there is nothing unresolved to surface. Float comparisons,
  unary/paren/multi-term expressions, and non-decimal-or-over-large integer literals are also left to
  runtime — folding them risks a false "always panics", and a false guarantee is worse than an
  unflagged decidable case.
- **No `analyze.Tables` extension:** constant folding reads only the source tokens; `t` is unused.

### Checker assumption: `Code` scheme and testdata layout (10-assert)
- **Kind:** assumption
- **Chose:** two greppable codes — `assert-always-false` (Error) and `assert-always-true` (Warning);
  messages quote the offending condition text verbatim (`assert condition \`...\` is statically
  false`). Testdata split by verdict: `always_false.goal` (Errors, claimed by `// want`),
  `always_true.goal` (tautology Warnings, claimed), `runtime_ok.goal` (non-constant conditions, no
  markers — pins the defer-boundary against false positives, incl. the `assert cond, msg, args...`
  message form and a `%` in the condition).
- **Why:** consistent with the per-feature `Code` convention; the user can veto the naming or ask for
  a wider fold scope.

---

## 11-doctests — runnable doctests

### Doctest marker: `///` triple-slash doc lines
- **Kind:** decision
- **Chose:** `///` doc lines carry doctests (the spec sample's form).
- **Over:** reusing standard Go `//` doc comments with a `>>>` line.
- **Why:** user-selected via `AskUserQuestion`. `///` visually flags doctest-bearing docs and never
  collides with ordinary `//` comments, so a stray `>>>` in prose can't accidentally become a test.
  Small deliberate familiarity spend (Rust/C# idiom; Go uses only `//`), justified because doctests
  are the top feedback band. Bonus: `///` is itself a valid Go line comment, so the original source
  compiles untouched.

### Expectation form: expected value on the next line
- **Kind:** decision
- **Chose:** `>>> <expr>` followed by an expected-output line (Python/Rust doctest transcript).
- **Over:** inline equality `>>> add(2, 3) == 5`.
- **Why:** user-selected. Reads like a REPL transcript and lowers directly to §8.6's
  `got := <expr>; want := <expected>; if got != want { t.Errorf(...) }` shape.

### Emit a generated `_test.go`; `transpile()` returns that test file
- **Kind:** decision
- **Chose:** extract doctests into a generated `<base>_doctest_test.go` that runs under `go test`;
  the reference `transpile()` returns that test file (the feature's product), and each
  `*.go.expected` holds the generated test file. The original code file is unchanged (doc comments
  are valid Go and pass through).
- **Over:** inlining checks into the code path; transforming the code file; a bespoke runner.
- **Why:** §8.6 specifies generated `_test.go` under `go test` — the idiomatic free runner that makes
  §4.1's "no way to silently not-run" true by construction. The two-output model is inherent to the
  feature; returning the test file keeps the shared single-output test harness unchanged while
  asserting the only interesting artifact. Verified by actually running `go test` on each example —
  the doctests execute and pass.

### Comments read from source, not tokens; expected is a Go expression; free funcs only
- **Kind:** assumption
- **Chose:** scan the raw source string for `///` blocks (the lexer skips comments); the expected
  line is a Go expression/literal lowered to `want := <expected>`; comparison is `!=`; doctests
  attach to the free function declared immediately below the block. Methods, multi-line expected
  output, non-comparable results (would need `reflect.DeepEqual`), and goscript's own runner are
  out of scope.
- **Over:** Python-style bare printed output (needs a value-print/parse layer); supporting methods
  and multi-line output in v1.
- **Why:** comments are invisible to the token stream, so source scanning is required for this
  feature (NEXT-SESSION flagged this). A Go-expression expected keeps the generated test trivially
  correct without a printing layer. The audit pins the **Go transpile path only**; goscript doctests
  are a separate workstream (§4.1, §9).

---

## 12-derive-convert — type-directed, completeness-checked conversion (post-audit)

### Feature originates from design exploration, not the spec
- **Kind:** decision
- **Chose:** add feature 12 (beyond the original 11) based on auditing the `telegraph/public-api`
  3-layer codebase + goverter usage during this session.
- **Over:** leaving the audit at 11 features; or treating cross-layer conversion as out of scope.
- **Why:** the audit showed conversion is a real, on-thesis friction (silent zero-value fields on a
  forgotten mapping; goverter's stringly annotations; silent enum `default:` fallbacks; silent
  int clamps). `goal-design-spec.md` is read-only (guardrail), so feature 12's "spec" is this ledger
  + its SYNTAX/TRANSPILE docs. Recorded so the divergence from the spec-driven 01–11 is deliberate.

### Key reframe: the value is type-directed conversion + completeness, NOT 1:1 field auto-mapping
- **Kind:** decision
- **Chose:** the feature centers on (a) a registry of type-pair conversions and (b) a
  completeness-checked derived conversion.
- **Over:** a "goverter-lite" feature whose pitch is auto-mapping same-named fields.
- **Why:** the audit (patterns, pmk_upgrade, booking_request_history, configurable_execution) found
  FREE 1:1 fields are the minority (~0–35%, often 0 at the persistence/view boundary) AND goverter
  already maps them for free. ~75–85% of fields are non-trivial, but ~90% of those collapse to ~6
  recurring type-pairs (UUID, three optionality reps, timestamps, int widths, JSON blobs, enums).
  So the leverage is the type-pair layer + the completeness guarantee, not the field-name layer.

### Syntax: `derive func` (bodyless) + partial-literal `...derive(src)` with `_` skip
- **Kind:** decision
- **Chose:** declaration `derive func NAME(src S) T`; bodyless = sugar for `{ return T{ ...derive(src) } }`;
  exceptions via a returned partial literal where `Field: expr` overrides (verbatim), `Field: _`
  skips, and `...derive(src)` fills the rest (completeness-checked).
- **Over (Q1):** bodyless `from func` (no body ⇒ derive — implicit, collides with leaf bodies);
  `convert A to B as name` (non-func-shaped statement).
- **Over (Q2):** a clause block (`ignore`/`from`/`=`).
- **Why:** user-selected via `AskUserQuestion`. `derive func` parallels the `from func`
  modifier convention and is distinct from a bodied leaf; `...derive` is the exact parallel of
  feature 08's `...defaults` (deriving IS complete construction), and `_` reuses goal's skip marker.
  **Reconciliation:** the two picks (bodyless decl vs partial literal) are the same construct at two
  ends of a spectrum; canonicalized as `{ return T{ ...derive(src) } }` (Go-shaped) rather than the
  `= expr` shorthand shown in the Q2 preview, to keep func bodies Go-consistent.

### Three conversion tiers; default narrowing = invariant-checked total (assert), not Result
- **Kind:** decision
- **Chose:** tiers encoded in the leaf's return type — lossless-total (`A→B`), invariant-checked
  total (`A→B` with internal `assert`, feature 10), recoverable-fallible (`A→Result[B,E]`, feature
  03/05, propagated by `?`). Default for an ambiguous narrowing (e.g. `int→int32`) = assert-total.
- **Over:** always-checked (Result everywhere — viral friction); silently clamping (band-3 footgun,
  refused); refinement/range types making narrowing compile-provable (too heavy — same trap §4.3
  refused for static asserts).
- **Why:** goal's three feedback bands — compile error > located runtime failure > silent wrong
  value (refused). A value-dependent narrowing can't be band-1 generically, so the choice is panic
  (assert: fatal-but-contained, total signature, no Result ripple) vs Result (recoverable-but-viral).
  Author picks by the conversion's nature ("bug vs expected bad input"), once per type-pair. Default
  assert-total because a silent-clamp replacement should fail loud-but-local unless explicitly opted
  into Result. The audit's `safeIntToInt32` silent clamp is exactly the band-3 case being killed.

### Generics: container recursion is a built-in deriver rule; user-facing generic `from` reserved
- **Kind:** decision
- **Chose:** `[]A→[]B`, `map[K]A→map[K]B`, `Option[A]→Option[B]`, and nested-struct conversion are a
  built-in recursion rule of the deriver (the user writes only the leaf `A→B`). `Option[T]↔*T` is a
  built-in unconstrained generic bridge. User-facing constrained generics
  (`from func [A,B] where convert(A,B)`) are reserved, not built.
- **Over:** requiring users to declare generic collection conversions; shipping full constrained
  generics in v1.
- **Why:** the audit showed the collection/nesting cases are real but few, and they decompose as
  "apply the element conversion in a loop" — pure transpile-time monomorphization, which goal already
  does. Making recursion a deriver built-in removes the need for the scary "a conversion A→B exists"
  constraint (no Go analog) from the user surface entirely. `Option[T]↔*T` is unconstrained → trivial
  Go generic. Reserve the constrained generic per §4.4 design-in-now/build-later.

### Dispatch: target-directed, one canonical conversion per (A,B), concrete beats generic
- **Kind:** decision
- **Chose:** a conversion is selected by the (source-field-type → target-field-type) pair (target
  known from the destination field). Registry holds one canonical conversion per ordered pair; a
  site needing different behavior calls a named conversion explicitly. Concrete beats built-in
  generic on overlap.
- **Why:** target-directed dispatch makes `Option[string]→*string` (generic) vs `Option[string]→
  null.String` (concrete) unambiguous; one-per-pair keeps `...derive` deterministic; concrete-beats-
  generic is the familiar overload rule (Rust/Swift).

### json.RawMessage blobs stay first-class opaque fields (do NOT force typing them)
- **Kind:** refusal
- **Chose:** n/a — explicitly NOT building blob-elimination pressure.
- **Over:** pushing authors to model `json.RawMessage` payloads as typed structs.
- **Why:** an opaque blob can be a legitimate modeling choice (genuinely heterogeneous/schemaless
  data), and you can't distinguish "escape-hatch blob" from "legitimate blob" from outside. The
  registry handles blobs via blob↔blob / blob↔string conversions; completeness checks structural
  completeness of the conversion, never the blob's contents — which is correct scope, not a gap.
  (Surfaced by user pushback during exploration: "it's ok to have a raw-JSON property … are we
  going too far?" — yes, on that point.)

### Reference transpiler scope (no full checker; lowered-form examples)
- **Kind:** assumption
- **Chose:** the transpiler builds the registry from `from func` signatures (strips `from`), parses
  struct fields (reusing feature 08), and expands `derive func` to `var out T` + field-by-field
  assignment (registry-resolved, target-directed), threading `?`/errors via `__goal_vN` for fallible
  conversions and emitting a `make`+loop for slice recursion. Unresolvable fields are DEFERRED with a
  located error (never silently zero). Examples use lowered Go forms (`(T,error)`, `*string`, local
  UUID/NullString stand-ins) for standalone compilation.
- **Over:** implementing full completeness/type checking here; depending on features 03/04 lowering.
- **Why:** the audit's no-checking-yet constraint puts the totality proof in the checker; the
  transpiler's job is valid goal → correct Go. Lowered-form examples keep the feature standalone (the
  same self-containment discipline prior features used). map/Option/nested recursion follow the slice
  rule and are noted but minimal in v1. Verified: `go test` passes (3/3), all generated packages
  compile + vet clean, AND behavioral tests confirm the conversions produce correct values and thread
  errors (empty ID → error).

### Checker (12-derive-convert): conversion-totality slot — defer-boundary
- **Kind:** decision
- **Chose:** `checkConvert` (`internal/check/convert.go`) walks every `derive func` exactly the way
  `pass.expandDerives` does — same source-param / return-type / fallibility parse, same `parseOverrides`
  body read — and for each unmentioned target field asserts resolvability with `resolveField`'s strategy
  order: same type → registered `from func` → built-in `[]A→[]B` slice recursion. A target field that is
  neither overridden, `_`-skipped, nor resolvable is an **Error**. Three Error codes: `unsourced-field`
  (no same-named source field), `unbridged-field` (sourced but no conversion for the differing type pair),
  `fallible-in-total-derive` (the only registered conversion is fallible but the derive is total —
  mirrors the pass's "declare the derive returning `(T, error)`" rejection).
- **Defer-boundary (located Warning, never a false Error):** `unresolved-derive-type` when the target
  *or source* type is not an in-file `struct` (out-of-package — field set unreadable); `unresolved-derive-field`
  when a `...derive` field's resolution needs a recursion class the v1 deriver keeps minimal — `map[…]`,
  `Option[…]`, pointer (`Option[T]↔*T` bridge), or a slice whose element pair has no total conversion.
  Those classes may yet be total via a rule this lexical check doesn't model, so they are deferred, not
  flagged. This is exactly the audit's "slice implemented; map/Option/nested + the two bespoke shapes
  (pmk_upgrade, patterns JSON) deferred" boundary, surfaced as a located Warning instead of silence.
- **Over:** proving totality of map/Option/nested recursion lexically (needs real type structure — defer
  to the planned `go/types` workstream); treating an out-of-package target as incomplete (a false Error).
- **Why:** a false "this conversion is incomplete" is as bad as a false "complete"; deferring the
  unprovable classes keeps the guarantee honest. Located at the `derive` keyword (the construct erased by
  lowering). No `analyze.Tables` extension — `Structs` + `FromRegistry` carry every fact.

### Checker (12): private ports of the derive pass's parse helpers
- **Kind:** assumption
- **Chose:** `convert.go` carries byte-for-byte private copies of `pass.parseOverrides`, `splitReturn`,
  `findField`, `indexOfTok`, `tokenAtOffset` (the derive pass's locators), since they are unexported and
  the check package must not import `internal/pass`.
- **Over:** exporting them from `internal/pass` (would change a file outside this slot's scope and couple
  the checker to the lowering package); re-deriving the parse differently (risks the check reading a
  `derive func` differently than the lowering does — the exact false-guarantee hazard).
- **Why:** the spine's reuse contract is "lift the pass's locating logic, then assert instead of splice."
  Mirroring the helpers verbatim keeps the check's view of a `derive func` identical to the lowering's.

### Checker (12): testdata avoids `...derive` literals that trip the 08 field-completeness check
- **Kind:** assumption
- **Chose:** the `...derive` spread path is exercised by **bodyless** derive testdata (no literal); the
  body-form clean case names every target field explicitly. Rationale: the 08-fields check
  (`checkFields`) runs in the same harness and recognizes only `...defaults` as a completeness spread, so
  a `T{ …, ...derive(src) }` literal reads to it as omitting the unnamed fields → a spurious
  `missing-field` Error on the shared case. Bodyless derives have no literal, so 08 never sees them.
- **Over:** editing `fields.go` to also recognize `...derive` (out of this slot's scope); claiming the
  08 error in the 12 case's markers (would mis-attribute it).
- **Why:** same shared-harness interaction the 02-match slot already noted (payload-binding arms tripping
  08); the conservative move is to write 12 testdata that does not provoke a sibling check. The
  `...derive`→08 interaction is real surface (a body literal with `...derive` *will* draw an 08
  `missing-field` today) and is worth a follow-up — recorded here, not silently worked around in code.
- **Resolved (08-fields fix):** the follow-up is done — `checkFields` now treats `...derive(src)` as
  complete-by-construction (same carve-out as `...defaults`), so a `T{ …, ...derive(s) }` body literal
  no longer draws a spurious 08 `missing-field`. Bodied derives with `...derive` spreads are now safe
  under the shared harness (proven by `testdata/check/08-no-zero-value/derive_spread_optout.goal`); the
  bodyless-only constraint above is no longer required. See §08 "Fix: `...derive(src)` spread and match
  payload-binding arms no longer false-flag."

### Lowering L1 (12-derive-convert): in-package map / pointer / array recursion
- **Kind:** decision
- **Did:** extended `pass.resolveField` (`internal/pass/derive.go`) — which previously lowered only
  same-type, a registered `from func`, and `[]A→[]B` — to also lower three in-package container shapes,
  each reusing the slice case's `elemConv` (total element conversion only, v1):
  - **`*A→*B`** (and **`Option[A]→Option[B]`**, which lowers to the same `*` strategy): a nil source
    stays the target's nil; a non-nil one is converted and re-addressed
    (`if src != nil { p := conv(*src); dst = &p }`). `ptrInner` recognizes both `*X` and `Option[X]` —
    folding Option in *without* emitting an `Option[…]` spelling, which matters because derive runs at
    pass 7, **after** the option pass (4) lowers `Option[T]→*T`, so any `Option[…]` text the deriver
    emitted would be left un-lowered and break the Go.
  - **`[N]A→[N]B`** (same length, compared as text): the target array is already zero; convert in place
    (`for i := range src { dst[i] = conv(src[i]) }`). No `make` (arrays are values). `arrElem` rejects
    slices (`[]E`).
  - **`map[K]A→map[K]B`** (same key type): `make(map[K]B, len(src))` then convert each value.
- **Scope/defer (unchanged v1 boundary):** element/value/pointee conversions must be **total** (identity
  or a non-fallible `from func`); a fallible or unresolved leaf still errors with the located
  "no conversion …" message — same rule the slice case already held. Nested containers
  (`[][]A`, `map[K][]A`) hit `elemConv`'s identity-or-registry-only limit and defer. **Out-of-package**
  target/source structs remain refused (`genConversion` reads in-package `t.Structs`) — that is the
  type-gated L5 case, not L1.
- **Why total-only:** matches the existing slice contract and the checker's documented
  `unresolved-derive-field` defer-boundary (§"Checker (12) … defer-boundary"); a partial/fallible
  container conversion needs explicit error-propagation plumbing not in v1. The lexical `checkConvert`
  (a separate check, not the pass) still defers these classes as Warnings — it does not consult the
  pass — so its behavior is unchanged; this unit makes the *lowering* succeed where the program is
  actually total, which is what B4's depth check will later verify.
- **Proof:** round-trip case `testdata/derive_container_recursion.goal` + `.go.expected` exercises all
  five field kinds (slice/map/array/pointer/Option-as-pointer) through one bodyless `derive func`; the
  expected Go compiles clean (`go vet`). Full suite green.

### Lowering L2 (12-derive-convert): nested in-package struct recursion
- **Kind:** decision
- **Did:** `pass.resolveField` now lowers a field whose source and target types are **both structs
  declared in this package** (`A→B`, no `from func`) by recursing field-by-field: it declares a temp
  `var __goal_sN B`, fills each target field via `deriveBody` (same strategy order, recursively), and
  assigns the temp. A registered `from func A→B` still wins (checked before the recursion). Fallible
  leaves propagate through the recursion via the outer derive's `return out, err` (so a nested fallible
  field requires the top-level derive to be `(T, error)`, same as a flat one). `resolveField`'s signature
  changed from taking just `FromRegistry` to the full `*analyze.Tables` (it needs `Structs`).
- **Matching checker change (required for check/build consistency):** the lexical `checkConvert`
  (`internal/check/convert.go`) previously emitted an **`unbridged-field` Error** for any concrete named
  type pair with no registry entry — including a struct→struct pair its `isDeferredShape` predicate
  (map/Option/pointer only) did not cover, **contradicting its own doc** which lists "nested-struct
  auto-recursion" as deferred. Left unchanged, `goal check` would now false-Error on a nested-struct
  derive that `goal build` successfully lowers. Fix: `resolvableField` now **defers a struct→struct pair
  (both in `Tables.Structs`)** as an `unresolved-derive-field` Warning, before the `unbridged-field`
  Error. A struct→non-struct pair (e.g. `UUID`→`string`) is *not* both-structs, so it stays an Error —
  `unbridged_field.goal` is preserved. Proving nested totality lexically stays the depth checker's job
  (B4); the checker defers, the pass lowers (or errors at lowering on a genuinely unbridged deep leaf,
  failing `goal build`), so no guarantee is lost.
- **Scope/defer:** covers a **direct struct-typed field** `A→B`. A pointer/slice/map/array *of* a
  nested struct (`*A→*B`, `[]A→[]B` where `A`,`B` are structs needing recursion) stays deferred: those
  go through `elemConv`, which renders a pure expression (identity or a `from func` call) and cannot
  express the statement-level temp build a struct recursion needs. Out-of-package structs remain refused
  (in-package `t.Structs` only) — the type-gated L5 case. No infinite-recursion guard needed: value
  struct nesting is acyclic in Go, and pointer cycles terminate at `sf==tf`/registry.
- **Proof:** round-trip case `testdata/derive_nested_struct.goal` + `.go.expected` (`Person→PersonV2`
  with a nested `Addr→AddrV2`, `Zip` bridged by a registered `string→Code`); expected compiles clean.
  Check case `testdata/check/12-derive-convert/defer_nested_struct.goal` pins the new deferral (no false
  Error). `unbridged_field.goal` still Errors (struct→non-struct). Full suite green.

## 07-implements — surface-syntax revision

### `implements` moves from standalone declaration to inline struct clause
- **Kind:** decision
- **Chose:** the inline clause `type T struct implements X, Y { … }` (between `struct` and the body
  `{`, comma-separated interface list), replacing the standalone top-level `implements X for T`
  declaration entirely. Lowering strips the clause and emits one declaration per interface right
  after the struct's closing brace: a `var _ X = T{}` / `var _ X = (*T)(nil)` assertion for an
  ordinary interface (feature 07), a `func (T) isX() {}` marker for a sealed one (feature 01). A
  single clause may mix both. This consolidates all `implements` handling into the implements pass;
  the enums pass no longer touches `implements` (it still emits the sealed interface declaration).
- **Over:** keeping the standalone `implements X for T` form (the one pinned in feature 01), or
  supporting both forms in parallel.
- **Why:** the contract reads better attached to the type (the §3.4 note that the syntax "could
  equally be an annotation on the type" is now realized), and a comma list expresses multiple
  interfaces in one place. Full replacement (not dual-support) keeps a single spelling. The comma
  list can mix sealed + ordinary in one clause, which made the old "partition `implements` across two
  passes by sealedness" impossible to keep — hence the consolidation into one pass (the enums pass's
  `genMarker` is reused, not duplicated). Scope is **structs only for now**; extending the clause to
  any concrete type as Go allows (`type Celsius float64 implements Stringer`) is noted as future
  work. Verified: root `go test ./...` green (pipeline transpiles + compiles `kitchen_sink`), both
  feature-01 and feature-07 reference transpiler suites green, examples/expected regenerated, no
  `implements … for` remains in any `.goal` or surface doc.

### Checker (`internal/check/implements.go`) — interface-satisfaction defer-boundary
- **Kind:** decision
- **Chose:** the implements check fires when a `type T struct implements I {…}` clause names an
  **in-file, non-sealed** interface `I`. For every method `I` declares (folding in any in-file
  embedded interface's methods), it looks up `T`'s declared methods (`analyze.Tables.Methods[T]`)
  and emits an Error — `unimplemented-method` when the method name is absent, or
  `method-signature-mismatch` when the name is present but the normalized signature differs. The
  error is **located at the `implements` clause** (the `implements` token's offset), mirroring
  goal's declaration-site contract. The clause locator (the `type T struct … implements … {` scan
  and the comma-split of the interface list) is lifted verbatim from `internal/pass/implements.go`.
- **Over:** locating the error at the missing method's would-be site (there is none) or at the call
  site (a distant, structural-satisfaction location goal exists to replace); over chasing
  out-of-package interface method sets (not lexically readable).
- **Why:** the clause is exactly where the author asserted the contract, so it is where an unmet
  contract should be reported — that *is* the feature (convert "satisfied invisibly / a wrong
  signature surfaces far away" into a located declaration-site error, §3.4/§8.5). Both value- and
  pointer-receiver methods of `T` contribute to the obligation's method set (a `var _ I = (*T)(nil)`
  assertion sees `*T`'s full set), so the check keys methods by receiver type stripped of `*`.

### Defer-boundary: qualified / out-of-file / out-of-file-embedded interface → Warning; sealed → trivially met
- **Kind:** decision
- **Chose:** four non-Error outcomes. (a) A **sealed** interface (`analyze.Tables.Sealed[I]`,
  feature 01) is satisfied by the unexported marker method the implements pass synthesizes — it is
  trivially met and **skipped silently**, never flagged. (b) A **qualified** interface (`io.Writer`)
  is from another package — its method set is unreadable in-file, so emit a located **Warning**
  (`unresolved-interface`, "interface-satisfaction deferred"). (c) An interface **not declared in
  this file** is likewise deferred with that Warning. (d) An interface that **embeds** a qualified or
  out-of-file interface makes the full obligation unknown — deferred (folding in a partial set could
  miss a method and yield a false "satisfied", so the whole obligation defers).
- **Over:** assuming an out-of-package interface is satisfied (a false guarantee), or flagging a
  sealed interface as missing its (synthesized, source-invisible) marker method (a false Error on
  every feature-01 enum variant).
- **Why:** "defer, never guess" (checker contract) — a false "implements" on an unresolvable
  interface is worse than an honest deferral. The qualified/external/embedded boundaries are exactly
  the lexical ceiling: without the imported package's source, the method set cannot be read, and the
  slot doc names this as the defer-boundary. Sealed-skip mirrors how the lowering pass treats sealed
  vs ordinary interfaces in one clause.

### `analyze.Tables` extension — method index (`Interfaces`, `EmbeddedIfaces`, `Methods`)
- **Kind:** decision
- **Chose:** extend `analyze.Tables` with three name-keyed, read-only tables built once in `Build`:
  `Interfaces` (in-file interface name → its declared methods, each a `Method{Name, Sig, Raw}`),
  `EmbeddedIfaces` (interface name → embedded interface names), and `Methods` (concrete type name →
  its declared methods). A `Method.Sig` is a **normalized signature** — the parameter and result
  *type* sequences with parameter names and whitespace stripped (`(p []byte) (int, error)` →
  `[]byte|int,error`), so an interface obligation and a concrete method compare by structural
  equality; `Raw` keeps the original text for the diagnostic message.
- **Over:** re-scanning interfaces and methods inside the check (duplicating analysis the slot doc
  said to put in the tables), or comparing raw signature text (whitespace/param-name differences
  would yield false mismatches).
- **Why:** the slot doc explicitly licenses (and predicts) a method index on `analyze.Tables` for
  this guarantee — it is a fact the existing tables didn't carry. Building it once, keyed by name,
  matches the package's discipline (survives re-lexing, read-only to checks). The `interface`
  branch of `analyzeTypeDecls` already located the interface body, so populating `Interfaces` there
  is minimal; a new `analyzeMethods` scan mirrors the implements pass's `scanPointerReceivers`
  receiver-walk to index concrete methods.

### File-layout / `Code` scheme + signature-equality limitation
- **Kind:** assumption
- **Chose:** `Feature` = `"07-implements"`; codes `unimplemented-method` and
  `method-signature-mismatch` (Errors), `unresolved-interface` (deferral Warning). Signature
  equality is **textual after normalization** (drop param names, collapse whitespace) — it does
  **not** resolve type aliases or otherwise-spelled-but-equal types. So the check only asserts a
  mismatch when a method of the **same name** has a **different normalized signature** (the common,
  lexically-decidable case); it never tries to prove two differently-spelled types unequal beyond
  that normalization. A genuinely-equal-but-differently-spelled signature (via an alias) could in
  principle surface as a false mismatch — the slot doc's named alias/embedding ambiguity — but the
  in-file cases this check fires on (both sides spelled against the same file's types) do not hit it;
  the unreadable cross-package cases are already deferred. Testdata uses data-less / primitive
  method signatures so no struct literal trips the 08-fields check under the shared harness.
- **Over:** no naming scheme was fixed by the spec; full type-aware signature comparison needs the
  later `go/types` workstream and is out of scope for this lexical iteration.
- **Why:** stable greppable codes per the slot doc; normalization handles the real friction
  (parameter-name and spacing differences between an interface decl and a method decl) without a type
  system, and the residual alias ambiguity is the documented lexical ceiling — deferred, not guessed.

---

## Build Model — Phase A (BUILD-MODEL-TODO)

The build model turns goal from a single-file transpiler into one that builds a multi-file
package. Phase A is plumbing around an unchanged front-end; the thesis and the two proving
spikes are in `BUILD-MODEL-TODO.md`. Decisions accrue here per unit.

### U1 — package model & discovery (`internal/project`)
- **Kind:** decision
- **Chose:** a new `internal/project` package with `File{Path,Name,Src}` and
  `Package{Dir,Name,Files}`, plus `Discover(root)` that walks recursively (the `./...` sense),
  groups `.goal` files by directory, and reads each directory's shared `package` clause. One
  directory = one package (Go's rule); files sorted by path for determinism; source read once at
  discovery so later units need not touch the disk.
- **Why:** the rest of Phase A (U2 table merge, U4 package transpile) needs a stable, offset-free
  unit to operate over. Modeling a package as a directory of files matches Go and the eventual
  `go build` target, and keeps discovery name-oriented like the rest of the front-end.

### U1 — one-package-per-directory enforced; reserved dirs skipped
- **Kind:** decision
- **Chose:** `Discover` errors when two files in a directory declare different `package` names, or
  when a file omits the clause — the same constraint `go build` enforces. `PackageClause` lexes
  (via `scan.Lex`) rather than regexping, so a `package` word in a string/comment is never the
  clause. Discovery prunes `testdata`, hidden (`.`-prefixed), and Go-convention `_`-prefixed
  directories (e.g. `features/_cut`).
- **Why:** surfacing the conflict at discovery is a located, early error instead of a confusing
  Go-compiler redeclaration later; lexing reuses the project's no-second-parser discipline; the
  skip set mirrors Go's non-buildable directory conventions so discovery doesn't sweep in fixtures.

### U1 — single-package goal imports deferred
- **Kind:** assumption
- **Chose:** Phase A v1 models discovery and grouping but not **cross-package goal imports** (one
  goal package importing another goal package). `Discover` finds and groups all packages; wiring up
  inter-package symbol resolution is a later unit, explicitly out of Phase A v1 (per
  BUILD-MODEL-TODO open decisions).
- **Over:** could have modeled an import graph now, but the common case is single-package multi-file
  and the cross-package resolution rules (visibility, import paths) deserve their own unit rather
  than being guessed here.

### U2 — cross-file table merge (`analyze.BuildPackage` / `Tables.Merge`)
- **Kind:** decision
- **Chose:** `BuildPackage([]string)` analyzes each file with the existing `Build` and unions the
  per-file `Tables` via `Tables.Merge`, which `maps.Copy`s every name-keyed map. Because the tables
  are position-free and name-keyed, the union is sufficient for a pass over one file to resolve
  symbols declared in a sibling file (proven by SPIKE-2 and `TestBuildPackageResolvesCrossFileEnum`).
  Refactored the map init out of `Build` into a shared `newTables()` constructor.
- **Why:** the union is the entire cross-file resolution mechanism the thesis predicted — no offset
  bookkeeping, no second analysis path. `maps.Copy` keeps it terse and sidesteps the map-loop lint.

### U2 — collision rule: last-merged-wins, Go compiler backstops dup-decls
- **Kind:** decision
- **Chose:** on a name present in two files, the later source wins (`maps.Copy` overwrite).
  `BuildPackage` processes sources in caller order, and `project.Discover` sorts files by path, so
  the outcome is deterministic. `Merge` does **not** detect or report a genuine duplicate
  declaration (same func/type name in two files) — that is a Go redeclaration the Go compiler
  reports at `go build`; analyze keeps the last definition so lowering can proceed and the real
  error surfaces downstream.
- **Why:** re-implementing Go's dup-decl detection here would duplicate the compiler and risk
  diverging from it; deferring to the Go toolchain matches the build model's lean-on-Go thesis. A
  pre-emptive located "same name in two files" diagnostic is a possible later refinement (noted in
  BUILD-MODEL-TODO open decisions), not needed for correctness now.

### U3 — shared prelude relocation (suppressible inline injection)
- **Kind:** decision
- **Chose:** the closed-E pass still injects `ResultPreamble` (now exported) inline by default, but
  skips it when `analyze.Tables.SuppressResultPrelude` is set — the construction/`match`/`?` rewrites
  always run regardless. Added `pass.NeedsResultPrelude(t)` so a driver can decide whether the
  package needs the prelude at all. Single-file `Transpile` never sets the flag, so its output is
  byte-identical (the full regression suite passes unchanged); the U4 package driver will set the
  flag and emit one `goal_prelude.go` per package instead of one preamble per file.
- **Why:** suppression at the existing injection site is the minimal output-preserving change — it
  avoids moving injection into the driver (which would shift the prelude past later-injected imports
  like assert's `fmt` and churn the golden files). The flag threads through the unchanged
  `Run(src, t)` pass signature instead of widening it.

### U3 — the suppression flag lives on `analyze.Tables`
- **Kind:** assumption
- **Chose:** `SuppressResultPrelude` is a field on `Tables`, explicitly documented as the one field
  that is a driver directive rather than a name-keyed source fact.
- **Over:** alternatives were a second `ResultClosedNoPrelude` pass entry (would fork the
  `pipeline.Passes` list for package mode) or a package-level variable (hidden global state). The
  flag is the least-surprising way to thread a per-build directive through the fixed pass signature;
  the slight grain-violation is called out in the field doc so it isn't mistaken for source analysis.

### U4 — in-memory package transpile driver (`pipeline.TranspilePackage`)
- **Kind:** decision
- **Chose:** `TranspilePackage(*project.Package) (PackageOutput, error)` returns the Go **in memory**
  — `GoFile{Name,Go}` per source, one synthesized `goal_prelude.go` when the package uses closed-E
  Result, and doctest `_test.go` sidecars — and does **no disk I/O**. It builds merged tables once
  (U2), sets `SuppressResultPrelude` (U3), and lowers each file with a shared `transpileWith(src,
  tables)` core factored out of `Transpile`. Names map `foo.goal -> foo.go` / `foo_test.go`.
- **Why:** keeping the driver pure (in-memory) is what the resolved output-layout decision needs —
  the build path (U6) compiles from a temp dir by default and `--emit` persists the same bytes, so
  both modes share one transpile with no I/O policy baked in. A real `go build` in the test (not just
  a golden compare) is the actual proof the cross-file lowering + single prelude cohere.

### U4 — output layout resolved: in-memory default, `--emit` to persist
- **Kind:** decision (supersedes the BUILD-MODEL-TODO "Output layout" open question)
- **Chose:** `goal build` compiles in-memory by default (write U4 output to a temp dir, `go build`,
  discard); a `--emit[=dir]` flag persists the generated `.go` (sibling to the `.goal`, gitignored)
  for tooling/inspection. (Pivoted from the earlier "sibling files always" lean.)
- **Why:** for a personal-use tool the common path should leave the repo clean — no generated twins
  committed or cluttering source dirs — while the flag still covers inspection and any
  `go:generate`/IDE wiring that needs real files on disk. `goal_prelude.go` collision with a
  user file literally named `goal_prelude.goal` is a known low-risk edge, noted for U6.

### U5 — `//line` source map (per-declaration)
- **Kind:** decision
- **Chose:** `addLineDirectives(goalSrc, genGo, goalFile, genFile)` inserts a Go `//line` directive
  before every top-level decl in the generated Go: a decl whose name matches a user decl in the
  goal source anchors to that source line (names survive lowering, so the match is by name); a
  synthesized decl (enum encoding, `var _` assertion, injected import) re-anchors to the generated
  file at its own physical line. Granularity is **per-declaration** (chosen over per-statement),
  wired into `TranspilePackage` per user file; the prelude carries no directives.
- **Why:** SPIKE-1 proved the Go compiler honors `//line` and gofmt preserves it, so the source map
  is directive emission, not a bespoke offset structure. Per-declaration needs no per-pass
  Replacement journal and no surviving-gofmt bookkeeping; it maps the realistic failure (a Go type
  error in a passed-through function body) to the right `.goal` line — proven by a planted-error
  `go build` landing on `shapes.goal:8`. Re-anchoring synthesized decls keeps their numbering
  truthful instead of inheriting the previous mapped decl's goal line.
- **Limitation (noted):** exact line within a body whose statements were themselves lowered (a
  `match`/`?` expansion) may drift from the source line; the per-statement precision that would fix
  it needs the pass Replacement journals and is deliberately deferred (BUILD-MODEL-TODO U5).

### U6 — `goal` umbrella CLI (build / run / check, ephemeral by default)
- **Kind:** decision
- **Chose:** a new `cmd/goal` command with `build`, `run`, `check` subcommands and a `--emit[=dir]`
  flag; `goalc` stays the single-file primitive. `build`/`run` default to **ephemeral**: the
  in-memory `TranspilePackage` output is materialized to a temp dir and mapped into the module via
  `go build -overlay`, so nothing is written to the source tree and module/stdlib imports still
  resolve. `--emit` instead writes the generated `.go` beside each `.goal` (or mirrored under dir)
  for tooling/inspection. `run` requires exactly one `package main`. Toolchain output is relayed
  verbatim, so errors arrive already `.goal`-mapped by the U5 `//line` directives.
- **Why:** the overlay is the clean way to "compile in-memory natively" — it keeps the repo
  untouched on the common path while letting the real module's imports/deps resolve, which an
  isolated temp module would break. Tests prove the round trip: `goal run` prints the program's
  output, a planted type error maps to `bad.goal:4`, and the default build leaves no `.go` behind.

### U6 — `goal check` is per-file pending U7
- **Kind:** assumption
- **Chose:** `goal check` currently runs the existing single-file `check.Analyze` over every
  discovered file. It does not yet use merged tables, so cross-file references are not resolved by
  the checker — that is U7's job. Wiring `check` to package-level tables is deferred to U7.
- **Over:** could have blocked `check` until U7, but a per-file check is useful now and the upgrade
  is localized (swap the per-file analyze for a package-tables analyze).

### U7 — cross-file checker (`check.AnalyzePackage`)
- **Kind:** decision
- **Chose:** `AnalyzePackage(srcs []string) ([][]Diagnostic, error)` builds merged tables once
  (`analyze.BuildPackage`) and runs the existing `Run(src, t)` over each file against them, returning
  per-file diagnostics aligned with input order. `goal check` now uses it, so the checker resolves
  cross-file symbols. Closes the 02/06/08 *out-of-file* deferrals at the lexical level (the
  type-information-dependent residue remains Phase B / `go/types`).
- **Why:** the checks are already source-anchored and read facts from tables by name, so package-mode
  is exactly "same checks, merged tables" — each file's constructs are checked once, the union only
  adds resolution. Proven by a non-exhaustive match whose enum lives in a sibling file: deferred
  (no Error) under single-file `Analyze`, caught as `non-exhaustive-match` under `AnalyzePackage`.

---

## Depth checks — Phase B (DEPTH-TODO)

The depth checker is a second stage that runs on the *lowered* Go (Phase A's
TranspilePackage output) and answers the type-information-dependent deferrals via stdlib
go/types. Thesis + proven SPIKE-B1 are in `DEPTH-TODO.md`.

### B1 — go/types harness (`internal/typecheck`)
- **Kind:** decision
- **Chose:** `typecheck.Load(*project.Package) (*Package, error)` transpiles the package
  (pipeline.TranspilePackage), parses the lowered Go, and type-checks it with stdlib go/types
  under an error-collecting `Config{Error: collect}`. `Package` exposes `{Fset, Types, Info, Files,
  Tables, Errors}` plus `GoalPos(node)`/`Lookup(name)`. Load errors only on a transpile/parse
  failure (a goal-compiler bug); Go type errors are collected into `Errors`, not fatal. Importer is
  `importer.Default()` (verified to resolve stdlib `fmt`). Stdlib-only — no x/tools.
- **Why:** every B2–B6 check needs the same typed view + the merged goal tables (to know which
  question to ask about which symbol) + a goal position. Error tolerance keeps partial type info
  available for a buggy program, and the collected errors are themselves goal-located via the U5
  //line directives (test: a type error maps to `bad.goal:4`). Parsing with SkipObjectResolution
  since go/types does its own resolution.

### B1 — depth checker is a separate stage beside the lexical checker
- **Kind:** decision
- **Chose:** `internal/typecheck` is distinct from `internal/check`. The lexical checker runs on the
  original source pre-lowering (no parser, name-keyed); the depth checker runs on the lowered Go
  post-transpile (go/types). `goal check` will run both and merge diagnostics (B-units wire this in).
- **Why:** the two operate on different artifacts with different machinery; conflating them would
  force a parser into the lexical stage or re-lower inside it. Keeping them separate preserves the
  front-end's no-new-parser discipline while letting the depth stage use the full Go type system.

### B2 — implements via real type identity (`typecheck.CheckImplements`)
- **Kind:** decision
- **Chose:** the depth 07 check locates each `type T struct implements I` clause in the goal source
  (reusing the lexical locator) and verifies it with `types.MissingMethod(*T, I)` against the
  type-checked package, reporting at the clause position. The interface is resolved through go/types:
  an in-package name via package scope, a qualified name (`io.Writer`) via the package's imports.
  Checks the pointer type's method set (the superset matching goal's `var _ I = (*T)(nil)` form).
- **Why:** real type identity removes both documented §07 lexical limits — an alias-equal-but-
  differently-spelled signature is no longer a false mismatch (test: `Get(id int)` vs `Get(id ID)`
  with `type ID = int` is accepted), and a qualified/out-of-package interface is *checked* rather
  than deferred (test: `io.Writer` satisfied → clean, missing `Write` → located error). This is the
  concrete payoff of the Phase B thesis: transpile to Go, ask go/types.
- **Note:** the depth and lexical 07 checks can both flag the same clause; dedup (prefer the
  type-backed verdict) is wired when `goal check` runs both stages (a later B-unit / integration).

### B3 — must-use stored-then-dropped (`typecheck.CheckMustUse`), the §03-refused class
- **Kind:** decision
- **Chose:** lift the §03 "go/types graduation boundary" refusal (assigned/stored Results never read)
  by covering the **two genuinely-deferred flow subsets that Go itself does not catch** and types can
  resolve. The simple bound-then-unused local needs nothing — once a Result/Option lowers to a Go
  local, Go's own "declared and not used" already rejects it (verified: `o := find(x)` unused →
  `declared and not used: o`). So B3 targets:
  1. **`discarded-result-error` (Error):** `v, _ := f()` / `_, _ = f()` where `f` is an open-E Result
     function (`Tables.FuncSignatures[f].Mode == ModeResult`, lowered to a `(T, error)` tuple) and the
     error (last) LHS position is the blank `_`. This is the canonical unchecked-error footgun: legal
     Go, but the must-use violation goal exists to prevent. Located at the discarding `_`.
  2. **`dropped-stored-result` (Error) / `unresolved-dropped-field` (Warning):** a Result/Option-typed
     struct field never read via any selector in the package. "Consulted" = the field's `*types.Var`
     appears as the `Obj()` of some `Info.Selections` entry (a composite-literal *store* is not a
     selection, so storing into a field does not count as using it). An **unexported** never-read
     field is package-private, so provably dropped → Error; an **exported** never-read-in-package
     field may be read by another package → honest deferral Warning, never an Error.
- **Scope confirmed with user** (AskUserQuestion): cover **both** subsets (vs. either alone).
- **Why:** these are exactly the cases CHECKER-TODO/§03 flagged as needing real type/flow info. Each
  is grounded in "tables locate, go/types decides": the goal tables name the Result-mode functions and
  the Option/Result fields; go/types decides the blank-error-position and the never-selected-field
  flow facts. A false "consumed" is worse than an honest deferral, so anything types cannot resolve is
  skipped or warned, never errored. 9 tests (positives per kind incl. closed-E Result field, the
  value-discard/error-kept and plain-`(int,error)` negatives that pin false positives, the read-field
  clean case, and the exported-field deferral).
- **Assumption — Result fields read from go/types, not the (buggy) struct table.** `analyze`'s
  `parseStructBody` splits a field line on `strings.Fields`, so a multi-arg `result Result[int, DBErr]`
  line is mis-split into garbage `Field`s (the embedded comma). That is a front-end limitation outside
  this unit's scope (`internal/analyze`). B3 sidesteps it: it iterates the **real go/types fields** of
  each goal-declared struct and recognizes a Result field from its resolved type (the injected generic
  `Result` named type), consulting `Tables.Structs` only for the type-ambiguous Option case (`*T`),
  whose single-argument line the table parses correctly. The user may prefer fixing
  `parseStructBody` to be bracket-depth aware instead — vetoable.
- **Defers (recorded, not faked):** a `v, _ :=` whose callee is a selector/method (mode not
  table-resolvable) is skipped silently (like the §02 boundary); an **open-E** `Result[T, error]`
  *field* has no single-value lowering (its type stays unresolved) and is skipped via an invalid-type
  guard; a field stored then written-back via a selector (`b.f = …`) counts that selector as a touch
  and is conservatively not flagged; a "bound and passed to a callee that ignores it" drop needs
  interprocedural analysis and is left to a later unit. An unexported must-use field consulted only via
  reflection/serialization would be a false Error — judged rare under goal's no-magic philosophy;
  vetoable.
- **No harness change, no CLI change.** `CheckMustUse(*Package) []Diagnostic` follows the B2 pattern;
  the depth stage is still not wired into `goal check` (no caller of `typecheck` outside the package,
  same as after B2) — wiring stays a later integration unit.

### B4 — conversion recursion is BLOCKED as a checker-only unit (refusal-with-reason)
- **Kind:** refusal (with reason)
- **Refused:** implementing B4 (12 conversion-recursion depth check) within the depth-checker loop's
  guardrails (touch only `internal/typecheck`; never edit the front-end passes / build model).
- **Why — the depth checker runs on the *lowered* Go, but the derive pass refuses to lower exactly the
  classes B4 must check, so those programs never transpile and the depth stage never sees them.**
  Verified empirically by transpiling derive programs through `pipeline.TranspilePackage`:
  - An **out-of-package** target/source struct → transpile error: `pass.genConversion` reads
    `t.Structs[type]`, which holds only in-package (package-merged) structs, and errors
    ("unknown target struct") on an imported type.
  - **map / `Option[A]`→`Option[B]` / pointer-differing / nested-without-`from func`** recursion →
    transpile error: `pass.resolveField` implements only same-text-type, a registered `from func`, and
    `[]A→[]B` (with same-or-registered element); anything else is "no conversion ... in scope".
  - Everything that **does** transpile is already fully and correctly decided by the lexical
    `checkConvert` (the depth check would be a port of the same same-type/registry/slice logic over the
    same merged `Tables.Structs` — go/types adds nothing). Feature 12 also has **no checker tests and
    no derive testdata** in the repo, consistent with these paths being unexercised end-to-end.
  - So B4's value (out-of-package types; map/Option/pointer/nested recursion; identity- not text-based
    matching) requires **first extending the derive *pass*** to lower those classes (and to use
    go/types identity) — front-end/build-model work the loop's guardrails forbid, and the same
    cross-cutting lowering nature the queue already flagged for B5. Recorded, not faked: a vacuous
    depth check would be a false signal of progress.
- **User decision (AskUserQuestion):** "Reassess the queue" — do not force B4; report the dependency
  analysis below and re-plan.

### B4 — CLOSED by reassessment after the front-end lowered the recursion (2026-06-21)
- **Kind:** refusal-with-reason (no depth check written) + decision (where the residual value lives)
- **Context:** the front-end workstream (LOWERING-TODO) lifted the guardrail and **L1/L2** extended
  `pass.resolveField` to lower in-package map/pointer/array/Option-as-pointer and nested-struct
  conversion recursion — exactly the classes B4's BLOCKED note said had to be "lowered first." User
  then pivoted to "close B4." On investigation it does not close as a *depth check*.
- **Finding — a B4 depth check is vacuous as an error-producer, so none was written.** The derive pass
  lowers a conversion **only when every leaf resolves** (same-type / registry / sub-recursion);
  therefore a conversion that *lowers* is **type-sound by construction** and `go/types` on the
  generated function has nothing to flag (verified: the L1 `derive_container_recursion` and L2
  `derive_nested_struct` outputs both type-check clean). A genuinely *incomplete* conversion **does not
  lower** — the pass returns an error, `goal build`/`TranspilePackage` fails, and the depth stage's
  `Load` never sees it. So there is no program that (a) lowers, (b) the lexical `checkConvert` accepts,
  and (c) `go/types` would reject — i.e. nothing for a depth check to catch. Writing one would be the
  "vacuous depth check = false signal of progress" the original BLOCKED note explicitly warned against.
- **Where feature 12's residual value actually lives — the FRONT-END, not a depth check:**
  - **alias/assignable-identity:** `resolveField` (and lexical `checkConvert`) match field types
    **textually**, so a `type Name = string` field sourced from a `string` is falsely treated as
    unbridged — even though the two are identical and directly assignable. Fixing this is a *pass*
    change (treat assignable-identical types as same-type) plus an `analyze` alias table; it broadens
    what *lowers*. It is **still not a depth check** — once the pass assigns them directly, the Go
    type-checks, so `go/types` again has nothing to add.
  - **out-of-package** target/source structs: `genConversion` reads in-package `t.Structs`; lowering
    these needs the type-feedback architecture (LOWERING-TODO **L5**, Option B). The "identity not
    textual" matching B4 envisioned would live there.
- **Outcome:** B4 marked CLOSED in DEPTH-TODO — its substance (lower the recursion) was delivered by
  L1/L2; its depth-check framing was a misconception (the original BLOCKED note half-anticipated this:
  "go/types adds nothing" for what transpiles). No `internal/typecheck` code added. User decision
  (AskUserQuestion): "Record + move on."

### Phase B queue reassessment (2026-06-20, after B1–B3)
- **Kind:** assumption (planning note; vetoable)
- **State:** B1 (harness), B2 (07 implements), B3 (03 must-use) are **done** — the units whose deferred
  classes survive transpilation and are decidable from the lowered Go. The remaining units are gated:
  - **B4 (12 conversion recursion):** BLOCKED on front-end lowering (see refusal above). Not
    checker-viable until the derive pass lowers out-of-package + map/Option/pointer/nested recursion.
  - **B5 (value-position `x := match`):** already flagged in the queue as a **lowering** unit
    (`internal/pass`/`pipeline`), "not a pure checker unit." Out of the depth-checker loop's scope by
    the same guardrail.
  - **B6 (promote residual 02/06/08 deferrals):** its stated "depends on B1–B4" is **conservative
    sequencing, not a real dependency** — B6 covers features 02/06/08, independent of B4's feature 12.
    Probed and found checker-viable: e.g. an **inferred/nested struct literal of an in-file goal
    struct** (`Outer{inner: {a: 1}}` omitting required `b`) transpiles cleanly **and the lexical 08
    check silently misses it** (it cannot type the bare `{…}`), while go/types resolves the literal's
    type — a genuine, loadable, type-backed Error the depth stage can add. (By contrast, cross-*file*
    02 exhaustiveness is already caught via merged tables; only truly cross-*package* 02/06 cases
    remain, which are semantically limited — unexported sealed markers aren't enumerable across a
    package boundary, and imported Go structs don't carry goal's no-zero-value contract.)
- **Recommendation:** **resequence B6 ahead of B4** as the next depth-checker unit (it has real,
  loadable, checker-only wins and no true B4 dependency). Treat **B4 and B5 as a separate front-end /
  lowering workstream** to be authorized explicitly (they extend the deriver and the match lowering),
  after which B4's depth check becomes meaningful. This keeps the depth-checker loop honest: it
  advances on what the lowered Go can actually express, and does not fake units gated on the front end.

### B6 — promote residual 08 deferral: elided composite literals (type-backed)
- **Kind:** decision
- **Did:** added `typecheck.CheckNoZeroValue` (`internal/typecheck/nozero.go`) — the depth version of
  feature 08 (no-zero-value) for **elided composite literals**: an element/value literal that omits its
  type because Go infers it from the surrounding array, slice, or map type
  (`[]Inner{{a: 1}}`, `map[string]Inner{"k": {a: 1}}`, `[N]Inner{{a: 1}}`). Such a literal is valid Go
  that silently zero-fills any omitted field — the exact footgun feature 08 closes — but its
  required-field set is invisible to the lexical scan. The check walks AST `*ast.CompositeLit` nodes with
  no type expression (`Type == nil`), resolves the inferred type via `Info.Types`, and — when it is a
  **named struct declared in this goal package** — reports each omitted field as a located Error
  (`Code: "elided-missing-field"`, at the literal's `{`, goal-mapped via `//line`). 8 tests
  (slice/map/array/empty positives, complete + typed-at-site + non-struct negatives, unresolved deferral).
- **Correction to the queue-reassessment probe (2026-06-20).** The reassessment cited
  `Outer{inner: {a: 1}}` (struct-field-value elision) as the win. **That example is wrong:** Go does
  **not** permit eliding the type of a *struct field value* (only of array/slice/map elements and map
  keys), so that program lowers to **invalid Go** — `go/types` reports "missing type in composite
  literal" and the literal's type is unresolved, not type-backed. (If goal intends to *accept* that
  surface, the deriver/transpiler must insert the field type — a front-end/lowering gap, out of this
  loop's scope.) The genuine, in-scope, type-backed win is the **valid** elision positions above. There
  the lexical stage does not "silently miss" but actively **misfires**: it latches onto the surrounding
  `Inner{` of `[]Inner{{…}}`, cannot see into the nested element, and reports the **wrong** field set
  (every field "missing", even ones the element supplies). The depth check returns the field-accurate set.
- **Defer-boundary (what types decide vs. what is punted):**
  - **In scope:** `Type == nil` literals resolving to an in-package named struct (`named.Obj().Pkg() ==
    p.Types` **and** the name is in `Tables.Structs`). The package-identity guard keeps feature 08's
    guarantee off **imported Go structs** (which carry no such contract) and off **injected helper types**
    (e.g. the generated `Result`/`Option` sum types are not in `Tables.Structs`).
  - **Positional element literals** (`Inner{1, 2}` style, non-keyed) are skipped: Go itself requires
    every field of a positional struct literal, so an incomplete one is already a Go error — no goal gap.
  - **Unresolved** elided literals (`Info.Types` type nil/invalid) are skipped silently, not warned: they
    are already a collected Go error (e.g. the invalid struct-field elision above), and a second
    feature-08 diagnostic on the same construct would be noise.
  - **Generic-instantiated literals** (`Box[int]{…}`) were initially deferred, then **delivered as a B6
    follow-up** — see the next entry. Qualified out-of-package literals (`pkg.User{…}`) stay deferred by
    design — not goal's guarantee. `...defaults` *inside* an elided/generic literal depends on whether the
    defaults pass expands it there (lowering-dependent); untested, deferred.
- **Assumption — `Code: "elided-missing-field"` (distinct from the lexical stage's `missing-field`).**
  A separate code makes the type-backed promotion greppable and lets the CLI merge apply the DEPTH-TODO
  dedup decision ("prefer the type-backed one") when both stages flag one construct. Vetoable — could be
  unified to `missing-field` if the merge keys on `Feature`+position instead.
- **No harness change, no CLI change.** `CheckNoZeroValue(*Package) []Diagnostic` follows the B2/B3
  pattern; the depth stage is still not wired into `goal check` (wiring remains a later integration unit,
  same as after B2/B3). `plural`/`quoteJoin` are local copies (the lexical stage's are in package `check`).
- **Residual 02/06 after B6:** unchanged from the reassessment — cross-*file* 02 exhaustiveness is
  already caught via merged tables; only cross-*package* 02/06 cases remain, which are semantically
  limited (unexported sealed markers aren't enumerable across a package boundary; imported Go structs
  carry no goal contract). Not promoted; recorded as a genuine narrow residue.

### Integration — wire the depth stage into `goal check` (both stages now run)
- **Kind:** decision
- **Did:** `goal check` (`cmd/goal/main.go`, `cmdCheck`→`checkPackage`) now runs **both** stages per
  package: the lexical stage (`check.AnalyzePackage`, original source) and the typed depth stage
  (`typecheck.Load` + `CheckImplements`/`CheckMustUse`/`CheckNoZeroValue`, lowered Go). Findings are
  merged into a stage-agnostic `checkDiag`, sorted by file/line/col, rendered uniformly
  (`file:line:col: severity: [code] message` — both stages already shared that shape), and Errors from
  either stage drive the exit code. Closes the DEPTH-TODO "Done when … `goal check` runs both stages"
  criterion. Until now the entire depth track (B2/B3/B6) was tested but never executed for a user.
- **Dedup decision (resolved here; was a B1 open decision): prefer the type-backed finding.** When both
  stages flag the same construct — same file *basename*, line, and `Feature` — the lexical finding is
  dropped and the depth one kept. This matters most for feature 08: on an elided element literal
  (`[]Inner{{a: 1}}`) the lexical scan **misfires** (latches onto the surrounding `Inner{`, reports the
  wrong field set), while the depth check reports the field-accurate set; suppressing the lexical misfire
  is strictly correct. For 07, both may flag a genuinely-unimplemented interface on the same clause line;
  the depth verdict (real `types.Implements`) supersedes the lexical text comparison. Keyed on the path
  **basename** because the two stages spell paths differently — lexical uses the discovered `File.Path`,
  depth positions come via `//line` (basename) or `goalPosition` (full path), inconsistently even across
  the depth checks; basenames are unique within a package, so the key is sound. `depthFilePath` maps a
  depth finding's basename back to the full `File.Path` so output paths are consistent.
- **Assumption — line+feature granularity for dedup (vetoable).** Two *different* constructs of the same
  feature on one line would over-suppress (the lexical one is dropped). Judged rare (a line rarely holds
  two literals of the same struct-completeness violation) and the safe direction (prefer the type-backed
  verdict). A position-exact key would need the lexical and depth offsets reconciled, which the
  stage-inconsistent filenames/offsets don't currently support.
- **Refusal-with-reason — do NOT surface raw `go/types` errors (`Package.Errors`) in `goal check` yet.**
  The harness collects Go type errors error-tolerantly, and they are goal-mapped, so surfacing them is
  tempting. But `typecheck.Load` uses `importer.Default()` (gc export data), which resolves stdlib but
  can **fail to import third-party modules**, producing *false* "could not import" errors — a false
  guarantee, worse than silence. The three depth checks degrade gracefully (they defer when types don't
  resolve, so a broken import yields no false Error), but raw `Package.Errors` would not. So `goal check`
  surfaces only the lexical + depth-check findings; **`goal build` remains the gate that surfaces real Go
  type errors** (mapped to `.goal`, already tested). Revisit when the importer decision (DEPTH-TODO open
  decisions: `Default()` vs `ForCompiler(…, "source", …)`) is made.
- **Depth-stage load failure is non-fatal to `check`.** If `typecheck.Load` fails (the program does not
  transpile), `checkPackage` prints a `depth stage unavailable for <dir>: <err>` note and returns the
  lexical findings; it does not fail `check` solely on that. A non-transpiling program is a `goal build`
  hard-failure, not a guarantee violation — `check` reports guarantees and stays usable on partial input.
- **Cost:** the typed stage transpiles + type-checks each package, heavier than lexing. Consistent with
  the DEPTH-TODO "lean: `check` only" decision — `build`/`run` are unchanged and do not run the depth
  stage. CLI tests: depth catches the elided literal + dedup suppresses the lexical misfire; clean
  program still prints `ok`.

### B6 follow-up — promote generic-instantiated struct literals (`Box[int]{…}`)
- **Kind:** decision
- **Did:** extended `CheckNoZeroValue` to also flag **generic instantiation** literals — `Box[int]{val: 1}`
  omitting `tag`. The lexical scan keys on `IDENT {`, but a `]` sits between the type name and the brace,
  so it never matches; the analyze tables don't register generic structs either (confirmed:
  `Tables.Structs` is empty for `type Box[T any] struct`). go/types resolves the instantiated `Box[int]`
  and reports the field-accurate omission (`Code: "generic-missing-field"`, message spells the
  instantiation via `types.TypeString`). The literal classifier `litClassOf` routes by AST type
  expression: `nil` → elided, `*ast.IndexExpr`/`*ast.IndexListExpr` → generic; plain `*ast.Ident`
  (lexical stage's job) and qualified `*ast.SelectorExpr` (out of package) are skipped.
- **Decision — replace B6's `Tables.Structs` membership guard with a declaration-position guard
  (`isGoalDeclared`).** The original guard ("name is in `Tables.Structs`") cannot admit generic structs,
  because analyze doesn't track them. The new guard accepts a resolved named type iff its object is in
  this package (`Obj().Pkg() == p.Types`) **and** its declaration position maps to a `.goal` file
  (`Fset.Position(Obj().Pos()).Filename` ends in `.goal`). Verified: a user type (`Box`, `Inner`) resolves
  to a `.goal` position; injected prelude structs (`Ok`/`Err`, built by the Result lowering) are not in
  `.goal` (synthetic prelude) — and `Result` itself is an interface, excluded by the underlying-struct
  check anyway. This is behavior-preserving for the non-generic elided cases (a user struct is still
  admitted, injected types still excluded) and strictly more capable (admits generics). A new test pins
  that an injected `Ok` construction (`Result.Ok(1)`) is never flagged.
- **Scope/limits unchanged:** keyed-only (positional → Go enforces), in-package only (qualified generics
  `pkg.Box[int]{…}` excluded by `isGoalDeclared`), unresolved literals deferred. `...defaults` inside a
  generic literal is lowering-dependent and untested — the message suggests it but it is not asserted.
- **Tests:** generic positive (omits `tag`, message spells `Box[int]`), generic complete (no diagnostic),
  injected-type-not-flagged; the four elided cases still pass under the new guard. 11 nozero tests total.
- **No CLI change needed:** the lexical stage emits nothing for generic literals, so there is no dedup
  conflict — the depth finding stands alone through the already-wired `goal check`.

---

## US-003 — differential checker parity gate (sema vs legacy internal/check)

### Parity is judged by (file, line, feature, code, severity), not message text
- **Kind:** decision
- **Chose:** the differential parity gate (`internal/corpus/parity_test.go`,
  `TestSemaLegacyParity`) compares the AST `sema` checker against the legacy
  `internal/check` checker over every `testdata/check/**` case by the tuple
  (file, line, feature, code, severity), excluding the message string.
- **Over:** comparing rendered messages byte-for-byte.
- **Why:** the two front-ends word their messages differently by design (the
  lexical checker quotes raw source spans; the AST checker reports resolved
  type/field names), so message equality would be noise. Parity is about *which
  guarantee fires where, at what severity* — that is what makes deleting
  `internal/check` (US-005) safe. The gate subtracts an explicit, DECISIONS.md-
  backed allowlist of known divergences and requires the remainder to be
  identical; it also fails on a *stale* allowlist entry (a documented divergence
  that no longer reproduces), so the allowlist cannot rot silently as sema evolves.

### Divergence 1–3 (improvement): AST resolves in-file derive targets the lexical checker defers
- **Kind:** decision (accept-and-document, per US-003 note in prd.json)
- **Divergence:** for three bodyless `derive func` cases the AST checker fires a
  real Error where the legacy checker defers with a Warning
  (`12-derive-convert/unresolved-derive-type`):
  - `testdata/check/12-derive-convert/fallible_in_total.goal:24` —
    sema `fallible-in-total-derive` (Error) vs legacy deferral (Warning).
  - `testdata/check/12-derive-convert/unbridged_field.goal:19` —
    sema `unbridged-field` (Error) vs legacy deferral (Warning).
  - `testdata/check/12-derive-convert/unsourced_field.goal:18` —
    sema `unsourced-field` (Error) vs legacy deferral (Warning).
- **Why it diverges:** the legacy checker reads the derive's target type *name*
  lexically and (in these fixtures) swallows the trailing `// want "…"` comment
  into the name, so it never resolves the in-file struct and defers. The AST
  checker resolves the struct from the parsed file and proves the incompleteness,
  firing the Error the fixture was written to expect.
- **Chosen resolution:** accept the AST behavior as the correct one — this is the
  documented improvement US-003 anticipates, and the reason `internal/check` is
  being deleted rather than matched byte-for-byte. The `// want` markers already
  contain the sema Error message substrings (they reflect the sema behavior), so
  no fixture edit was required; the three entries are recorded in the gate's
  allowlist as sema-side Errors paired with legacy-side deferral Warnings.
- **Over:** weakening the AST checker back to a deferral to match legacy (would
  discard a real, provable guarantee), or treating it as a gate failure (would
  block the very deletion US-003 enables).

### Divergence 4 (extra deferral): AST surfaces a located `unresolved-err-value` Warning legacy omits
- **Kind:** decision (accept-and-document)
- **Divergence:** `testdata/check/06-error-e/defer_err_value.goal:16` — the AST
  checker emits a located `06-error-e/unresolved-err-value` Warning (the
  `Result.Err(e)` returns a bound variable whose concrete error type is not
  lexically resolvable, so closedness is deferred); the legacy checker emits
  nothing there.
- **Why accepted:** both are non-rejecting deferrals (the case has no `// want`
  marker and warnings may go unclaimed by the corpus check runner), so neither
  checker rejects the program and no fixture changes. The AST checker simply
  surfaces the deferral honestly; recorded as a sema-side Warning in the gate's
  allowlist.
- **Over:** silencing the sema deferral to match legacy's silence — the located
  deferral is the more honest behavior and aligns with the "defer, never guess"
  discipline used throughout the checker.

## US-005 — delete internal/check (legacy lexical checker)

### The US-003 parity gate is retired together with internal/check
- **Kind:** decision
- **Chose:** delete `internal/corpus/parity_test.go` (`TestSemaLegacyParity`) and
  its `knownDivergences` allowlist as part of US-005, along with
  `internal/corpus/check_runner_test.go` (which drove the now-deleted
  `check.Analyze`).
- **Why:** the parity gate exists to compare the AST `sema` checker against the
  legacy `internal/check` checker. US-005 deletes `internal/check`, so the gate
  has nothing to compare against and cannot compile — its purpose (prove the
  deletion loses no guarantee) is discharged the moment the deletion lands. The
  four divergences it documented (above) remain recorded here as the historical
  rationale. Ongoing coverage of the AST checker over the corpus is provided by
  the SemaCheck-driven runners (`sema_checker_test.go`, `sema_fields_test.go`,
  `sema_question_test.go`, `ast_gate_test.go`).
- **Over:** keeping a one-sided "gate" that only runs sema (redundant with the
  SemaCheck runners) or freezing a copy of the legacy checker just to keep the
  comparison alive (defeats the deletion).

### check.OffsetToPosition's survivors land in internal/token; severity unifies on sema
- **Kind:** decision
- **Chose:** the pure `OffsetToPosition` helper moves to `internal/token`
  (`token.OffsetToPosition`, a leaf with no imports); consumers that held a
  `check.Diagnostic` now read `sema.Diagnostic` (Line/Col carried on `Pos`);
  `typecheck.Diagnostic.Severity` and the depth checks unify on `sema.Severity`
  (`sema.Error`/`sema.Warning`). `typecheck` keeps a tiny local `offsetLineCol`
  helper because it imports `go/token` as `token` and cannot also alias
  `goal/internal/token`. The corpus checker seam drops the `Checker` interface and
  `CheckerFunc` adapter; `RunCheck` takes the checker func (`SemaCheck`) directly.
- **Why:** positions are a `token` concern, so that package is the honest home for
  the offset→line/col helper; sema is the one surviving checker, so its severity
  and diagnostic types are the canonical ones. `lsp` keeps `analyze.DirResolver`
  as its resolver type (converted to `sema.DirResolver` at the call site), so it
  remains the lone `analyze` resolver-type consumer the later stories address.

---

## self-host idiomatic audit — US-005 (token)

> Note: a separate, earlier "US-005" above belongs to the AST-rewrite PRD
> (delete internal/check). This section is US-005 of the **self-host idiomatic**
> PRD (`prd.json`): the per-package idiomatic audit of the self-hosted compiler.

### `selfhost/token`'s `Kind` stays an iota `const` block — NOT a goal `enum`
- **Kind:** refusal (with reason)
- **Refused:** rewriting the iota-based `type Kind int` + `const ( ILLEGAL Kind =
  iota; ... )` block in `selfhost/token/token.goal` into a goal `enum`.
- **Why:** a goal `enum` lowers to a **sealed interface + one struct per variant +
  an unexported marker** (see §01-enums / §8.1) — i.e. a *closed sum type whose
  values are boxed interface values*. It is deliberately **not** an ordered
  integer type. `Kind`, by contrast, depends on integer identity in three
  load-bearing ways that the enum encoding cannot provide:
  1. **Array indexing.** `kindNames [...]string` is indexed directly by `Kind`
     (`kindNames[k]`), and `init()` iterates `for k := keywordBeg + 1; k <
     keywordEnd; k++` to build the `keywords`/`operators` maps. A sealed-interface
     value is not an integer and cannot index an array nor be counted in a `for`.
  2. **Range arithmetic.** The class predicates are pure integer-range tests —
     `IsLiteral`: `literalBeg < k && k < literalEnd`; likewise `IsOperator` /
     `IsKeyword` — relying on the unexported `*_beg`/`*_end` sentinels that bracket
     each contiguous `iota` run. An enum has no ordering and no sentinel run.
  3. **Dense contiguous numbering.** The whole design mirrors `go/token.Token`: a
     compact integer space grouped by ranges. Enum variants carry no stable
     ordinal the predicates could use.
  Forcing `Kind` into an `enum` would mean replacing array lookup with a giant
  `match`, the range predicates with per-variant membership tables, and would
  change `Kind`'s representation from `int` — breaking the package's public API
  (`type Kind int`, `func Lookup(...) (Kind, bool)`) that the **US-003 verbatim
  self-host oracle tests** are run against unchanged. The goal `enum` is the right
  idiom for a *closed, unordered, possibly-payload-carrying* set; `Kind` is an
  *ordered, dense, payload-free integer enumeration*, which is exactly the case Go
  `iota` + a typed int already expresses idiomatically and which goal inherits
  verbatim. This is the AC-1 "deliberate decision not to … recorded in
  DECISIONS.md" branch.
- **Over:** a full `enum Kind { ILLEGAL; EOF; ... }` rewrite (loses indexing,
  range predicates, and the `int` representation the oracle pins); a *partial*
  enum over just one sub-range (the `*_beg`/`*_end` sentinels span the whole block,
  so no sub-range is separable without breaking the others).

### No `switch`→`match` and no Result/`?` conversions apply to `selfhost/token`
- **Kind:** assumption
- **Chose:** leave `selfhost/token/token.goal` source unchanged beyond this
  ledger entry.
- **Why:** the package contains **no `switch` statement** (only the `"switch"`
  keyword spelling string), so there is nothing to convert to `match`. It is
  **import-free and has no `(T, error)` function**, so there is no manual `if err
  != nil` propagation for `goal fix` to idiomatize — `goal fix
  selfhost/token/token.goal` produces no diff and reports nothing (AC-2 already
  holds). The one multi-value helper, `Lookup(name string) (Kind, bool)`, is the
  **comma-ok** idiom, not a fallible `(T, error)`; converting it to
  `Option[Kind]` was refused because (a) it is not a `goal fix` propagation site
  and (b) it would change the public signature the reused oracle test
  (`got, ok := Lookup(...)`) depends on, violating the behavior-preserving
  constraint. `Lookup` therefore stays comma-ok.
- **Over:** an `Option[Kind]` rewrite of `Lookup` (breaks the oracle test; not a
  propagation site).

## self-host idiomatic audit — US-006 (lexer)

> US-006 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of `selfhost/lexer/lexer.goal`. Follows the US-005 (token)
> pattern: classify each Go-ism against the goal idiom it could become, convert
> where it FITS, and record refusals-with-reason here.

### The lexer's `switch` statements stay `switch` — none are over an in-file enum
- **Kind:** refusal (with reason)
- **Refused:** rewriting any of `selfhost/lexer/lexer.goal`'s `switch`
  statements into `match`.
- **Why:** AC-1 scopes the conversion to "`switch` statements over an in-file
  enum". The lexer declares **no `enum`** (and imports none — `token.Kind` is an
  iota `type Kind int`, deliberately NOT an enum per the US-005 decision, so
  even a switch on a `Kind` would not be an enum switch). Every `switch` in the
  lexer is over a non-enum value:
  1. `Next()` — an **expression-less `switch`** whose cases are boolean
     predicates (`ch == eof`, `isLetter(ch)`, `isDigit(ch) || (ch == '.' &&
     isDigit(l.peek()))`, `ch == '"'`, ...). This is condition dispatch, not a
     closed-variant scrutinee; `match` has no role here.
  2. `scanOperator()` and its nested switches (`'+' '-' '&' '|' '<' '>' '='`,
     and the `op2` helpers) — switch over `l.ch` / `ch`, a **`rune`** (a
     primitive integer). The arms are individual character literals, an open
     not-closed domain.
  Per DECISIONS §02-match ("Switch-coexistence", §228), plain `switch` is
  **legal on non-enum types** and `match` is reserved for closed enums (a plain
  `switch` on a closed enum is the compile error). Forcing these rune/condition
  switches into `match` would be both non-idiomatic and outside the AC; the
  reference transpiler does not transform plain `switch`, and the US-003 oracle
  pins the emitted tokens byte-for-byte. This is the AC's "convert where it
  fits" → it does not fit.
- **Over:** a `match l.ch { '+' => ..., ... }` rewrite of `scanOperator`
  (rune is not a closed enum; non-idiomatic and not behavior-preserving under
  the match lowering); a `match { ch == eof => ..., ... }` rewrite of `Next()`
  (condition dispatch, no scrutinee enum).

### No Result/Option/`?` conversion applies — the lexer has no fallible helper
- **Kind:** assumption
- **Chose:** leave `selfhost/lexer/lexer.goal` source unchanged beyond this
  ledger entry.
- **Why:** the lexer is a **total tokenizer**: no function returns `error`. It
  reports lexical problems *in-band* by emitting a `token.ILLEGAL` token
  (`scanOperator`'s default) rather than via a fallible `(T, error)` signature,
  so there is **no manual `if err != nil` propagation** for `goal fix` to
  idiomatize into Result/`?`. `goal fix selfhost/lexer/lexer.goal` produces no
  diff and reports nothing — AC-2 ("goal fix reports no remaining
  auto-convertible propagation sites") already holds. The package's multi-value
  uses are not fallible-error sites: `token.Lookup(lit)` is the **comma-ok**
  idiom `(Kind, bool)` (pinned by the US-003 oracle, kept comma-ok per the
  US-005 decision); `utf8.DecodeRuneInString` returns `(rune, int)` (size, not
  an error). Neither is an Option/Result candidate. Converting any internal
  `scan*`/`peek`/`next` helper would also change behavior or signatures the
  oracle pins, violating the behavior-preserving constraint.
- **Over:** an `Option[token.Token]` / `Result` rewrite of the `scan*` family
  (they are infallible and ILLEGAL is the deliberate in-band signal); an
  `Option[Kind]` rewrite of the `Lookup` call site (not a propagation site;
  breaks the oracle test — same as the US-005 token decision).

## self-host idiomatic audit — US-007 (ast)

> US-007 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of `selfhost/ast` (ast.goal, goal_decl.goal, goal_expr.goal,
> goal_stmt.goal, walk.goal). This is the highest idiomatic-opportunity package
> (sealed interfaces + match over node kinds) and the highest risk. Follows the
> US-005 (token) / US-006 (lexer) pattern: classify each Go-ism against the goal
> idiom it could become, convert where it FITS and is behavior-preserving, and
> record refusals-with-reason here. Outcome: a recorded DECISION with **no
> `.goal` source change** — every candidate's only behavior-preserving form lies
> outside `selfhost/ast`, so converting would break the US-003 verbatim
> self-host oracle and/or ripple out of this story's scope.

### The node category interfaces stay plain Go interfaces — NOT goal `sealed interface`
- **Kind:** refusal (with reason)
- **Refused:** rewriting `Node` / `Decl` / `Stmt` / `Expr` / `Spec`
  (the `go/ast`-mirrored marker interfaces in `selfhost/ast/ast.goal`) into goal
  `sealed interface` declarations.
- **Why:** these *are* the conceptual sealed interfaces of the AST — closed sets
  of node types kept package-private by the unexported markers `declNode()` /
  `stmtNode()` / `exprNode()` / `specNode()`. But converting them is neither
  behavior-preserving nor in-scope:
  1. **Public-API break (oracle).** A goal `sealed interface` lowers to the
     §8.1 encoding: the explicit marker methods are replaced by a synthesized
     `isNAME()` marker, and every concrete node struct must be re-spelled with
     an `implements NAME for T` clause (or `type T struct implements NAME`).
     That changes the package's surface the **US-003 verbatim self-host oracle**
     pins — `selfhost/ast` is modeled byte-for-byte on `go/ast` and its ported
     `internal/ast` tests run against the transpiled package unchanged.
  2. **§9 switch-coexistence blast radius (out of scope).** Per DECISIONS
     §02-match ("Switch-coexistence", §228), a plain `switch` on a closed
     enum/sealed value is a **compile error** in the live `sema` checker. The
     consumers of these category interfaces — `selfhost/sema` (check, resolve,
     question, fields, mustuse, foreign, assert, convert, implements),
     `selfhost/backend` (lower, emit), and `selfhost/parser` (parser,
     goal_construct) — dispatch on them with **dozens of plain
     `switch n := x.(type)` type-switches**. Sealing `Node`/`Expr`/`Stmt`/`Decl`
     would turn every one of those into a closed-switch compile error, forcing a
     tree-wide `switch`→`match` conversion that is explicitly **outside US-007**
     (scoped to `selfhost/ast` only).
- **Over:** sealing only `Decl`/`Stmt`/`Expr`/`Spec` (same §9 problem, same
  oracle problem, smaller only in degree); a "sealed interface with no
  implementors-rewrite" (not expressible — the §8.1 encoding requires the
  `implements` clauses).

### `Walk`'s type-switch stays a Go type-switch — NOT `match`
- **Kind:** refusal (with reason)
- **Refused:** rewriting `Walk`'s `switch n := node.(type)` over the ~60 node
  kinds (`selfhost/ast/walk.goal`) into a goal `match`.
- **Why:** goal `match` is reserved for a **closed enum / sealed-interface
  scrutinee** (§02-match §228); a `match` over node kinds is only legal once
  `Node` is a `sealed interface`. Since sealing `Node` is refused above (oracle
  break + §9 cross-package blast radius), its `match` form has no legal
  scrutinee and the type-switch must remain a plain `switch`. The traversal is
  behavior-identical either way; `match` would buy exhaustiveness checking only
  at the cost of the refused interface conversion.
- **Over:** a `match node { *Ident => …, … }` rewrite (no closed scrutinee type
  to match on; would not compile without first sealing `Node`).

### `FuncMod` and `ChanDir` stay iota `const` blocks — NOT goal `enum`s
- **SUPERSEDED by SEAM-002** (see "SEAM-002 — FuncMod & ChanDir iota → goal enum"
  near the top of this file). This was a SCOPE refusal of the per-package gate
  (US-007 could not edit cross-package `==`/`switch` consumers, and that gate
  required byte-identical emitted Go). Under the relaxed SEAM gate both types ARE
  now goal `enum`s and every consumer is a `match`. The "why" below describes the
  per-package-gate obstacles (cross-package `==` / §9 plain-switch), NOT a
  semantic impossibility — SEAM-CAP/CAP-2 + the atomic tree-wide switch→match
  conversion cleared them. Retained for ledger history; do not act on it.
- **Kind:** refusal (with reason) — superseded
- **Refused:** rewriting `type FuncMod int` + `const ( FuncPlain … iota; FuncFrom;
  FuncDerive )` (goal_decl.goal) and `type ChanDir int` + `const ( SendRecv …
  iota; SendOnly; RecvOnly )` (ast.goal) into goal `enum`s.
- **Why:** both are public, `go/ast`-style **ordered integer** enumerations the
  oracle pins, and both are consumed cross-package by patterns a goal `enum`
  (which lowers to a *boxed sealed interface*, not an `int`, per §01-enums/§8.1)
  cannot serve:
  1. **`==`/`!=` comparison.** `selfhost/sema` reads `FuncMod` with equality
     tests — `fn.Mod != ast.FuncPlain` (question.goal), `fd.Mod != ast.FuncDerive`
     (convert.goal), `d.Mod == ast.FuncFrom || d.Mod == ast.FuncDerive`
     (resolve.goal). Enum values are boxed interface values, not comparable
     integers, so these comparisons would not survive the conversion.
  2. **Plain `switch` over the value (§9).** `selfhost/backend/emit.goal` does a
     plain `switch fn.Mod { case ast.FuncPlain, ast.FuncFrom: … case
     ast.FuncDerive: … }`; `sema/resolve.goal` and `backend/emit.goal` do plain
     `switch` over `ChanType.Dir` (`case ast.RecvOnly: … case ast.SendOnly: …`).
     Sealing these into enums makes each a §9 closed-enum-plain-switch **compile
     error** in `sema` — and all sit in packages **outside US-007's scope**.
  This is the same canonical "ordered/comparable iota int, keep as-is" case as
  `token.Kind` (US-005): the goal `enum` is the idiom for a *closed, unordered,
  payload-or-not* variant set matched with `match`; `FuncMod`/`ChanDir` are
  *ordered, comparable, payload-free integer modifiers* that Go `iota` already
  expresses idiomatically and that goal inherits verbatim. This is the AC-1
  "deliberate decision not to … recorded in DECISIONS.md" branch.
- **Over:** converting `FuncMod`/`ChanDir` and also rewriting every consuming
  `==`/`switch` in sema/backend/parser to `match` (out of US-007 scope; changes
  the oracle-pinned representation from `int`).

### No Result/Option/`?` conversion applies — `selfhost/ast` has no fallible function
- **Kind:** assumption
- **Chose:** leave `selfhost/ast`'s `.goal` source unchanged beyond this ledger
  entry.
- **Why:** the package is **pure AST data plus the total `Walk` traversal** — it
  declares **no function that returns `error`** (the only "error" in the source
  is the word inside a doc comment on `IndexListExpr`). There is therefore **no
  manual `if err != nil` propagation** for `goal fix` to idiomatize into
  Result/`?`, and `Walk` is total (a nil node is a no-op; unknown kinds simply
  fall through). `goal fix selfhost/ast/*.goal` produces **no diff and reports
  nothing** across all five files — AC-2 ("goal fix reports no remaining
  auto-convertible propagation sites") already holds.
- **Over:** an `Option`/`Result` rewrite of the `Pos()`/`End()` accessors or the
  `walk*` helpers (all infallible; returning a sentinel `token.Pos{}` is the
  deliberate in-band signal, mirroring `go/ast`).

## self-host idiomatic audit — US-008 (parser)

> US-008 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of `selfhost/parser` (parser.goal, goal_construct.goal,
> goal_decl.goal, goal_match.goal, goal_stmt.goal). The headline idiom the story
> anticipated is Result/`?` for `(node, error)` parser functions. Following the
> US-005/006/007 pattern: classify each Go-ism against the goal idiom it could
> become, convert where it FITS and is behavior-preserving (intra-package, no
> cross-package caller edits), and record refusals-with-reason here. This audit
> went looking specifically for the intra-package Result/`?` surface the story
> names; it genuinely does not exist (see below), so the outcome is a recorded
> DECISION with **no `.goal` source change**.

### Parser error handling stays the error-accumulator pattern — NO intra-package Result/`?` surface exists
- **Kind:** refusal (with reason)
- **Refused:** rewriting parser functions to return `Result[node, error]` with `?`
  propagation.
- **Why:** the anticipated `(node, error)` propagation simply is not how this
  parser is built. `selfhost/parser` uses an **error-accumulator** design, not
  per-function `(T, error)` propagation:
  1. **No internal `(T, error)` functions.** An exhaustive scan of all five
     `.goal` files for `func … error` returns exactly two hits: the exported
     `ParseFile` and `(*parser).errorf` — and `errorf` returns *nothing* (it is
     the error *recorder*). Every internal parse helper (`parseDecl`,
     `parseStmt`, `parseExpr`, `parseType`, `parseGenDecl`, the `goal_*`
     constructors, …) returns a **bare `ast` node** (or `nil`); none returns an
     `error`. There is therefore **no `if err != nil { return …, err }`
     propagation site anywhere** for Result/`?` to replace (a grep for
     `err != nil` / `, err` across the package finds only `ParseFile`'s closing
     `errors.Join`). Errors are appended to `parser.errs []error` by `errorf`
     and the parser keeps making progress (`expect`/`advance` always advance);
     this in-band accumulation is the parser's deliberate recovery strategy and
     is behavior-load-bearing (it produces *all* parse errors and a partial
     `*ast.File`, not just the first).
  2. **The one `(T, error)` function is the exported, oracle-pinned entry point.**
     `ParseFile(src string) (*ast.File, error)` joins `p.errs` at the end. It is
     the public API the **US-003 verbatim self-host oracle** pins — the ported
     `internal/parser/parser_test.go` calls it with the `(*ast.File, error)`
     signature unchanged. Converting it to `Result[*ast.File, error]` would break
     that signature (forbidden) and is exactly the case `goal fix` already refuses:
     `goal fix selfhost/parser/parser.goal` emits a result-sig **SKIP** —
     `parser.goal:57: skipped: [result-sig] ParseFile has a non-propagating
     return; not auto-converted to Result` — i.e. it is **not an auto-convertible
     site**. AC-2 ("goal fix reports no remaining auto-convertible propagation
     sites") therefore holds: zero auto-convertible sites remain.
- **Over:** converting `ParseFile` to `Result` and rewriting its cross-package
  callers + the reused oracle test (out of scope; changes the oracle-pinned public
  API); inventing `(T, error)` returns on the internal helpers purely to apply
  `?` (would replace the accumulate-and-recover strategy, changing error-reporting
  behavior — not behavior-preserving).

### No `switch`→`match` applies — `selfhost/parser` declares no in-file `enum`
- **Kind:** refusal (with reason)
- **Refused:** converting the package's `switch` statements to `match`.
- **Why:** AC-1 scopes the conversion to "switch over an in-file enum", and
  `selfhost/parser` declares **no `enum`** of its own (it *parses* enum syntax in
  other source; it has none). Every `switch` in the package is over a **non-enum
  scrutinee**, for which `match` has no legal subject (§02-match §228):
  - over `token.Kind` (`p.kind()`, `tok`, `k`) — `Kind` is `type Kind int`,
    deliberately NOT a goal `enum` per the US-005 decision;
  - type-switches `x.(type)` / `s.(type)` / `fun.(type)` over the `ast` category
    interfaces (`Node`/`Stmt`/`Expr`/`Decl`) — those cannot be sealed per the
    US-007 decision (§9 switch-coexistence + oracle break), so they remain plain
    type-switches;
  - bare `switch { … }` over boolean conditions (e.g. `describe`,
    `parseSimpleStmt`).
  None is a closed-enum value, so each stays a plain `switch`.
- **Over:** sealing `token.Kind` or the `ast` interfaces to manufacture a `match`
  scrutinee (both refused cross-package conversions per US-005/US-007).

### No `Option` conversion applies — the bool helpers are pure predicates
- **Kind:** assumption
- **Chose:** leave the package's `bool`-returning helpers unchanged.
- **Why:** the `bool` funcs (`at`, `onNewLine`, `atTypeParams`, `nameThenType`,
  `startsType`/`startsTypeKind`, `startsExpr`, `startsArmStmt`, `isContextual`,
  `isTypeSwitchGuard`) are **pure predicates**, not fallible comma-ok lookups with
  a missing-value case; there is no `(T, bool)` lookup in the package. `Option`
  would be neither meaningful nor behavior-preserving for a predicate.
- **Over:** an `Option` rewrite of any predicate (no absent-value semantics to
  model).

### Verification
`goal fix selfhost/parser/*.goal` → no content diff on any file; the only stderr
is the `ParseFile` result-sig SKIP above (a deliberately non-auto-convertible
site). `task check` (incl. the `internal/selfhost` port gate that builds and tests
the transpiled `selfhost/parser` against `internal/parser/parser_test.go`, plus
`internal/parser`) green; `task build` green; `task fixpoint` → FIXPOINT OK
(`selfhost/parser/*.go` byte-identical across goal-c-1/goal-c-2). The
`selfhost/parser` `.goal` source is unchanged.

## self-host idiomatic audit — US-009 (sema)

> US-009 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of `selfhost/sema` (12 files: analyze, assert, check, convert,
> fields, foreign, implements, mustuse, package, question, resolve, sema). sema is
> the FIRST ported package with a genuine fallible `(T, error)` surface (the
> US-004 result-sig SKIPs and the US-008 carry-forward note both name it). Unlike
> US-005..US-008 (which were pure refusals), this audit makes ONE genuine
> behavior-preserving Result/`?` conversion and refuses the rest with reason.
> Pattern unchanged: classify each Go-ism against the goal idiom it could become,
> convert where it FITS (intra-package, behavior-preserving, no cross-package
> caller edits, no oracle-pinned signature change), record refusals here.

### `Analyze` converted to `Result[[]Diagnostic, error]` with `?` — the one fitting site
- **Kind:** conversion
- **Did:** `Analyze(src string) ([]Diagnostic, error)` → `Analyze(src string)
  Result[[]Diagnostic, error]`, propagating the parse step with
  `parser.ParseFile(src)?` and returning success via
  `return Result.Ok(Check(file, info))` (was `file, err := parser.ParseFile(src);
  if err != nil { return nil, err }` … `return Check(file, info), nil`).
- **Why it FITS:** (1) `Analyze` has a real `(T, error)` propagation site
  (`parser.ParseFile` returns `(*ast.File, error)` = open-E ModeResult), which is
  exactly what `?` is for (feature-05). (2) It has **zero consumers anywhere in the
  selfhost tree** and **zero oracle tests** (no `analyze_test` is in the sema
  behavioral gate; grep finds no `Analyze(` call in `selfhost/**` or the copied
  `internal/sema/*_test.go`), so the conversion needs **no caller or test edits**.
  (3) Behavior is preserved EXACTLY: open-E `Result[T, error]` lowers to native
  `(T, error)`, so the emitted Go is `func Analyze(src string) (ok []Diagnostic,
  err error) { file, err := parser.ParseFile(src); if err != nil { return ok, err
  } … return Check(file, info), nil }` — the same two-value `([]Diagnostic, error)`
  public signature and the same propagation, just spelled idiomatically in goal
  source. The fixpoint stays byte-identical (both bootstrap stages emit this same
  form). This is the canonical "exported function whose emitted signature is
  unchanged" case: changing the goal-source idiom does NOT change the oracle-pinned
  emitted API.

### Exported, oracle-pinned, cross-package APIs stay `(T, error)`/`[]error` — NOT converted
- **Kind:** refusal (with reason)
- **Refused:** converting `EnrichForeign`, `DefaultResolver`, `AnalyzePackageInDir`,
  `AnalyzePackageInDirWith` to Result.
- **Why:** each is exported and either consumed cross-package or pinned by the
  US-003 verbatim oracle tests, so a signature change would require out-of-scope
  cross-package/cross-file edits:
  - `EnrichForeign(…) []error` is an **error-ACCUMULATOR** (it appends per-import
    errors and `continue`s — `errs = append(errs, err)`), not a `(T, error)`
    propagation site; `?` cannot apply to a function that returns a list of
    *collected* errors. `package.go` calls it (in-package) and `foreign_test.go`
    pins its `[]error` shape.
  - `DefaultResolver(importPath, fromDir string) (string, error)` **is the value
    assigned to the exported `DirResolver` func type** (`resolve = DefaultResolver`)
    and `foreign_test.go` passes resolvers of that exact type. Converting it to
    Result breaks the `DirResolver` contract and the tests. `goal fix` already
    SKIPs it ("non-propagating return").
  - `AnalyzePackageInDir ([][]Diagnostic, error)` / `AnalyzePackageInDirWith
    ([][]Diagnostic, []error, error)` are exported and pinned by `package_test.go`;
    the `With` variant returns **three** values (a Result holds one success value)
    and `InDir` calls it three-valued, so `?` cannot apply. `goal fix` SKIPs both
    ("non-propagating return" / "returns multiple non-error values").

### Unexported helpers `foreignDecls` / `goListResolve` stay `(…​, error)` — NOT converted
- **Kind:** refusal (with reason)
- **Refused:** converting these two internal helpers to Result.
- **Why:** `foreignDecls(dir, alias) (structs, funcs, methods map…, err error)`
  returns **four values** (three maps + error); a Result holds one success value
  (would need a struct bundle), and its sole caller `EnrichForeign` **accumulates**
  the error rather than propagating it, so there is no `?` call site even if it were
  bundled. `goListResolve(…) (string, error)` is **tail-returned** by the exported
  `DefaultResolver` (`return goListResolve(...)`, which stays `(string, error)`), a
  non-propagating boundary where `?` cannot apply — exactly the `goal fix` SKIP.

### comma-ok value helpers stay `(T, bool)` — NOT converted to Option
- **Kind:** refusal (with reason)
- **Refused:** converting `constIntLit (int64, bool)`, `moduleResolve
  (string, bool)`, `readModulePath (string, bool)` to `Option`.
- **Why:** these are **comma-ok control-flow** helpers: each maps two distinct
  failures (e.g. "not a BasicLit" vs "ParseInt failed"; "no go.mod" vs "no module
  directive"; "no enclosing module" vs "path outside module") onto a **single bool
  the in-file caller branches on** (`if v, ok := f(…); ok { … }`). The inner
  `err != nil` is deliberately *swallowed to a bool*, not propagated — `?` has
  nothing to propagate and `Option` would change the `(T, bool)` contract the
  in-file callers consume. This is the established comma-ok refusal (US-005). The
  remaining `goal fix` output for these is an advisory `suggestion: [call-site]`
  ("manual error handling in X … convert its signature to use ?"), NOT an
  auto-conversion: `goal fix` produces **no source diff**, so AC-2 ("no remaining
  auto-convertible propagation sites") holds.

### `Mode` and `Severity` stay `type X int` + iota — NOT converted to `enum`
- **SUPERSEDED by SEAM-003** (see "SEAM-003 — Mode & Severity iota → goal enum"
  at the end of this file). The refusal below was a SCOPE refusal ("cross-package
  consumers, out of scope for a single-package story"), not a semantic one: unlike
  `token.Kind`, neither `Mode` nor `Severity` carries genuine numeric identity
  (no array index, range predicate, or wire value — the claimed `sema.Severity(x)`
  numeric conversions do not actually exist in the tree). Under the relaxed seam
  gate SEAM-003 converts both to goal `enum`s with every cross-package consumer.
  Retained for the audit trail.
- **Kind:** refusal (with reason)
- **Refused:** rewriting `Mode` (ModeNone/ModeResult/ModeResultClosed/ModeOption)
  and `Severity` (Error/Warning) as goal `enum`s.
- **Why:** both are **exported, ordered iota ints** consumed cross-package by `==`
  and numeric conversion: `sig.Mode == sema.ModeResultClosed` and
  `calleeMode(…) sema.Mode` in `selfhost/backend` (lower.goal, emit.goal);
  `Severity: sema.Error/Warning`, the `sema.Severity` field type, and
  `sema.Severity(x)` numeric conversions in `selfhost/typecheck` (and lsp). A goal
  `enum` lowers to a boxed sealed interface (§8.1), not an `int`, so `==`,
  ordering, and the numeric conversions all break and every cross-package consumer
  would need a coordinated edit — out of scope for a single-package story. This is
  the same canonical "ordered/comparable iota int, keep as-is" case as `token.Kind`
  (US-005) and `FuncMod`/`ChanDir` (US-007).

### No `switch`→`match` applies — `selfhost/sema` declares no in-file `enum`
- **Kind:** refusal (with reason)
- **Refused:** converting the package's `switch` statements to `match`.
- **Why:** AC-1 scopes the conversion to "switch over an in-file enum", and sema
  declares **no `enum`** of its own (it *resolves and checks* enum syntax in other
  source). Its switches are over `token.Kind` (`type Kind int`, not an enum per
  US-005), type-switches over `ast` category interfaces (cannot be sealed per
  US-007), `Mode`, and strings — none a closed-enum scrutinee (§02-match §228). The
  diagnostic kinds `Diagnostic.Code` and `Diagnostic.Feature` are stable **string**
  identifiers, not enum kinds, so they are not enum/match candidates either.

### Verification
`goal fix selfhost/sema/*.goal` → no content diff on any file (`Analyze` no longer
appears in the report at all now that it is Result-returning); the only stderr is
the deliberately non-auto-convertible SKIPs (DefaultResolver, goListResolve,
AnalyzePackageInDir/With) and advisory call-site suggestions (EnrichForeign,
foreignDecls, moduleResolve, readModulePath, constIntLit, AnalyzePackageInDirWith)
— zero auto-convertible sites remain. `go test ./internal/selfhost -run
TestPortedSemaPackage` green (the transpiled `selfhost/sema` with the Result/`?`
`Analyze` compiles and passes the copied `internal/sema` behavioral suites).
`task check` (incl. the selfhost port gate + `internal/sema`) green; `task build`
green; `task fixpoint` → FIXPOINT OK (`selfhost/sema/*.go` byte-identical across
goal-c-1/goal-c-2). Only `selfhost/sema/analyze.goal` changed.

## self-host idiomatic audit — US-010 (project + pipeline)

> US-010 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of the two smallest selfhost packages, combined into one story
> — `selfhost/project` (project.goal) and `selfhost/pipeline` (pipeline.goal,
> sourcemap.goal). Pattern unchanged from US-005..US-009: classify each Go-ism
> against the goal idiom it could become, convert where it FITS (intra-package,
> behavior-preserving, no cross-package caller edits, no oracle-pinned signature
> change), record refusals here. Outcome: a documented refusal for both packages
> (no `.goal` source change) — every fallible function is either oracle-pinned or
> has no valid `?` host, and `pipeline` has no fallible surface at all. This
> mirrors the pure-refusal leaf packages (token/lexer/ast/parser, US-005..US-008)
> rather than the one-conversion sema audit (US-009).

### `Discover` stays `([]*Package, error)` — NOT converted to Result
- **Kind:** refusal (with reason)
- **Refused:** converting `Discover(root string) ([]*Package, error)` to
  `Result[[]*Package, error]`.
- **Why:** `Discover` is exported AND heavily oracle-pinned: the US-003 verbatim
  oracle suite `internal/project/project_test.go` calls it five ways
  (TestDiscoverGroupsByDirectory / ConflictingPackageNames / MissingPackageClause
  / SkipsReservedDirs / EmptyTreeYieldsNoPackages, all `pkgs, err := Discover(...)`
  / `_, err := Discover(...)`), and the self-host port gate
  `internal/selfhost/port_test.go` calls `project.Discover(...)` for **every**
  ported package (token, lexer, ast, parser, sema, project, pipeline) in its
  two-value form. The open-E Result lowering would keep the emitted Go signature
  `([]*Package, error)` byte-identical, BUT the body uses `?` only if the enclosing
  function is Result-returning, and the two collapsible propagation sites it owns
  (`name, err := packageName(...); if err != nil { return nil, err }` and the walk
  closure's `if err != nil { return nil, err }`) cannot be `?`-ified without making
  `Discover` itself Result-typed in goal source — which is precisely what `goal fix`
  reports it cannot do ("exported `Discover` has callers fix cannot see"). The
  US-009 safety rule converts an exported fn ONLY when it has no in-tree callers
  AND no oracle test; `Discover` fails both. Refuse.

### `packageName` stays `(string, error)` — NOT converted to Result
- **Kind:** refusal (with reason)
- **Refused:** converting the unexported `packageName(dir string, files []File)
  (string, error)` to Result.
- **Why:** its sole caller is `Discover`, which is not Result-returning and cannot
  become one (oracle-pinned, above). With no Result host, a `?` at the call site is
  impossible; converting `packageName` to Result would force the caller to manually
  unpack a Result (a struct match / `.Unwrap`) — strictly MORE code than the
  current `name, err := …; if err != nil { return nil, err }`, not a
  behavior-preserving idiom gain. Independently, `packageName` builds its errors in
  place (`return "", fmt.Errorf(...)`) rather than propagating an existing one, so
  `goal fix` SKIPs it as a "non-propagating return" — it is not an auto-convertible
  site. Refuse.

### `PackageClause` is not a `(T, error)` function — nothing to convert
- **Kind:** N/A (not a conversion candidate)
- **Note:** `PackageClause(src string) string` returns a plain `string` and
  deliberately swallows the parser error (`file, _ := parser.ParseFile(src)`) to
  preserve the legacy lexer's tolerance (it reads the package name even when later
  body parsing failed). There is no error in its signature, so Result/`?`/Option
  do not apply. It is also exported and oracle-pinned (TestPackageClause*), so its
  signature is frozen regardless.

### `selfhost/pipeline` has no fallible surface — nothing to convert
- **Kind:** N/A (no fallible functions)
- **Note:** `pipeline.goal` is pure type declarations (Output, GoFile,
  PackageOutput) — no functions at all. `sourcemap.goal`'s functions
  (`AddLineDirectives` → string, `declSites` → []declSite, `declLines` →
  map[string]int, `declName` → string) return no error; `declSites` swallows the
  parse error with `file, _ := parser.ParseFile(src)`. No `(T, error)`, no
  comma-ok `(T, bool)`, no error-returning helper exists, so Result/Option/`?`
  have no surface here. `goal fix` reports nothing for either file.

### No `switch`→`match` applies — neither package declares an in-file `enum`
- **Kind:** refusal (with reason)
- **Refused:** converting the `switch` statements to `match`.
- **Why:** the only `switch`es are in `sourcemap.go declName`: a type-switch over
  `ast.Decl` (a category interface that **cannot be sealed** per US-007 §9 — its
  values are consumed by plain switches in sema/backend/parser, and sealing it is a
  cross-package break) with a nested type-switch over `ast.Spec` (`TypeSpec` /
  `ValueSpec`), gated by `d.Tok == token.IMPORT` comparisons over `token.Kind`
  (`type Kind int`, not an enum per US-005). Neither package declares an `enum` of
  its own, so AC-1's "switch over an in-file enum → match" has no candidate. Refuse
  (consistent with US-007/US-008).

### No `enum` / `Option` applies
- **Kind:** N/A
- **Note:** neither package has a closed, unordered, payload-or-not variant set
  (the only `const` is `Ext = ".goal"`, a single string), nor a comma-ok value
  helper. enum and Option have no candidate.

### Verification
`goal fix selfhost/project/*.goal` and `goal fix selfhost/pipeline/*.goal` →
**no content diff** on any file. The only stderr is project.goal's two deliberate
non-auto-convertible SKIPs (`Discover` exported-with-hidden-callers, `packageName`
non-propagating-return) plus one advisory `suggestion: [call-site]` on `Discover`
(not an auto-conversion, no diff); pipeline produces no output at all. Zero
auto-convertible propagation sites remain in either package (AC-2 holds). The
self-host port gate `go test ./internal/selfhost -run
'TestPortedProjectPackage|TestPortedPipelinePackage'` is green (both transpile to
compiling Go and pass the copied `internal/project` / behavioral suites);
`task check` (incl. the port gate + `internal/project` + `internal/pipeline`)
green; `task build` green; `task fixpoint` → FIXPOINT OK
(`selfhost/{project,pipeline}/*.go` byte-identical across goal-c-1/goal-c-2). No
`.goal` source changed — this story is documentation only.

## self-host idiomatic audit — US-011 (backend)

> US-011 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of `selfhost/backend` — the LARGEST ported package
> (arity/backend/doctest/emit/lower/package.goal, ~3,447 LOC). Pattern unchanged
> from US-005..US-010: classify each Go-ism against the goal idiom it could
> become, convert where it FITS (intra-package, behavior-preserving), record
> refusals here. Outcome: a GENUINE conversion of the derive/`from`-conversion
> subsystem to idiomatic Result/`?`/`match` (a coherent complete subset), plus
> documented refusals for the pervasively-wrapping and oracle/interface-pinned
> functions. This is the second package (after sema, US-009) with real executable
> Result/`?` conversions, and the first to use `match` in executable selfhost
> source.

### KEY front-end fact: the EMITTER gates `?` to a Result/Option host
- The **sema** mode model (`selfhost/sema/sema.goal:39`) says a `?` host may be
  "a Result, a `func(…) error`, or a tuple ending in error" — so sema *accepts*
  `?` inside a plain `(T, error)` function. BUT the **backend emitter** is
  stricter: `emit.goal` rejects `?` whose enclosing function is not Result/Option
  with `backend: \`?\` outside a Result- or Option-returning function`. So to
  use `?` at a pure-propagation site, the ENCLOSING function must be converted to
  `Result` (matching what `goal fix` already insists on — its `suggestion`
  reports all say "…which is not Result-returning; convert its signature to use
  `?`"). This is why a `(T,error)` host cannot simply gain a `?`.

### CONVERTED: the derive-conversion cluster → Result/`?`/`match`
- **Kind:** conversion (behavior-preserving, intra-package, byte-equivalent at
  the emit boundary)
- **`elemConv` (lower.goal)** stays `(func(string) string, error)` — it is the
  leaf callee; the `?` requirement is on the HOST, not the callee, so it needs no
  signature change (and a func-typed `Result.Ok` payload is avoided).
- **`resolveField` (emit.goal)** `([]string, error)` → `Result[[]string, error]`.
  It owns NO error-wrapping internally (every error return is either a direct
  `fmt.Errorf("no conversion…")` or a pure `return nil, err` propagation of
  `elemConv`/`deriveBody`), so it converts cleanly: the 4 `elemConv(...)` sites
  and the 1 `e.deriveBody(...)` site become `?`; the other returns become
  `Result.Ok(...)` / `Result.Err(...)`. (The `return out, %s` at the fallible-leaf
  branch is GENERATED-Go *string* text, not a goal return — untouched.)
- **`deriveBody` (emit.goal)** `([]string, error)` → `Result[[]string, error]`
  (same shape as US-009's `sema.Analyze`). Its returns become `Result.Ok/Err`; its
  one call to `resolveField` *wraps* the error (`nested field %q: %w`), so that
  consumer uses `match { Result.Ok(s) => … Result.Err(err) => return
  Result.Err(fmt.Errorf("nested field %q: %w", …)) }` — `?` would drop the wrap.
- **`genConversion` (emit.goal)** stays `void`/`e.fail`-based (it has no error
  return), so its call to the now-`Result` `resolveField` uses `match` whose `Err`
  arm calls `e.fail("derive %s, field %q: %v", …)` and `return`s — never `?`.
- **Why behavior-preserving + fixpoint-safe:** open-E `Result[T, error]` lowers to
  native `(T, error)`; `f()?` lowers to `v, err := f(); if err != nil { return
  zero, err }`; statement-position `match` over an open-E Result lowers to
  `v, err := f(); if err != nil { <Err arm> } else { <Ok arm> }`
  (`emit.goal resultMatch`/`posStmt`). All three are the manual code they replace,
  so the cluster's runtime behavior (including every diagnostic string) is
  identical. The fixpoint compares goal-c-1 vs goal-c-2 (both built from this
  converted source), so it stays byte-identical; the verbatim oracle
  (`internal/backend`, unchanged) plus the corpus behavioral tier confirm the
  derive output is unchanged.

### REFUSED: `Transpile` stays `(pipeline.Output, error)`
- **Kind:** refusal (with reason)
- **Why:** exported AND oracle-pinned (the `internal/backend` suite drives it as
  the AST-engine entry point, and `corpus.TranspilerFunc` adapts it). It also
  *wraps* two of its three propagation sites (`parse: %w`, `generated Go did not
  parse: %w`), and only the middle `goBackend{}.Emit(...)` site is pure. Making
  `Transpile` Result to `?`-ify that one site would force the two wrapping sites
  onto `match` and break callers/`corpus` that consume `(pipeline.Output, error)`.
  Open-E lowering keeps the emitted signature identical, but the cross-cutting
  caller + wrap churn is not a behavior-preserving per-package win. `goal fix`
  SKIPs it ("non-propagating return"). Refuse.

### REFUSED: the `Backend.Emit` / `goBackend.Emit` interface method
- **Kind:** refusal (with reason)
- **Why:** `Emit` is pinned by the `Backend` interface (`Emit(...)
  (pipeline.Output, error)`); converting the method to `Result` while the
  interface declares the tuple would (at goal source level) no longer satisfy the
  interface, requiring an interface-wide change beyond this story's scope. `Emit`
  also wraps its `emitDoctests` site (`doctests: %w`); only the `emitFile` site is
  pure. Refuse (kept `(T, error)`; `goal fix` emits an advisory `suggestion` only,
  not an auto-conversion).

### REFUSED: `TranspilePackage`, `GoFormatter.Format`
- **Kind:** refusal (with reason)
- **Why:** `TranspilePackage(pkg) (pipeline.PackageOutput, error)` is exported +
  oracle-pinned (package-mode corpus + backend tests) and wraps its propagation
  (`format …`). `GoFormatter.Format` is a `Formatter`-interface method that simply
  returns `format.Source(src)` (a foreign `(…, error)` tail call — a
  non-propagating return, not an `if err != nil` site). Neither is a clean,
  behavior-preserving Result conversion. `goal fix` SKIPs `TranspilePackage`.

### N/A: `emitFile` / `emitFileWith` / `emitDoctests`
- **Kind:** N/A (shape is not a clean `(T, error)` propagation)
- **Note:** `emitFileWith` returns 3 values `(src string, usedOption bool, err
  error)` and builds its error from the emitter accumulator (`e.err`), not by
  propagating a callee; `emitFile` is a 2-value pass-through of it (`src, _, err
  := emitFileWith(...); return src, err` — a non-propagating return `goal fix`
  SKIPs); `emitDoctests` is likewise 3-value and accumulator-based with wrapping
  callers. None is an open-E `(T, error)` with a collapsible `if err != nil`
  propagation site, so none is a Result/`?` candidate.

### REFUSED: no `switch`→`match` — backend declares NO in-file `enum`
- **Kind:** refusal (with reason)
- **Why:** `emit.goal` (46 switches) and `lower.goal` (9) switch exclusively over
  NON-enum scrutinees: `ast` category interfaces (`ast.Decl`/`Expr`/`Stmt`/`Spec`
  type-switches — unsealable per US-007 §9, consumed by plain switches across
  sema/backend/parser), `token.Kind` (`type Kind int`, not an enum per US-005),
  `rune`, and boolean `switch {}`. Backend defines no `enum` of its own, so AC-1's
  "switch-over-in-file-enum → `match`" has no candidate. (The `match` added by this
  story is over Result, the §8.3 idiom — not an enum/switch conversion.) Refuse,
  consistent with US-006/US-007/US-008/US-010.

### Verification
`goal fix -inplace selfhost/backend/*.goal` → **no source diff** on any file (the
former `genConversion`/`resolveField`/`deriveBody` `suggestion`s are gone now that
the cluster is Result/`?`/`match`). Remaining stderr is only deliberate
non-auto-convertible reports: `skipped` on `Transpile`/`emitFile`/`elemConv`/
`TranspilePackage` (non-propagating returns) and advisory `suggestion`s on
`Emit`/`Transpile`/`emitDoctests`/`TranspilePackage` (the documented refusals) —
neither is an auto-conversion, so AC-2 (zero remaining auto-convertible
propagation sites) holds. `go test ./internal/selfhost -run
TestPortedBackendPackage` green (the converted backend transpiles to compiling Go
and passes the copied `internal/backend` behavioral suite); `task check` (incl.
the port gate + `internal/backend` + the corpus transpile/behavioral tiers) green;
`task build` green; `task fixpoint` → FIXPOINT OK (`selfhost/backend/*.go`
byte-identical across goal-c-1/goal-c-2).

## self-host idiomatic audit — US-012 (typecheck)

> US-012 of the **self-host idiomatic** PRD (`prd.json`): the per-package
> idiomatic audit of the FINAL package, `selfhost/typecheck` (the go/types-over-
> lowered-Go depth-checker harness: checker.goal, typecheck.goal, implements.goal,
> mustuse.goal, nozero.goal). Pattern unchanged from US-005..US-011: classify each
> Go-ism against the goal idiom it could become, convert where it FITS
> (intra-package, behavior-preserving, no cross-package caller edits, no
> oracle-pinned signature change), record refusals here. Outcome: a documented
> refusal (no `.goal` source change) — the only two error-returning functions are
> exported, oracle-pinned `Load` (wrapping propagation, non-Result callees) and the
> `TypeChecker`-interface-pinned `Check` method; every depth check returns
> `[]Diagnostic` with no error at all. This mirrors the pure-refusal packages
> (token/lexer/ast/parser, US-005..US-008, and project/pipeline, US-010) rather
> than the conversion audits (sema US-009, backend US-011).

### `Load` stays `(*Package, error)` — NOT converted to Result
- **Kind:** refusal (with reason)
- **Refused:** converting `Load(pkg *project.Package) (*Package, error)` (typecheck.goal)
  to `Result[*Package, error]`.
- **Why:** every internal propagation site in `Load` WRAPS context —
  `fmt.Errorf("transpile: %w", err)`, `fmt.Errorf("parse goal source %s: %w", …)`,
  `fmt.Errorf("parse generated %s: %w", …)`. `?` propagates the error UNCHANGED, so
  `?`-ifying these would DROP the context = a behavior change (the §-wrapping rule).
  Independently, the callees those sites guard (`backend.TranspilePackage`,
  `goalparser.ParseFile`, `go/parser.ParseFile`) return Go `(T, error)` tuples, not
  Result, so there is nothing to `?`. And `Load` is exported AND heavily
  oracle-pinned: the US-003 verbatim oracle suite calls `p, err := Load(…)` (and
  `_, err := Load(…)`) in SIX places across the depth tests
  (implements_test.go:10, checker_test.go:13, mustuse_test.go:12,
  nozero_test.go:12 & :212, typecheck_test.go:42 & :73) and it has an in-tree
  caller (`GoTypesChecker.Check`). The US-009 safety rule converts an exported fn
  ONLY when it has no in-tree callers AND no oracle test; `Load` fails both.
  Converting only its return wrapper to Result would introduce zero `?`, force the
  interface-pinned `Check` into a `match` wrapper (below), and leave the wrapping
  `if err != nil` blocks intact — net churn, no idiom gain. `goal fix` agrees: it
  emits `skipped: [result-sig] Load has a non-propagating return; not
  auto-converted to Result`. Refuse.

### `GoTypesChecker.Check` stays `([]Diagnostic, error)` — NOT converted to Result
- **Kind:** refusal (with reason)
- **Refused:** converting the `TypeChecker.Check(pkg *project.Package) ([]Diagnostic,
  error)` interface method (checker.goal) to a Result-returning signature.
- **Why:** `Check` is pinned by the `TypeChecker` interface — the seam's entire
  purpose is a swappable interface so a native goal checker can replace the
  go/types crutch WITHOUT changing any caller (checker.goal doc). The oracle suite
  pins it directly: `var _ TypeChecker = GoTypesChecker{}` (checker_test.go:33) and
  `got, err := tc.Check(pkg)` / `_, err := tc.Check(bad)` (checker_test.go:63,84,99)
  consume it as a two-value method through the interface. Its lone propagation
  `p, err := Load(pkg); if err != nil { return nil, err }` IS pure, but the
  backend emitter requires a `?`'s host to be Result-returning; making `Check`
  Result would change the interface contract and break every consumer. `goal fix`
  only emits an advisory `suggestion: [call-site]` here (not a `skipped`, not a
  diff) — advisory, not auto-convertible. Refuse.

### `litClass` stays `type litClass int` + iota — NOT converted to `enum`
- **Kind:** refusal (with reason)
- **Refused:** expressing the `litClass` iota block (`classElided`/`classGeneric`,
  nozero.goal) as a goal `enum`.
- **Why:** there is NO switch over `litClass`, so AC's "switch-over-in-file-enum →
  `match`" has no candidate (the only switch in nozero.goal is a TYPE switch over
  `go/ast` node types — `ast.IndexExpr`/`IndexListExpr` — which is foreign and
  unsealable per US-007 §9). `litClass` is consumed via `kind == classGeneric`
  (litMessage) and produced via a numeric `return 0, false` (litClassOf's default).
  A goal `enum` lowers to a sealed interface with NO integer identity (US-005 / §01),
  so `==` comparison and the `return 0` literal would both break — a behavior
  change, not a behavior-preserving idiom gain. Same "keep as iota int" verdict as
  `token.Kind` (US-005) and the ast modifiers (US-007). Refuse.

### Verification
`goal fix selfhost/typecheck/*.goal` → **no source diff** on any of the five files.
The only stderr is the deliberate non-auto-convertible reports: `skipped:
[result-sig]` on `Load` (non-propagating return) and advisory `suggestion:
[call-site]` on `Load` and `Check` — neither is an auto-conversion, so AC-2 (zero
remaining auto-convertible propagation sites) holds with no source change. The
selfhost port gate (`internal/selfhost` port_test) transpiles `selfhost/typecheck`
and runs the copied `internal/typecheck` depth tests against it; `task check`
(incl. that port gate + `internal/typecheck`) green; `task build` green; `task
fixpoint` → FIXPOINT OK (`selfhost/typecheck/*.go` byte-identical across
goal-c-1/goal-c-2, since the package is unchanged). No `.goal` source changed.

## self-host idiomatic audit — US-013 (final whole-tree sweep + self-host proof)

> US-013 of the **self-host idiomatic** PRD (`prd.json`): the CLOSING story. After
> the per-package audits (US-005..US-012) this is a whole-compiler proof that no
> auto-convertible plain-Go propagation remains anywhere in the `selfhost/` tree,
> that every remaining deliberately-Go construct is documented, and that the
> idiomatic compiler still self-hosts to a byte-identical fixpoint while passing the
> corpus. No `.goal` source change — the tree is already at the `goal fix` fixed
> point; touching source would risk the byte-identical oracle.

### Whole-tree machine proof (AC-1: zero auto-convertible propagation sites)
- **Kind:** proof
- Running `goal fix -inplace` over a COPY of the entire tree (all 39
  `selfhost/**/*.goal` files) and `diff -r`-ing against the original yields an EMPTY
  diff. (A bare `goal fix` always prints the possibly-unchanged rewritten file to
  stdout, so the reliable "did it convert anything" check is the `-inplace`-on-a-copy
  diff, not stdout inspection — see progress.txt patterns.)
- The whole-tree stderr report contains ONLY `skipped: [result-sig]` (refusals) and
  `suggestion: [call-site]` (advisory) lines — and NO `fixed` lines. Neither a skip
  nor a suggestion is an auto-conversion, so AC-1 (zero remaining auto-convertible
  propagation sites) holds with no source change. The autofixer has reached its
  fixed point across the whole compiler.

### Every flagged construct maps to a documented refusal
- **Kind:** roll-up
- Each function the whole-tree report flags is a deliberately-Go construct already
  documented in this file by its per-package audit:
  - `ParseFile` (parser) — US-008 (recovery-by-accumulation; exported oracle-pinned).
  - `Discover`/`packageName` (project) — US-010.
  - `EnrichForeign`/`foreignDecls`/`DefaultResolver`/`goListResolve`/`moduleResolve`/
    `readModulePath`/`constIntLit`/`AnalyzePackageInDir`/`AnalyzePackageInDirWith`
    (sema) — US-009.
  - `Transpile`/`Emit`/`emitDoctests`/`emitFile`/`elemConv`/`TranspilePackage`
    (backend) — US-011.
  - `Load`/`Check` (typecheck) — US-012.
- The ONLY file never given a per-package audit story is the top-level
  `selfhost/main.goal`; its two refusals are recorded immediately below. With those
  documented, every flagged construct in the whole-tree sweep is accounted for.

### `run` stays `(error)` — NOT converted to Result (main.goal)
- **Kind:** refusal (with reason)
- **Refused:** converting `run(args []string) error` (selfhost/main.goal) to a
  Result-returning signature.
- **Why:** `run` is the CLI entry point — `main()` consumes it as
  `if err := run(os.Args[1:]); err != nil { … os.Exit(1) }`. It returns a BARE
  `error` (no value channel), and the result-sig rule refuses bare-error functions:
  `goal fix` emits `skipped: [result-sig] run returns a bare error; not
  auto-converted to Result`. Its propagation is a mix that does not fit `?`: several
  returns CONSTRUCT usage errors via `fmt.Errorf` (`usage: …`, `unknown flag %q`,
  `selfhost requires --emit=…`, `no .goal packages found …`) which are not
  propagation sites at all, and the genuine propagations (`return err` from
  `project.Discover`, `backend.TranspilePackage`, `emitPackage`) bottom out at Go
  `(T, error)` / bare-error callees, not Result, so there is nothing to `?`.
  Converting a bare-error top-level CLI entry to Result buys zero idiom and changes
  the program's plumbing. Same bare-error refusal class as typecheck `Load`/`Check`
  (US-012). Refuse.

### `emitPackage` stays `(error)` — NOT converted to Result (main.goal)
- **Kind:** refusal (with reason)
- **Refused:** converting `emitPackage(pkg *project.Package, out pipeline.PackageOutput,
  emitDir string) error` (selfhost/main.goal) to Result.
- **Why:** `emitPackage` is an IO helper called only by `run` (`if err :=
  emitPackage(…); err != nil { return err }`). It returns a bare `error` and
  propagates `os.MkdirAll` / `os.WriteFile` failures — both bare-error Go stdlib
  calls, nothing to `?`. The result-sig rule refuses it the same way it refuses
  `run` (bare-error host, no value channel). Making it Result would force `run` (its
  only caller, itself a documented refusal) into a `match`/Result wrapper for no
  gain. Refuse.

### Verification (AC-2: fixpoint; AC-3: corpus)
- `task check` green: `go vet ./...` + the full `go test ./...` suite, which includes
  the `internal/corpus` transpile/behavioral/check tiers AND the `internal/selfhost`
  behavioral port gates (each transpiles a `selfhost/<pkg>` through the goal front end
  and runs the copied `internal/<pkg>` tests against the emitted Go) — i.e. the
  goal-built packages pass the corpus + behavioral tiers.
- `task build` green (both binaries).
- `task fixpoint` → FIXPOINT OK: stage-0 builds goal-c-1, goal-c-1 builds goal-c-2,
  and `diff -r` of the two stages' emit over `./selfhost` is empty — goal-c-1 and
  goal-c-2 emit byte-identical Go for the compiler's own source. The self-host is
  PROVEN: a goal-built, idiomatic-goal compiler compiles itself to a byte-identical
  fixpoint while passing the corpus. No `.goal` source changed in this story.

## Seam methodology — equivalence oracle for cross-package idiom changes (SEAM-001)

> SEAM-001 of the **self-host idiomatic SEAMS** PRD (`prd.json`, stories
> SEAM-001..006): the OPENING story. The per-package idiomatic audits
> (US-005..US-013, all passed) reached only the intra-package idiomatic surface
> because their gate required emitted Go to stay byte-identical and the US-003
> verbatim oracle signatures to stay fixed — which is exactly what every deep,
> cross-package idiom (seal AST → match, iota → enum, fallible API → Result/?)
> must violate. This section defines the RELAXED gate the seam stories
> (SEAM-002..006) verify against, so "behavior-preserving" has one written
> meaning that does NOT require byte-identical emitted output. Documentation +
> procedure only; no source idiom change in this story.

### The two gates, contrasted
- **Kind:** methodology
- **Per-package gate (US-005..US-013):** an audit was behavior-preserving only if
  the emitted Go stayed **byte-identical** and the oracle-pinned exported/interface
  signatures stayed fixed. Under that gate, every cross-package idiom was correctly
  recorded as a refusal — `token.Kind`/`FuncMod`/`ChanDir` kept as iota int
  (US-005/US-007), `Mode`/`Severity` kept as iota (US-011), the `ast` category
  interfaces left unsealed and their type-switches left as plain switches
  (US-007), and the fallible exported API left as `(T, error)` (US-008..US-012).
  Each refusal was tagged "cross-package, out of scope" — a SCOPE limit of the
  per-package gate, not a semantic impossibility.
- **Seam gate (this PRD):** a seam story is a single ATOMIC cross-package unit
  (type definition + every consumer across package lines, landing together). It is
  EXPLICITLY ALLOWED to change emitted Go. Equivalence is re-proven by the three
  proofs below instead of by byte-identical output. Partial application that
  leaves the tree red (e.g. a sealed interface whose plain-`switch` consumers
  still exist — a §9 switch-coexistence compile error) is NOT a valid intermediate.

### The crux: `task fixpoint` is self-consistency, not output-stability
- **Kind:** methodology
- `task fixpoint` proves **goal-c-1 == goal-c-2** — stage-0 builds goal-c-1,
  goal-c-1 builds goal-c-2, and `diff -r` of the two stages' emit over `./selfhost`
  is empty. This is STAGE1==STAGE2 self-consistency (both bootstrap stages agree on
  the *new* idiomatic form), **NOT** output==before (the new form vs the old form).
- Therefore an emitted-Go change introduced by a seam edit does **not** break the
  fixpoint: as long as both stages are built from the same converted source and
  reach the same fixed point, FIXPOINT OK still holds. This is the single fact that
  makes the relaxed gate sound — the per-package audits leaned on the fixpoint AS
  IF it pinned output, but it only ever pinned self-consistency.

### The three equivalence proofs for a seam edit
- **Kind:** methodology
- A seam edit is behavior-preserving when ALL THREE hold:
  1. **Fixpoint self-consistency** — `task fixpoint` → FIXPOINT OK on the new
     idiomatic source (goal-c-1 and goal-c-2 byte-identical to each other).
  2. **Corpus behavioral tier** — the compiled programs behave identically
     (`internal/corpus` behavioral tier + the `internal/selfhost` behavioral port
     gates, which transpile each `selfhost/<pkg>` through the goal front end and run
     the copied `internal/<pkg>` tests against the emitted Go). Behavioral green is
     the real "same program" proof once exact emitted-Go bytes are allowed to move.
  3. **Reviewed golden regeneration** — any golden/shape fixture whose bytes
     legitimately change is regenerated DELIBERATELY and reviewed for behavior
     preservation (procedure below), never blindly accepted.

### Which tests are EXPECTED to change vs MUST stay byte-green
- **Kind:** methodology
- **EXPECTED to change under a seam edit (regenerate + review, do not treat the
  diff as a regression):**
  - The ported **go/ast-mirror unit tests** in `internal/ast` (ast_test.go and the
    byte-for-byte mirror the US-003 oracle pins). Sealing the `ast` category
    interfaces (SEAM-004) directly changes their shape; that conflict is confronted
    in SEAM-004, with the mirror either updated to the sealed shape (documented
    rationale: the self-hosted AST no longer needs to mirror go/ast) or
    deliberately retained and the seal scoped around it.
  - **Golden transpile-shape fixtures** — the corpus exact-tier goldens
    (`*.go.expected`) and the parser AST snapshots
    (`internal/parser/testdata/snapshots/`). iota→enum and switch→match change the
    emitted Go, so these shift by design.
- **MUST stay byte-green (a diff here is a real regression, never regenerate to
  "fix" it):**
  - `task fixpoint` → FIXPOINT OK (stage1==stage2; see the crux above).
  - The corpus **behavioral** tier and the interp/check tiers (the repo's corpus
    tiers are transpile / behavioral / check — "interp" denotes the
    behavioral/execution tier; there is no separately-named interp target).
  - The full `task check` AFTER goldens have been regenerated — i.e. once the
    EXPECTED-to-change fixtures are deliberately updated, the whole `go test ./...`
    suite + `go vet` must be green again with no remaining diffs.

### Procedure: regenerate and review goldens
- **Kind:** procedure
- **Regenerate** (only after a deliberate, behavior-preserving emitted-Go change):
  - Corpus exact-tier goldens (`*.go.expected`):
    `go test ./internal/corpus -run TestUpdateGoldens -update-goldens`
    (rewrites every exact-tier golden from the AST backend; a no-op/skip without
    the flag — see `internal/corpus/update_goldens_test.go`).
  - Parser AST snapshots: `go test ./internal/parser -update-snapshots`
    (see `internal/parser/snapshot_test.go`).
  - The go/ast-mirror unit tests in `internal/ast` are hand-updated to the sealed
    shape when SEAM-004 seals the category interfaces (no update flag).
- **Reviewer checklist** — before accepting a regenerated golden, confirm:
  1. The diff is **gofmt-stable** (both sides normalize; no stray formatting
     churn masking a real change).
  2. The change is **behavior-preserving** — same control flow / same values, only
     the idiom moved (e.g. a plain `switch` became a `match` lowering, an iota int
     became the §8.1 sealed-interface enum encoding). No new/removed cases, no
     altered constants with numeric identity.
  3. `task fixpoint` is still **FIXPOINT OK** after the change.
  4. The corpus **behavioral** tier is still green after the change.
  5. The regen was scoped to the seam being landed — no unrelated golden moved.
  A golden diff that fails any check is a regression, not a regeneration: fix the
  source, do not check in the diff.

### Verification (this story is documentation + procedure only)
- **Kind:** verification
- No `.goal`/`.go` source idiom change in SEAM-001, so the relaxed gate is not yet
  exercised; it is only DEFINED here for SEAM-002..006.
- `task check` green (`go vet ./...` + full `go test ./...`); `task build` green
  (both binaries); `task fixpoint` → FIXPOINT OK — all unchanged from the
  US-013 baseline because no source moved.

## SEAM-CAP-2 — cross-.goal-package enum/sema-fact propagation during self-host build

- **Kind:** compiler capability
- **Problem:** SEAM-CAP lowered a cross-package enum `match`/construction only when the
  DEFINING package was available as generated `.go` (its fixture used a `.go` foreign
  package; `EnrichForeign`/`foreignDecls` read only `.go`). The real per-package
  `goal build ./selfhost` transpiles each package from sibling `.goal` SOURCE, so an enum
  defined in a sibling `.goal` package (e.g. `ast.FuncMod`) was invisible to consumers in
  other `.goal` packages: a `match` over it failed (`unsupported expression *ast.MatchExpr`)
  and bare construction `a.Mod.From` lowered VERBATIM instead of the §8.1
  `a.Mod(a.Mod_From{})`.
- **Fix (bounded — enrichForeign reads `.goal`):** `foreignDecls` now classifies the
  resolved dir's entries; when it holds NO non-test `.go` but some `.goal`, it delegates to
  a new `goalForeignDecls` that runs the goal front end (`parser.ParseFile` +
  `ResolvePackage`) over those `.goal` files and projects the package's EXPORTED enums into
  `info.Enums` keyed `alias.Enum` (the same `sema.Enum` shape the `.go` path reconstructs
  from the generated §8.1 sum encoding). The `.go` path is unchanged and takes precedence
  when both forms exist, so SEAM-CAP's behavior is preserved; this is strictly ADDITIVE (a
  `.goal`-only dir previously yielded nothing).
- **Construction lowering:** added `enumRef` (the construction-side counterpart of
  `matchQualifier`) resolving an enum key from a bare `Ident` (`Enum`) OR a package-qualified
  `SelectorExpr` (`pkg.Enum`). `selectorExpr`, `variantLit`, and `armBodyType` use it, so a
  bare cross-package variant `pkg.Enum.Variant` lowers to `pkg.Enum(pkg.Enum_Variant{})`.
  Match lowering already handled the qualified case (SEAM-CAP `matchQualifier`); it only
  needed the enum present in `info.Enums["pkg.Enum"]`, which the enrichment now supplies.
- **Scope:** enum facts only from `.goal` source (the keystone for SEAM-002/003/004 —
  FuncMod/ChanDir/Mode/Severity are tag-only). Struct/func/method foreign facts from sibling
  `.goal` source are deferred (the `.go` path covers them once a package is emitted).
  `qualifyForeignType` requalifies variant field types best-effort (moot for tag-only).
- **Applied in BOTH** `internal/` (live transpiler — `internal/sema/foreign.go`,
  `internal/backend/lower.go`+`emit.go`) AND `selfhost/` (`selfhost/sema/foreign.goal`,
  `selfhost/backend/lower.goal`+`emit.goal`), so the self-host stays consistent and the
  fixpoint holds.
- **Proof:** a 2-package fixture under `internal/backend/testdata/goalenum/` where the enum
  is DEFINED IN A SIBLING `.goal` PACKAGE (`mood/mood.goal`) and consumed via cross-package
  `match` + bare construction (`use/use.goal`). `crosspkg_goal_enum_test.go` transpiles BOTH
  packages per-package (real topology), asserts the §8.1 type-switch + construction form, and
  builds+runs them against a reference switch (identical behavior).
- **Gates:** `task check`, `task build`, `task fixpoint` → FIXPOINT OK; corpus behavioral
  tier unchanged (the fixture is additive).

## SEAM-002 — FuncMod & ChanDir iota → goal enum (tree-wide); token.Kind kept as iota

- **Kind:** seam (relaxed gate; see "Seam methodology" above)
- **What converted.** `selfhost/ast` `FuncMod` (FuncPlain/FuncFrom/FuncDerive) and
  `ChanDir` (SendRecv/SendOnly/RecvOnly) are now goal `enum`s (§8.1 sealed-interface
  encoding) instead of `type X int` + iota. Every cross- and same-package consumer
  converted to `match` in the same atomic change:
  - `selfhost/ast/ast.goal:170` (`FuncDecl.Pos`, same-package) — `d.Mod != FuncPlain` →
    `notPlain := match d.Mod {…}`.
  - `selfhost/sema/question.goal` (`plainResultFuncs`) — `fn.Mod != ast.FuncPlain` →
    a match-bound bool, with the `!ok` type-assert guard split out FIRST so the
    short-circuit that protected the nil deref is preserved.
  - `selfhost/sema/resolve.goal` — `d.Mod == ast.FuncFrom || …FuncDerive` → match-bound
    bool inside the (retained) tagless `switch {}`; the `switch x.Dir {…}` in
    `typeString` → value-position `match` (the former `default` body re-homed under the
    explicit `ast.ChanDir.SendRecv` arm).
  - `selfhost/sema/convert.goal` — `fd.Mod != ast.FuncDerive` → match-bound bool (`!ok`
    split out).
  - `selfhost/backend/emit.goal` — `funcDecl`'s control `switch d.Mod` → `isDerive :=
    match d.Mod {…}` + `if isDerive { e.deriveDecl(d); return }` (the unreachable
    `default: e.fail(…)` arm dropped); `chanType`'s `switch x.Dir` → statement-position
    `match` (former `default` → explicit `SendRecv` arm).
  - `selfhost/parser/parser.goal` — construction sites requalified to
    `ast.FuncMod.FuncFrom`/`…FuncDerive` and `ast.ChanDir.SendRecv`/`…RecvOnly`/`…SendOnly`.
- **Zero-value gap (real semantic, fixed not refused).** An enum's zero value is `nil`,
  not the first variant — unlike iota, where `FuncMod`'s zero was `FuncPlain` and
  `ChanDir`'s was `SendRecv`. `parseFuncDecl` previously relied on `&ast.FuncDecl{}`
  defaulting to `FuncPlain`; it now sets `Mod: ast.FuncMod.FuncPlain` explicitly
  (`parseModFuncDecl` overwrites it for from/derive). `ChanType`'s sole constructor
  already set `Dir` explicitly. This construction invariant (no site leaves Mod/Dir nil)
  is what keeps the 3-arm exhaustive `match` total at run time.
- **token.Kind KEPT as iota (documented refusal, AC-1 escape hatch).** `selfhost/token`
  `Kind` retains `type Kind int` + iota because it has a genuine NUMERIC-IDENTITY
  dependence that survives the seam relaxation, not a mere scope limit: it is used as an
  ARRAY INDEX (`kindNames[k]`), in RANGE PREDICATES over contiguous numbering
  (`literalBeg < k && k < literalEnd`, and the analogous operator/keyword ranges), and
  relies on dense ordered integers. A goal `enum` lowers to a boxed sealed interface, not
  an ordered int, so converting `Kind` would break those operations — a behavior change,
  not an idiom gain. AC-1 explicitly permits keeping a member as iota for exactly this
  reason. (This supersedes nothing for FuncMod/ChanDir, which had no such dependence and
  were correctly converted; it upholds the US-005/US-007 `Kind`-stays-iota call on its
  real, semantic ground.)
- **internal/ast stays Go-iota.** The bootstrap reference compiler's own AST
  (`internal/ast`) keeps Go iota for FuncMod/ChanDir — it is plain Go, need not mirror the
  self-hosted representation, and is not part of the fixpoint comparison. The only
  internal/ change is test relocation: the FuncMod Walk/Pos/Mod assertions moved from the
  port-gated `internal/ast/ast_test.go` into a new internal-only
  `internal/ast/funcmod_test.go` (NOT added to `internal/selfhost/port_test.go`'s
  BuildAndTest slice), so the shared `ast_test.go` is FuncMod/ChanDir-symbol-free and
  compiles against BOTH Go-iota internal/ast and the enum-transpiled selfhost/ast the port
  gate builds it against.
- **Equivalence proof (relaxed gate).** `task fixpoint` → FIXPOINT OK (goal-c-1 and
  goal-c-2 emit byte-identical Go for `./selfhost`, both agreeing on the new enum/match
  form); `task check` green (incl. the `internal/selfhost` behavioral port gates that
  transpile selfhost/ast as an enum and run the relocated `ast_test.go` against it, and
  `internal/corpus` `TestASTEngineWholeCorpusBehavioralGate`); `task build` green. The
  emitted Go for the converted sites moved (iota int → §8.1 sealed-interface, `switch` →
  type-switch); verified behavior-preserving (same control flow / values, only the idiom
  moved). No golden regeneration was needed — corpus/snapshot goldens test the unchanged
  internal/ Go compiler, not the selfhost tree.
- **Capabilities relied on:** SEAM-CAP (cross-package enum-match lowering) and SEAM-CAP-2
  (cross-`.goal`-package enum visibility + bare-construction lowering); both pre-landed.

## SEAM-003 — Mode & Severity iota → goal enum (sema + backend + typecheck consumers)
- **Kind:** decision (reverses the US-011 "Mode and Severity stay iota" SCOPE refusal)
- **Chose:** express `sema.Mode` (ModeNone/ModeResult/ModeResultClosed/ModeOption) and
  `sema.Severity` (Error/Warning) as goal `enum`s (§8.1 sealed-interface encoding), and
  convert every cross- and same-package consumer to `match` in one atomic change.
- **Total conversion (no member kept as iota).** Unlike `token.Kind` (SEAM-002), neither
  type carries numeric identity: no array index, range predicate, contiguous-numbering, or
  wire value. The US-011 refusal asserted `sema.Severity(x)` numeric conversions existed —
  they do **not** (verified by grep across the tree). So both convert cleanly.
- **Variant surface form.** Dataless variants are referenced `Enum.Variant`: same package
  `Mode.ModeResult` / `Severity.Error`; cross-package `sema.Mode.ModeResultClosed` /
  `sema.Severity.Warning`. Construction, comparison, and match all use this qualified form;
  the bare `ModeResult`/`Error` symbols no longer exist after lowering.
- **`==`/`!=`/`switch` → `match`.** Converted sites: `caller.Mode != ModeResult` and the
  switch-true Mode cases in question.goal; the `sig.Mode` switch in resolve.goal (split into
  two value-position `match`es for Arity/EndsInError, since a no-op `ModeNone` arm cannot be
  an empty `{}`); `isResultFunc` in mustuse.goal; `needsResultPrelude` and the
  `closedResultTE`/`unwrapClosed`/`resultMatch` guards plus `calleeMode` in backend; the
  `mustuse` guard in typecheck. `==` would in fact still compile (interface comparison), but
  the proven SEAM-002 idiom is `match` and the PRD mandates it.
- **Enum zero is nil — set explicitly.** The enum zero value is `nil`, not the first
  variant. `funcSig` and emit.goal construction already set Mode; the critical fix is
  `foreign.goal` (foreign sigs land in `info.FuncSignatures`, which `needsResultPrelude`
  matches over — an implicit zero would fault at run time) and `calleeMode` (a missing map
  key yields a zero FuncSig; it now returns `sema.Mode.ModeNone` explicitly). Guards over a
  possibly-zero `csig`/`sig` (when `!known`/`!ok`) use a nil-safe `_` rest-arm so the match
  never faults on nil; guards over an always-set value use full variant enumeration.
- **`Severity.String()` → free `SeverityLabel(s Severity)`.** A goal enum lowers to a
  sealed interface, and Go forbids declaring a method on an interface type, so the
  `(s Severity) String()` Stringer is replaced by a free `match`-based `SeverityLabel`
  function. The two render sites that relied on the Stringer via `%s` (analyze.goal Render,
  typecheck.goal Diagnostic.String) call `SeverityLabel`/`sema.SeverityLabel` explicitly.
- **internal/ test handling (port-gate compatibility).** `internal/sema` and
  `internal/typecheck` stay Go-iota (plain Go, not in the fixpoint diff), but their
  white-box test files are reused by the self-host port gate against the enum-transpiled
  selfhost packages. Resolution differs by type:
  - **Severity (has a production label):** add a mirrored `SeverityLabel(Severity) string`
    to `internal/sema`, and rewrite the shared tests' `x.Severity != Error` comparisons to
    `SeverityLabel(x.Severity) != "error"` — a form that compiles against BOTH the iota
    package and the enum-transpiled one, preserving full coverage with no relocation.
  - **Mode (no production string form):** relocate the four `FuncSig.Mode` assertions from
    the port-gated `resolve_test.go` into a new internal-only `internal/sema/mode_test.go`
    (NOT in the port-gate slice), leaving `resolve_test.go` Mode-symbol-free (it still
    checks T/E/Arity/EndsInError against selfhost). Mirrors the SEAM-002 funcmod_test.go
    relocation.
- **Equivalence proof (relaxed gate).** `task fixpoint` → FIXPOINT OK (goal-c-1/goal-c-2
  byte-identical on `./selfhost`, both agreeing on the new enum/match form); `task check`
  green (incl. the `internal/selfhost` behavioral port gates transpiling selfhost/{sema,
  backend,typecheck} as enums and running the rewritten/relocated tests, plus the corpus
  behavioral tier); `task build` green. No golden regeneration was needed — goldens test the
  unchanged internal/ Go compiler, not the selfhost tree.
- **Capabilities relied on:** SEAM-CAP (cross-package enum-match lowering) and SEAM-CAP-2
  (cross-`.goal`-package enum visibility + bare-construction lowering); both pre-landed.

## SEAM-005 — lift the fallible exported/interface API to Result/? with callers

> SEAM-005 of the **seam** PRD (`prd.json`): lift the genuinely-propagating
> fallible EXPORTED/INTERFACE API of the self-hosted compiler to goal `Result`/`?`
> together with cross-package callers and interface contracts, under the relaxed
> seam gate (emitted Go may change; re-proven by `task fixpoint` self-consistency +
> corpus behavioral tier). This REVERSES the **scope-blocked** half of the
> US-008..US-012 "cross-package API stays `(T, error)`" refusals, and — per the
> honesty requirement — KEEPS the half blocked by **semantics**, documented below.
> Result/Option already cross package boundaries (special-cased), so this story
> does NOT depend on the enum/sealed capability gaps.

### Converted (scope-blocked-only, pure propagation → Result/`?`)

#### `typecheck.Load` → `Result[*Package, error]`
- **Kind:** conversion (reverses the US-012 `Load` SCOPE refusal).
- **Did:** `Load(pkg *project.Package) (*Package, error)` →
  `Result[*Package, error]`. Each internal guard WRAPS context
  (`fmt.Errorf("transpile: %w", …)`, `…"parse goal source %s: %w"…`,
  `…"parse generated %s: %w"…`), so `?` (which propagates the error UNCHANGED)
  cannot apply — every guard stays statement-form
  `if err != nil { return Result.Err(fmt.Errorf(...)) }`; the final
  `return p, nil` becomes `return Result.Ok(p)`.
- **Why it now fits:** the US-012 refusal was SCOPE — `Load` is oracle-pinned and
  has an in-tree caller (`GoTypesChecker.Check`). Under the seam gate the oracle is
  fixpoint + corpus, not byte-pinned signatures, and the caller IS in scope.
  Converting `Load` to Result is what lets `Check` propagate it with `?` (the real
  idiom gain). Open-E Result lowers to native `(*Package, error)`, so the emitted
  Go signature is unchanged (verified:
  `func Load(pkg *project.Package) (ok *Package, err1 error)`).

#### `TypeChecker.Check` + `GoTypesChecker.Check` → `Result[[]Diagnostic, error]`
- **Kind:** conversion (reverses the US-012 `GoTypesChecker.Check` SCOPE refusal).
- **Did:** the `TypeChecker` interface method and its `GoTypesChecker` impl
  (checker.goal) → `Result[[]Diagnostic, error]`. The body's pure propagation
  `p, err := Load(pkg); if err != nil { return nil, err }` becomes the idiomatic
  `p := Load(pkg)?`; success returns `Result.Ok(diags)`.
- **Why it now fits:** the US-012 refusal was that the interface contract + oracle
  pin (`var _ TypeChecker = GoTypesChecker{}`, two-value `tc.Check`) put the
  callers out of a single-package scope. Under the seam the contract is lifted in
  LOCKSTEP (interface method + impl together) and the emitted Go is byte-identical:
  an open-E Result interface method lowers to `Check(pkg) ([]Diagnostic, error)`,
  and the impl lowers to `func (GoTypesChecker) Check(...) (ok []Diagnostic, err
  error) { p, err := Load(pkg); if err != nil { return ok, err } … return diags,
  nil }`. So the Go oracle test (`internal/typecheck/checker_test.go`:
  `var _ TypeChecker = GoTypesChecker{}` + `got, err := tc.Check(pkg)`) stays valid
  against BOTH the internal Go bootstrap AND the transpiled selfhost Go — the seam
  stays swappable with no test edit. That IS the lockstep.

#### `sema.AnalyzePackageInDir` → `Result[[][]Diagnostic, error]`
- **Kind:** conversion (reverses the US-009 `AnalyzePackageInDir` SCOPE refusal).
- **Did:** `AnalyzePackageInDir(srcs, dir) ([][]Diagnostic, error)` →
  `Result[[][]Diagnostic, error]`. It delegates to the 3-value
  `AnalyzePackageInDirWith` (kept — see below), so the unpack is explicit:
  `diags, _, err := AnalyzePackageInDirWith(...); if err != nil { return
  Result.Err(err) }; return Result.Ok(diags)`.
- **Why it now fits:** US-009 refused it as SCOPE (exported, pinned by
  `package_test.go`). It carries a SINGLE success value (`[][]Diagnostic`) + error,
  so it is a clean Result. The relaxed gate allows the emitted Go to differ from
  the prior `return diags, err`; behavior is preserved because
  `AnalyzePackageInDirWith` returns `(nil, nil, err)` on its only error path, so
  `return diags, err` and `if err != nil { return nil, err }; return diags, nil`
  are behaviorally identical. The oracle `internal/sema/package_test.go`
  (`out, err := AnalyzePackageInDir(...)`) compiles against the lowered
  `(ok [][]Diagnostic, err1 error)` and stays green.

### Mirror & gate notes
- **`internal/` stays the lowered `(T, error)` mirror — NO production edit.** Go
  cannot express `Result`/`?`; `internal/{typecheck,sema}` are the bootstrap Go
  compiler and already ARE the lowered form of the converted selfhost source. The
  behavioral mirror holds automatically.
- **No oracle-test edits.** Because every conversion lowers to the SAME emitted Go
  signature, the port-gated white-box tests keep their two-value call sites and the
  `var _ TypeChecker = GoTypesChecker{}` assertion; they compile against both
  compilers unchanged.

### KEPT — genuine SEMANTIC non-fits (NOT scope; carved out, documented)
These survive the seam relaxation because cross-package reach does not change their
shape — each would need a struct bundle or a contract break, neither a real idiom
gain. (Confirms / refines the US-008..US-012 refusals.)

- **`parser.ParseFile (*ast.File, error)` — value-AND-error / partial result.**
  ParseFile returns a NON-NIL partial `*ast.File` AND joined errors SIMULTANEOUSLY
  (the parser is accumulator-style: helpers append to `p.errs` and never `error`).
  Two callers consume the partial AST on error: `project.PackageClause`
  (`file, _ := parser.ParseFile(src)` — its comment: "Reading the name even when
  later body parsing failed keeps the old lexer's tolerance") and
  `pipeline.declSites` (`file, _ := parser.ParseFile(src); if file == nil …`). A
  `Result[*ast.File, error]` Err arm carries NO file, so converting would DROP the
  partial AST = a behavior change (not behavior-preserving). This is the same class
  as the multi-value carve-outs, not a cross-package-caller scope problem. KEEP
  `(T, error)`. (The pure-propagation callers — `sema.Analyze`, `backend`,
  `AnalyzePackageInDirWith` — already consume ParseFile via `?`/2-value against the
  `(T, error)` form; ParseFile being `(T, error)` is exactly the open-E shape `?`
  expects, so no caller is harmed by keeping it.)
- **`sema.EnrichForeign (…) []error` — error ACCUMULATOR.** Appends per-import
  errors and `continue`s; returns the COLLECTED list, not a single propagated
  error. `?` cannot propagate a list. KEEP `[]error`.
- **`sema.AnalyzePackageInDirWith (…) ([][]Diagnostic, []error, error)` —
  MULTI-VALUE (3).** The per-import error slice is a SECOND meaningful value
  alongside success; a Result holds one. KEEP 3-value (a struct bundle here is
  ceremony, not idiom — `AnalyzePackageInDir` is the single-value Result wrapper).
- **`sema.foreignDecls` / `sema.goalForeignDecls (…, err error)` — MULTI-VALUE
  (4).** Three maps (structs/funcs+methods/enums) + error; a Result holds one, and
  the sole caller `EnrichForeign` ACCUMULATES the error rather than propagating it,
  so there is no `?` site even bundled. KEEP.
- **`sema.moduleResolve (string, bool)`, `sema.readModulePath (string, bool)`,
  `sema.constIntLit (int64, bool)` — comma-ok control flow.** Each collapses two
  distinct failures onto a single bool the in-file caller branches on
  (`if v, ok := f(…); ok`); the inner `err != nil` is deliberately swallowed.
  `?` has nothing to propagate; `Option` would change the `(T, bool)` contract.
  KEEP `(T, bool)`.

### Equivalence proof (relaxed gate)
- `task fixpoint` → **FIXPOINT OK** (goal-c-1/goal-c-2 byte-identical on
  `./selfhost`, both stages agreeing on the new Result/`?` source).
- `task check` green — incl. the `internal/selfhost` port gate transpiling
  `selfhost/{typecheck,sema}` and running the unchanged oracle tests
  (`checker_test.go`, `package_test.go`) against the lowered Go, plus the corpus
  behavioral tier; `internal/{typecheck,sema}` green.
- `task build` green. No golden regeneration needed — goldens test the unchanged
  internal/ Go compiler, not the selfhost tree; the emitted selfhost Go signatures
  are byte-identical (verified in `_bootstrap/fb/selfhost/...`).

## SEAM-CAP-3b — type-pattern match over a same-package sealed interface

CAP-3 part 2 of 3: `match` now lowers a SEALED-INTERFACE scrutinee (same package)
to a Go type-switch with concrete `case *T:` labels — the language feature that lets
`switch x := n.(type)` become an exhaustive `match`. CAP-3a (commit d79ebee) made
sealed interfaces preserve method signatures; this story adds the match capability.
Cross-package implementor propagation is CAP-3c (next), explicitly out of scope.

Four gaps closed, mirrored in BOTH internal/ (live Go transpiler) and selfhost/
(.goal mirror compiled by the port gate):

1. Implementor registry. `sema.Info` gains `SealedImpls map[string][]string`
   (interface name → implementor concrete types, stored as `*T`), built in
   resolve.go `resolveTypeDecl` from `StructType.Implements` clauses and unioned
   (deduped) across a package's files in `Info.Merge`. DECISION: keep the existing
   `Sealed map[string]bool` (sealedness) rather than fold it into the registry.
   Rationale — `implements` clauses also target ORDINARY (non-sealed) interfaces,
   whose feature-07 satisfaction `CheckImplements` verifies by short-circuiting on
   `Sealed[iface]`; a single presence-keyed `map[string][]string` would make an
   ordinary `implements` target look sealed and SKIP that check (a corpus
   regression). Sealedness and has-implementors are distinct facts, so two fields.
   `SealedImpls` holds the implements relation for ALL interfaces; it is consulted
   only for keys that are also in `Sealed` (the reverse-lookup `sealedInterfaceOf`),
   so non-sealed entries are inert.

2. Parser. New `ast.TypePattern{Type, Lparen, Binding, Rparen}` (distinct from
   `VariantPattern`/`RestPattern`). `parsePattern` routes a `*`-leading arm to
   `parseTypePattern` (`*T` or `*T(binding)`); a bare-ident arm stays a variant
   pattern, so enum/Result/Option matches are unchanged. A NEW parser field
   `armBody` makes a value-position arm body stop its binary expression at a
   newline-leading operator — Go's implicit-semicolon rule never starts a
   continuation with an infix operator, so a newline-leading `*` opens the next
   `*T` arm instead of being absorbed as `body * T` (the one genuine grammar
   ambiguity type patterns introduce). Targeted to arm bodies to keep the fixpoint
   safe.

3. Backend. New `sealedMatch`/`emitSealedArm` lowering (lower.go `isSealedMatch`;
   emit.go), DISTINCT from `enumMatch`: it emits `switch [guard :=] subj.(type)`
   with `case *T:` per type-pattern arm (rendered via the type expr), a `_`-rest →
   `default`, else a panicking default; the narrowed binding is renamed to the guard
   (no enum field-set machinery — the narrowed value IS the binding). Dispatched
   from matchStmt / returnStmt / tryVarMatch / tryAssignMatch / matchValue ahead of
   the matchQualifier checks.

4. Sema exhaustiveness. `checkOneMatch` routes a type-pattern match to
   `checkOneSealedMatch`, which resolves the sealed interface from the registry and
   requires every implementor covered or a `_` rest-arm — else a
   `non-exhaustive-match` Error (consistent with the §9 switch-coexistence rule). An
   unresolvable sealed type (e.g. cross-package — CAP-3c) defers with a new
   `unresolved-match-sealed` Warning (added to the guide catalog + AI-KNOWLEDGE
   bootstrap golden) rather than a false rejection.

Proof: internal/backend/sealed_match_test.go (lowering shape + behavioral build/run
identical to the reference `switch x := n.(type)`); internal/sema/sealed_match_test.go
(exhaustive ok, non-exhaustive Error naming the missing `*T`, `_`-rest ok, unresolved
defers); internal/ast/ast_test.go (TypePattern node identity + walk).

Gates: `task check`, `task build`, `task fixpoint` (FIXPOINT OK — stage1==stage2 on
the new parser/sema/backend source) all green; corpus behavioral tier unchanged
(the feature is additive — no existing match shape changes).

## SEAM-CAP-3c — cross-.goal-package sealed-interface match

CAP-3 part 3 of 3, and the final prerequisite for SEAM-004: a `sealed interface`
DEFINED in a sibling `.goal` package is now matchable (type-pattern `match`) from a
consumer `.goal` package during the real per-package `goal build ./selfhost`
bootstrap. CAP-3b made same-package sealed match work; this story carries the
implementor set across the package boundary. It matters because 35 of SEAM-004's 36
type-switches consume ast.Node/Expr/Stmt/Decl from packages OTHER than selfhost/ast.

The gap: foreign enrichment propagated ENUM facts from sibling `.goal` source (the
SEAM-CAP-2 `goalForeignDecls` path: `parser.ParseFile` + `ResolvePackage`, projecting
exported enums into `info.Enums`) but NOT sealed-interface implementor sets. So a
consumer's `checkOneSealedMatch` → `sealedInterfaceOf` found nothing cross-package and
deferred with the `unresolved-match-sealed` Warning CAP-3b explicitly left open.

The fix (mirrored line-for-line in `internal/sema/foreign.go` and
`selfhost/sema/foreign.goal`):

1. `goalForeignDecls` now also projects EXPORTED sealed interfaces. `ResolvePackage`
   already builds `info.Sealed` (sealed iface names) and `info.SealedImpls`
   (iface → `*T` implementors) from the defining package's `sealed interface` /
   `implements` clauses. For each exported iface it requalifies the interface name and
   each implementor by the import alias via the existing `qualifyForeignType`
   (`*Lit` → `*shape.Lit`), exactly as `typeString` renders a `*shape.Lit` type
   pattern, so the registry keys align by string. Returned as a new
   `sealed map[string][]string` (the 6th `foreignDecls`/`goalForeignDecls` result).

2. `EnrichForeign` merges that map into the consumer's `info.Sealed[iface] = true` and
   `info.SealedImpls[iface] = impls`. `checkOneSealedMatch`/`sealedInterfaceOf` then
   resolve the foreign sealed interface and check exhaustiveness across the boundary —
   no code change needed there.

The `.go` foreign path returns nil sealed: the real bootstrap resolves a sibling to its
`.goal` SOURCE, so the goal-source path is the one exercised; reconstructing sealed sets
from an already-generated `.go` sibling is deferred, exactly as struct/func/method facts
are from `.goal` source (the SEAM-CAP-2 precedent). The backend `sealedMatch` lowering is
dispatched purely by pattern SHAPE (`isSealedMatch` — any TypePattern arm) and renders
`case *shape.T:` directly from the pattern, so cross-package match ALREADY lowered before
this story; CAP-3c closes only the sema resolution/exhaustiveness gap.

Proof: internal/sema/crosspkg_sealed_test.go (EnrichForeign projects shape.Node sealed +
`*shape.Lit`/`*shape.Neg`; a complete cross-package match is clean, a non-exhaustive one
is a `non-exhaustive-match` Error naming `*shape.Neg`) over a sibling-.goal fixture
(internal/sema/testdata/sealedshape/shape.goal); internal/backend/crosspkg_sealed_test.go
(both the defining `.goal` package and the consumer transpiled per-package, built into a
throwaway `module goal`, and run against a reference `switch x := n.(type)` — behaviorally
identical) over internal/backend/testdata/goalsealed/{shape,use}.

Gates: `task check`, `task build`, `task fixpoint` (FIXPOINT OK — stage1==stage2 on the
new sema source) all green; corpus behavioral tier unchanged (additive fixtures).

## SEAM-CAP-3d — nested sealed-interface hierarchies (one sealed interface embeds another)

CAP-3 part 4, and the FINAL prerequisite for SEAM-004: a concrete type may now be an
implementor of BOTH a sealed category interface AND a sealed interface it embeds. This is
the 2-level AST topology SEAM-004 needs — `ast.Expr`/`Stmt`/`Decl`/`Spec` all embed
`ast.Node`. CAP-3a/b/c only proved FLAT single-level sealed interfaces.

The gap: a concrete type carries exactly one `implements` clause → one marker. When sealed
`Expr` embeds sealed `Node`, a type declared `implements Expr` emitted only `isExpr()`, so
the emitted Go failed `go build` ("`*T` does not implement Expr (missing method isNode)"
and "impossible type switch case … missing method isNode"), and `Node`'s implementor set
(used for exhaustiveness) was incomplete.

Design choice: **embedding cascade (option b), NOT a multi-interface clause.** A
multi-clause `implements Expr, Node` would need new parser grammar and would re-state by
hand what Go's interface embedding already expresses. The cascade mirrors Go's own
embedding: declaring `implements Expr` where sealed `Expr` embeds sealed `Node`
automatically registers the type under both interfaces and emits both markers. No parser
change.

The fix (mirrored line-for-line in `internal/` and `selfhost/`):

1. **sema** (`resolve.go` / `resolve.goal`): new `cascadeSealedImpls()` — for each sealed
   iface B with implementors, propagate them to every sealed interface B transitively
   embeds (walking the existing `info.EmbeddedIfaces`, which is already populated for sealed
   interfaces because their body shares `resolveInterfaceMethods` with ordinary
   interfaces), gated on `info.Sealed`. Run at the tail of `Resolve(f)` (single-file path
   used by `backend.Transpile`) and again at the tail of `ResolvePackage` after the merge
   loop (cross-file hierarchies). Idempotent — `addImplementor` dedups. Helper
   `sealedEmbedClosure`/`collectSealedEmbeds` (plain recursion, no closure, for the
   self-host mirror).

2. **backend** (`lower.go`+`emit.go` / `.goal`): new `sealedEmbeds(info, iface)` (transitive
   embedded-sealed set, source order, deduped). `implementsMarker` now emits the sealed
   marker for `iface` PLUS a marker per embedded sealed interface, so a `implements Expr`
   type emits both `func (T) isExpr() {}` and `func (T) isNode() {}`.

Cross-package needs NO change: `goalForeignDecls` (CAP-3c) projects `info.Sealed` +
`info.SealedImpls` from the defining package's `ResolvePackage`, which now runs the cascade
before projection — the foreign implementor sets arrive already cascaded. (SEAM-004's AST
is a single package anyway.)

Fixpoint safety: the selfhost source contains no nested sealed hierarchy YET (SEAM-004
introduces it), so emitted Go is byte-identical for every existing flat sealed interface
(the cascade adds nothing; `sealedEmbeds` is empty). The map-iteration order in the cascade
does not reach emitted Go — `SealedImpls` order is sema metadata only; marker order follows
source-ordered `EmbeddedIfaces`.

Proof: internal/backend/nested_sealed_test.go (sealed `Node`; sealed `Expr` embeds `Node`;
`Num`/`Neg` implement `Expr`, `Comment` implements `Node` directly — asserts both cascaded
markers are emitted and `Comment` does NOT get `isExpr`, then builds into a throwaway
`module goal` and runs `match` over `Expr` AND over `Node` against a reference
`switch x := n.(type)`); internal/sema/nested_sealed_test.go (cascade registers `*Num`/`*Neg`
under both `Expr` and `Node` while `*Comment` stays Node-only; exhaustiveness enforced at
both levels — a match over `Expr` omitting `*Neg` and a match over `Node` omitting the
cascaded `*Neg` are each `non-exhaustive-match` Errors).

Gates: `task check`, `task build`, `task fixpoint` (FIXPOINT OK) all green; corpus
behavioral tier unchanged (additive tests only).

## SEAM-004 — seal ast.Node/Expr/Stmt/Decl/Spec + convert AST type-switches to match
- **Kind:** seam (reverses the US-007 AST-seal refusal and the Walk refusal)

The AST category interfaces in `selfhost/ast` are now sealed:

    sealed interface Node { Pos() token.Pos; End() token.Pos }
    sealed interface Decl { Node }   // and Stmt / Expr / Spec, each embedding Node

Every concrete node type declares `implements <Category>` (e.g. `type Ident struct
implements Expr { … }`); the SEAM-CAP-3d embedding cascade emits BOTH the category
marker (`isExpr()`) and the embedded `isNode()` from that single clause. The
Node-only support types (Field, FieldList, File, DocComment, Variant, PayloadField,
MatchArm, ImplementsClause) declare `implements Node` directly. The old unexported
marker methods (declNode/stmtNode/exprNode/specNode) are deleted — the seal
generates the markers.

**Type-switches converted to `match`.** Every type-switch over a single sealed AST
interface (scrutinee Node/Expr/Stmt/Decl/Spec) is now a goal `match` with type
patterns and exhaustiveness: in `selfhost/sema` (check, resolve, question, fields,
mustuse, assert), `selfhost/backend` (lower, emit), and `selfhost/parser` (parser,
goal_construct). Partial switches carry a `_` rest-arm (the §8.2 opt-out);
value-returning switches stay statement-position matches with `return` inside arm
blocks; a `case nil:` is hoisted to an `if x == nil` guard before the match (a
sealed match dispatches on concrete `*T`, so the nil-interface case has no pattern).

**Three switches deliberately kept as plain type-switches** (recorded here so the
non-conversion is a decision, not an oversight):
1. `selfhost/ast/walk.goal` Walk — the AC permits a justified exclusion. It is the
   one switch enumerating the ENTIRE Node implementor set and uses two Go
   type-switch features `match` lacks: grouped multi-type clauses
   (`case *Ident, *BasicLit:`) and no-op clauses. Expanding to ~60 single-type arms
   (several empty) is strictly less readable with no idiom gain; the visitor is
   exhaustive by construction.
2. `selfhost/backend/emit.goal` armBody — its cases discriminate across the SIBLING
   sealed interfaces `ast.Stmt` and `ast.Expr` (plus a nil arm): one value tested
   against two DIFFERENT sealed-category interfaces, not against the concrete `*T`
   variants of ONE. `match` lowers type patterns as concrete `case *T:` labels and
   cannot express interface-typed `case ast.Stmt:` clauses.
3. `selfhost/sema/foreign.goal` ×3 — these switch over `go/ast` (Go stdlib) nodes
   read from imported `.go` packages. go/ast's interfaces are not goal-sealed and
   are unsealable from here, so they stay plain Go type-switches.

**The US-003 go/ast-mirror oracle is RESOLVED by retaining the mirror unchanged.**
`selfhost/ast` was modeled byte-for-byte on `go/ast`, with the ported `internal/ast`
unit tests as the oracle. Sealing changes the EMITTED Go shape (markers move from
`exprNode()` to `isExpr()`/`isNode()`), but the public API the mirror tests exercise
— struct fields, `Pos()`/`End()`, node construction, and assignability to the
category interfaces — is unchanged. The port-gated `internal/ast/ast_test.go`
references no marker method (verified), so it compiles and passes against BOTH the
open `internal/ast` (the live Go compiler's own AST, left open) and the sealed
`selfhost/ast` transpile. No mirror test was changed or deleted; the seal simply
does not disturb the behavior the oracle pins. `internal/ast` stays open Go because
it is the bootstrap compiler's AST and is not in the fixpoint diff (same precedent
as SEAM-002 keeping internal/ast iota).

**A prerequisite nondeterminism bug was fixed** (required for a reliably-green
gate): `sema.sealedInterfaceOf` resolved a sealed match's interface from one arm's
concrete type by iterating the `SealedImpls` map and returning the first hit. Under
the now-real nested AST hierarchy a concrete type belongs to several sealed sets via
the cascade (every `*ast.Expr` is also registered under `ast.Node`), so map
iteration order made resolution nondeterministic — an exhaustive `match` over the
embedding interface could intermittently be checked against the embedded
super-interface and falsely flagged non-exhaustive (a ~50%-per-run flake in
`TestNestedMatchExhaustiveBothLevels`). The fix resolves to the MOST SPECIFIC sealed
interface (smallest implementor set, ties broken by name) whose set contains every
covered arm type — deterministic and semantically the narrowest level the match
targets. Mirrored line-for-line in `internal/sema/check.go` + `selfhost/sema/check.goal`.

Gates: `task check`, `task build`, `task fixpoint` (FIXPOINT OK) all green; corpus
behavioral + interp tiers unchanged. Equivalence re-proven under the relaxed seam
gate by fixpoint self-consistency + corpus behavioral tier (no golden regen was
needed — the goldens test the unchanged internal/ Go compiler, not selfhost/).

## SEAM-006 — cross-cutting proof: the compiler showcases goal end-to-end

> SEAM-006 of the **seam** PRD (`prd.json`): the closing proof story (mirror of
> US-013, but for the seam end state). Not a source idiom change — it VERIFIES
> and MEASURES that the SEAM-002..005 conversions hold together, then states the
> honest end state. Every number below is counted from the live `selfhost/` tree,
> not asserted.

### Gates (the equivalence oracle, all green)

- `task check` — green (go vet + full `go test ./...`, including the
  internal/selfhost port gates that transpile every `selfhost/*` package and run
  the ported oracle tests against the lowered Go, plus the corpus behavioral +
  interp + check tiers).
- `task build` — green (both binaries).
- `task fixpoint` — **FIXPOINT OK**: goal-c-1 and goal-c-2 emit byte-identical Go
  for the compiler's own (now idiomatic) source. `diff -r _bootstrap/fa
  _bootstrap/fb` is empty. This is the genuine differential self-host proof on
  the new enum/match/Result form — both stages agree on the idiomatic shape.

### Per-seam tally — converted vs documented semantic non-fit

| Seam | Converted | Kept (documented non-fit / out-of-scope, with reason) |
|------|-----------|--------------------------------------------------------|
| SEAM-002 (iota→enum) | `ast.FuncMod` (FuncPlain/FuncFrom/FuncDerive), `ast.ChanDir` (SendRecv/SendOnly/RecvOnly) + every cross-package consumer (`==`/`!=`/switch → `match`) | `token.Kind` kept iota — genuine numeric identity (`kindNames[k]` indexing, `literalBeg<k<literalEnd` range arithmetic, contiguous wire numbering) |
| SEAM-003 (iota→enum) | `sema.Mode` (ModeNone/ModeResult/ModeResultClosed/ModeOption), `sema.Severity` (Error/Warning) — TOTAL conversion, no member kept iota; `Severity.String()` → free `SeverityLabel` func (an enum lowers to a sealed interface; Go forbids a method on it) | none kept for these two types (neither had numeric identity; the US-011-claimed `sema.Severity(x)` numeric conversions did not actually exist) |
| SEAM-004 (seal AST + type-switch→match) | `ast.Node/Expr/Stmt/Decl/Spec` sealed (§8.1, 2-level hierarchy via CAP-3d cascade); **27 type-switches → exhaustive `match`** (134 type-pattern arms) across sema(check,resolve,question,fields,mustuse,assert), backend(lower,emit), parser(parser,goal_construct) | `ast/walk.goal` Walk (60-arm grouped/no-op, AC-permitted exclusion); `backend/emit.goal` armBody (switches over SIBLING category interfaces `ast.Stmt`/`ast.Expr`, not variants of one — match patterns are concrete `*T`); `sema/foreign.goal` ×3 (over the `go/ast` STDLIB, unsealable); `pipeline/sourcemap.goal` ×2 and `typecheck/{nozero,implements}.goal` ×2 (partial switches with a `default`/fall-through over sealed AST, outside SEAM-004's stated package scope of sema/backend/parser/ast) |
| SEAM-005 (fallible API→Result/?) | cross-package cluster lifted: `typecheck.Load` →Result, `TypeChecker.Check` interface + `GoTypesChecker.Check` impl (lockstep) →Result, `sema.AnalyzePackageInDir` →Result (callers use `?`). With the pre-seam per-package conversions the tree now has **7 Result-returning fallible APIs** and **56 `Result.Ok`/`Result.Err`/`?` sites** | genuine SEMANTIC non-fits (NOT scope): `parser.ParseFile` (partial AST + error simultaneously), `EnrichForeign` (`[]error` accumulator), `AnalyzePackageInDirWith`/`foreignDecls`/`goalForeignDecls` (multi-value), `moduleResolve`/`readModulePath`/`constIntLit` (comma-ok); command-boundary `run`/`emitPackage`/`Emit`/`Transpile`/`TranspilePackage` stay bare `error` (top-level `if err != nil` handlers) |

### Honest end state — idioms the compiler now showcases tree-wide

- **enum from iota:** the four closed, unordered classification types are goal
  `enum`s (FuncMod, ChanDir, Mode, Severity). The only `iota` left in `selfhost/`
  is `token.Kind` and `litClass` (numeric-identity types, documented) plus two
  private backend implementation helpers (`roKind`, `matchPos`) that are not part
  of the public idiom surface. The idiom is the default; the exceptions are named.
- **match over a sealed AST:** the compiler's central data structure
  (`ast.Node`/`Expr`/`Stmt`/`Decl`/`Spec`) is sealed and its dispatch is 27
  exhaustive `match` blocks. The remaining 9 plain type-switches are each a
  documented non-fit (stdlib `go/ast`, sibling-interface switch, the 60-arm Walk,
  or out-of-scope partial switches), not a silent gap.
- **Result/?:** the genuinely-propagating exported/interface API at the
  compiler's seams is Result-valued and threaded with `?`; the remaining
  `(T,error)` sites are accumulators, multi-value, comma-ok, partial-value, or
  the command boundary — each a real semantic non-fit, not a scope dodge.

### `goal fix` over the whole selfhost tree — the autofixer agrees the API is idiomatic

`goal fix` was run over all 39 `selfhost/**/*.goal` files. **It would auto-modify
ZERO of them** (verified by diffing each file against its `goal fix` output). It
makes no automatic propagation conversion anywhere — i.e. the propagating API is
already idiomatic by the autofixer's own judgement. Its reports are all
deliberate refusals + advisories, recorded honestly here:

- **12 `[result-sig]` skips** (refusals, no change): `Transpile`, `emitFile`,
  `elemConv`, `TranspilePackage`, `packageName`, `ParseFile`, `DefaultResolver`,
  `goListResolve` (non-propagating return); `AnalyzePackageInDirWith` (multiple
  non-error values); `Discover` (exported, callers fix can't see); `run` +
  `emitPackage` (bare `error` at the command boundary). Every one maps to a
  SEAM-005 documented non-fit or the command boundary.
- **14 `[call-site]` suggestions** (advisory only, not an auto-fix withheld):
  `Emit`, `Transpile`, `emitDoctests`, `TranspilePackage`, `run`, `Discover`,
  `constIntLit`, `EnrichForeign`, `foreignDecls`, `goalForeignDecls`,
  `moduleResolve`, `readModulePath`, `goListResolve`, `AnalyzePackageInDirWith` —
  each names a manual `if err != nil` whose enclosing fn is intentionally not
  Result-returning (multi-value / accumulator / comma-ok / command boundary).
  These are the exact carve-outs SEAM-005 documented; the autofixer cannot
  cross-file/cross-package coordinate to convert them and correctly does not try.

Net: no residual *auto-convertible* propagation. The reports are the autofixer
confirming the same boundary SEAM-005 drew by hand.

### Before / after — quantified shift (measured from the tree)

| Idiom | Before (transpiled-Go shape) | After (idiomatic goal, this PRD) |
|-------|------------------------------|----------------------------------|
| Closed classification types | 6 `type X int` + iota blocks | 4 are now `enum` (FuncMod, ChanDir, Mode, Severity); 2 kept iota for numeric identity (token.Kind, litClass) |
| Dispatch over the AST | ~36 plain `switch x := n.(type)` over OPEN interfaces | **27** are now exhaustive `match` over a **sealed** AST (**134** type-pattern arms); 9 remain as documented non-fits |
| Fallible seam API | `(T, error)` + manual `if err != nil` propagation everywhere | **7** Result-returning APIs, **56** `Result.Ok`/`Err`/`?` sites; remaining `(T,error)` are documented semantic non-fits |

### META-finding (the central result) — the deep idioms needed NEW compiler features

The honest, load-bearing conclusion of this PRD is not "we converted N
type-switches." It is that **goal's deep idioms were blocked not merely by
per-package audit scope but by missing language/compiler capabilities that did
not exist before this PRD.** The per-package audits (US-005..013) could not have
reached this end state at any scope, because the features they would have needed
weren't built. SEAM-006's proof rests on FOUR new compiler capabilities created
in this PRD:

1. **SEAM-CAP** — cross-package enum-match lowering in the backend (a `match`
   over an enum DEFINED in an imported package now lowers; previously only
   Result/Option crossed package lines, via hardcoded special-casing).
2. **SEAM-CAP-2** — cross-`.goal`-package enum/§8.1-fact propagation during the
   real per-package `goal build ./selfhost` bootstrap (a sibling-`.goal`-defined
   enum is now visible to consumers in other `.goal` packages; foreign
   enrichment reads `.goal` source, not just generated `.go`).
3. **SEAM-CAP-3a–d** — sealed-interface type-pattern `match`, built in four
   parts: (a) sealed interfaces preserve declared method signatures (so sealing
   `ast.Node` keeps `Pos()/End()`); (b) same-package type-pattern match lowering
   to a Go type-switch with exhaustiveness; (c) cross-`.goal`-package
   sealed-interface match (implementor sets propagated across the boundary); (d)
   nested sealed hierarchies (the 2-level AST: `Expr`/`Stmt`/`Decl`/`Spec` embed
   `Node`, with marker + registry cascade).

Before this PRD, none of these existed: a cross-package enum `match` errored with
`unsupported statement-position match`, and sealed-interface match did not exist
at all. The idiomatic self-host was therefore gated on building real language
features, and SEAM-006 is the proof that those features hold together
end-to-end — `task fixpoint` byte-identical on the idiomatic source, corpus
behavioral + interp + check tiers green.

---

## US-001 — self-host flip: adopted layout & trust model

> First story of the **self-host flip** PRD (`prd.json`). A pure decision/record
> story: it adopts and writes down the flip's foundational choices *before any
> compiler code moves*, so the loop and reviewers share one model. The flip's
> plan (`SELF-HOST-FLIP-PLAN.md`) lays out the alternatives; this entry pins the
> ones the PRD adopts. This **supersedes** the "coexist as peers" / "tooling out
> of scope" / permanent-Go-trust-root framing of `SELF-HOST-RESEARCH.md` §1/§5
> and `REWRITE-ARCHITECTURE.md` §7 (those docs are annotated to point here).

### Adopted layout: colocated `<file>.goal` + committed generated `<file>.go`
- **Kind:** decision
- **Chose:** the **colocated** variant — each package directory holds the canonical
  goal source `<file>.goal` *and* its committed generated Go `<file>.go` side by
  side (the `.proto`/`.pb.go` idiom). The goal front-end reads the `.goal`; the Go
  toolchain (`go build`) reads the `.go`. Source and generated therefore resolve at
  the **same import path** (`goal/internal/compiler/<pkg>`, later `goal/internal/<pkg>`).
- **Over:** the **split-dir** variant that `SELF-HOST-FLIP-PLAN.md` §3 sketched as
  its default — goal source under `internal/compiler/<pkg>/*.goal` and committed
  generated Go at a *different* path `internal/<pkg>/*.go`.
- **Why:** colocation keeps source and generated on one import path, so no module
  gymnastics, no `-C`/second-module trick, and no path-rewrite step between source
  and artifact: `go build ./internal/compiler/...` resolves the generated `.go`
  exactly where the `.goal` lives. Split-dir's cleaner directories were refused
  because the dual-path indirection (and the implied path rewrite on every
  generate) costs more than the slightly busier directory listing. Cost accepted:
  each package dir lists both extensions. **Revisit** only if the doubled listing
  becomes a real ergonomic problem.

### Adopted bootstrap: committed-generated Go (B-commit); corpus is the primary gate
- **Kind:** decision
- **Chose:** **B-commit** — transpile the goal source to Go and **commit that
  generated Go**. A clean `git clone && go build` compiles the committed Go →
  trusted stage-0 → re-transpiles the goal source → a drift gate verifies the
  regenerated Go is byte-identical to what is committed → `task fixpoint`. This is
  the standard self-hosting bootstrap (rustc stage0): reproducible, reviewable, no
  prebuilt binaries.
- **Over:** **B-binary** (ship a prebuilt stage-0 binary — opaque, platform-bound,
  not reviewable) and **B-emit-only** (commit no Go, regenerate on every build —
  chicken-and-egg, needs a bootstrap binary, i.e. B-binary). Both refused.
- **Why:** once the hand-written reference Go transpiler is deleted, the line-by-line
  Go differential oracle is gone, so correctness can no longer lean on "diff against
  the reference Go." From that point the **corpus behavioral conformance tier is the
  primary correctness gate** (the goal-built compiler must pass the same corpus the
  Go compiler did), alongside the `task fixpoint` byte-identity check. B-commit keeps
  a clean checkout buildable with only the Go toolchain while making the generated Go
  a reviewable, drift-gated artifact of the goal source — not a hand-maintained peer.

### "Self-hosted" = the shipped goal+goalc library closure; test/dev infra stays Go
- **Kind:** decision
- **Chose:** define **self-hosted** as *the shipped `goal` + `goalc` library closure
  is written in goal* — the compiler proper (token, lexer, ast, parser, sema,
  project, pipeline, backend, typecheck) plus the shipped tooling the binaries link
  (goalfmt, textedit, cap, guide, fix, interp, lsp). Explicitly **staying Go** (test
  and dev infrastructure, not part of any shipped binary): the `corpus` harness,
  `byexample`, the `internal/selfhost` bootstrap/port harness, `cmd/corpus-gen`, and
  `cmd/build-playground`.
- **Over:** a maximalist "every `.go` file in the repo becomes `.goal`" reading.
- **Why:** the goal of the flip is that the *shipped toolchain* dogfoods goal end to
  end; the Go-only test/dev harnesses exist to *verify* that toolchain and gain
  nothing from being ported (some, like `internal/selfhost`, are the very machinery
  that transpiles and proves the goal source). Scoping "self-hosted" to the shipped
  closure keeps the flip's must-port set bounded and honest. **Revisit** only if a
  later decision wants the harnesses ported too.

