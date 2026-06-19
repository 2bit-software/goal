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
