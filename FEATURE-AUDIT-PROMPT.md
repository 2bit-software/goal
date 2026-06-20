# Feature Audit Loop — one feature per iteration

You are auditing and pinning down **one** feature of **goal** (the correctness-oriented Go
dialect formerly called "Go+"). The full design rationale lives in `goal-design-spec.md`; the
work queue lives in `TODO.md`. This prompt runs in a loop — **do exactly one feature per
iteration, then stop.**

Do not redesign the language. The spec already made the hard calls (closed sum-type spine,
conventional names, familiarity budget). Your job is to take a feature from "described in prose"
to **(1) nailed-down syntax, (2) concrete Go it transpiles to, (3) a runnable reference
transpiler** — and nothing more.

---

## Step 0 — Pick the feature

1. Read `TODO.md`. Find the **first** feature whose checkbox is unchecked (`- [ ]`).
2. If every feature is checked, report "All features audited" and **stop** — do not invent work.
3. Respect **dependencies** listed on that feature. If a dependency is unchecked, something is
   wrong with the order — flag it and stop rather than guessing.
4. Read the spec sections cited on that TODO item (and §8 for the codegen contract). Read the
   prior feature's `features/*/` artifacts if this feature builds on them (e.g. `match` reuses
   the enum encoding).

Everything below applies to **that one feature**.

---

## Step 1 — Nail down the syntax (Deliverable A) — **user-driven, ask first**

The spec calls syntax "the cheapest thing to change" and most samples are marked *illustrative,
not final*. **Syntax is the user's call, not yours.** Do not silently pick a form and write it
down. Instead, for **every** syntax decision this feature owns, present concrete candidates and
let the user choose via the **`AskUserQuestion` tool**.

> **Where this step runs.** `AskUserQuestion` only works in an interactive turn — it cannot run
> inside a detached/background sub-agent. So the syntax-selection part of an iteration must happen
> in the **main loop** (or a foreground sub-agent that can surface questions back to you). A
> sub-agent may *prepare* the candidate set, but the actual `AskUserQuestion` call and the user's
> choice happen in the interactive thread. Do not auto-decide syntax just because you're in an
> agent context — stop and ask.

For each decision point (e.g. for `match`: the arm separator, statement-vs-expression form, the
rest-arm spelling; for enums: the variant-payload syntax, the construction form):

1. **Offer 2–4 distinct candidates** via `AskUserQuestion`. Make them genuinely different idioms,
   not cosmetic variants — e.g. a Rust-shaped form, a Swift/Scala-shaped form, a Go-shaped form.
2. **Use the `preview` field** on each option to show a **real code mockup** of that syntax on
   this feature's worked example, so the user compares actual `goal` snippets side by side.
3. In each option's `description`, name the **widely-seen idiom** it lands on and the **tradeoff**
   (familiarity spent vs. error class caught), per the two principles. Put your recommendation
   first and label it `(Recommended)`.
4. Batch related decisions into one `AskUserQuestion` call (up to 4 questions) rather than many
   round-trips, but keep each question to a single decision.
5. Only after the user has chosen do you write `SYNTAX.md`. Record **the chosen option and the
   alternatives that were rejected** (so the decision is auditable). If the user picks "Other"
   and supplies their own, that wins.

Then write `features/<NN-name>/SYNTAX.md` containing:

- **Final surface syntax.** Pick it. Commit to it. One form, not a menu.
- **Grammar.** An EBNF (or clear BNF-ish) fragment for the construct — enough that a parser
  author has no ambiguity. Include how it nests inside existing Go-shaped grammar.
- **Worked examples.** At least 3, covering the common case and the awkward edges (e.g. for
  `match`: expression-position vs statement-position; payload binding; the `_` rest-arm).
- **Rationale, tied to the two principles.** For every divergence from Go-shape, state which
  *widely-seen idiom* (Rust/Swift/Scala/OCaml/TS) it lands on and why, per the **familiarity
  principle**. For every check it enables, name the **silent error class** it converts to a
  located error, per the **feedback principle**. If a choice spends "familiarity budget" (§7),
  justify the spend.
- **Resolved open questions.** If this feature has entries in §9 (e.g. switch-coexistence rule,
  explicit-defaults syntax, `From`-conversion shape), route the **syntax-bearing** ones through
  the same `AskUserQuestion` flow above, then record the decision and one-line reasoning here.
  Leave nothing as "TBD" that this feature owns.
- **Conventional names are non-negotiable.** `Some`/`None`, `Ok`/`Err`, `?`, `=>`, `_` stay as-is
  — do not Go-ify them (§7).

---

## Step 2 — Specify what it transpiles to (Deliverable B)

The target is **idiomatic Go** that a Go developer would recognize. §8 is the governing
contract — follow it; if you must deviate, say why in the doc.

Write `features/<NN-name>/TRANSPILE.md` containing:

- **Input → output pairs.** For each worked example in `SYNTAX.md`, show the exact Go it lowers
  to. Side by side.
- **Lowering rules.** The general algorithm, not just the examples: how payloads map to fields,
  how bindings rewrite to field accesses, what the marker/encoding is, etc.
- **Erasure vs preservation (§8.0).** State explicitly which parts are **erased** (static
  guarantees: exhaustiveness, must-use, field-completeness, `implements`) and which are
  **runtime-preserved** (match value, `Result`/`?` control flow, `Option` branching, `assert`).
  Where the checker proves unreachability, emit the **defensive `panic("unreachable: ...")`**
  per the erasure-with-defensive-panic rule — never silent fall-through.
- **Strategy forks.** Where a feature has more than one lowering (e.g. open-`E` native tuple vs
  closed-`E` sum encoding; `Option` pointer vs sum; immediate vs stored per §8.7), document each
  and state the rule that selects between them. For v1, handling the **immediate/consumed** case
  is required; note where **stored-as-value** must fall back to the sum encoding.
- **Hygiene.** All generated temporaries use the `__gop_` prefix (§8).

---

## Step 3 — Build a reference transpiler (Deliverable C)

Decided setup (do not re-litigate): **Go**, **per-feature standalone**. Each feature gets its
own self-contained Go program in `features/<NN-name>/transpiler/` — its own `go.mod`, no
dependency on other features' code. Copy/duplicate small shared helpers rather than building
shared infra; a unified front-end is a later, separate effort.

**Crucial constraint: NO error checking yet.** Assume the input is already well-formed and
type-correct. Do **not** implement exhaustiveness checking, must-use tracking, field-completeness
validation, or `implements` verification. Those are the checker's job and
come later. The reference transpiler's only job is: **valid `goal` source for this feature → the
correct Go from Deliverable B.** If input is malformed, undefined behavior is acceptable.

The transpiler must:

- **Be runnable.** `go run` / `go test` works from within `transpiler/`. It builds.
- **Parse only what it needs.** You do not need a full Go grammar. A focused parser (or even a
  scoped recognizer that handles this construct and passes the rest through) is fine for a
  reference. Prefer producing real Go via `go/ast` + `go/printer` or `go/format` so the output is
  gofmt-clean; string templating is acceptable if you `go/format` the result.
- **Emit the exact Go from Deliverable B** for every example.

Also create `features/<NN-name>/examples/` with paired files: `*.goal` inputs and
`*.go.expected` outputs (one pair per worked example). Add `transpiler/transpile_test.go` that
runs each `*.goal` through the transpiler and asserts the output equals the matching
`*.go.expected` (compare after `go/format` on both sides to ignore trivial whitespace). The test
passing is the definition of "the reference transpiler works."

---

## Step 4 — Record decisions in the running ledger (`DECISIONS.md`)

Maintain `DECISIONS.md` at the repo root as the **single running tally** of every choice made
across the whole project — not just the user-facing syntax picks. Append this feature's entries
under a `## <NN-name>` section (create it if missing). Three kinds of entry, each only when it
actually applies:

1. **Decisions (options weighed).** Anything where real alternatives existed: the chosen option,
   the options **refused**, and the **why** for both. This includes the `AskUserQuestion` syntax
   choices (record the rejected candidates from the prompt), transpile-strategy forks, and any
   design call you made while building the transpiler. Mirror the spec's own "Refused / denied"
   discipline — name the refused option *and* its justification so it stays refused.
2. **Assumptions (no discussion happened).** Anything you decided **on your own** that wasn't
   explicitly discussed with the user or fixed by the spec — naming, file layout, a default you
   picked to keep moving, an interpretation of an ambiguous spec passage. Flag these clearly as
   assumptions so the user can veto them later. **This is the category most easily skipped — do
   not skip it.** If you made a judgment call, it goes here.
3. **Refusals without a chosen alternative.** Options considered and rejected outright, with the
   reason — even if nothing replaced them.

Entry format (keep it scannable — one short block each):

```
### <decision/assumption title>
- **Kind:** decision | assumption | refusal
- **Chose:** <what, or "n/a" for a pure refusal>
- **Over:** <refused option(s)>
- **Why:** <the justification; if none exists yet, write "no justification yet — revisit">
```

If a later feature **revisits or overturns** an earlier entry, don't delete the old one — add a
new entry that references it ("supersedes <title> in <NN-name> because …"). The ledger is history,
not just current state.

---

## Step 5 — Close out

1. **Verify it runs.** From `features/<NN-name>/transpiler/`, the tests pass. Report the result
   honestly — if a case is unhandled or a test fails, say so in the TODO note; do not claim done.
2. **Update `TODO.md`:** check this feature's box (`- [x]`), and under it add a one-line pointer
   to the three artifacts plus any decision worth surfacing (e.g. "chose `match { }` with `=>`
   arms; resolved switch-coexistence rule").
3. **Stop.** Exactly one feature per iteration. Do not start the next.

---

## Step 6 — Commit the turn's work

At the **end of every iteration**, commit what you produced before stopping. This keeps each
feature's audit as its own reviewable, revertible unit of history.

- Use the **`/commit-message` (commit) skill** to author the message — never run `git commit`
  directly. Prefer the `zombiekit` git tool for staging/committing.
- Stage only this turn's artifacts (the feature's `features/<NN-name>/` files, the `TODO.md`
  checkbox update, and the new `DECISIONS.md` entries). One commit per feature.
- If the turn produced no committable change (e.g. you only asked the user a syntax question and
  are awaiting their answer), skip the commit and say so — don't make empty commits.

---

## Guardrails

- Touch only the current feature's directory, `TODO.md`, and (read-only) the spec + prior
  features. Never edit `goal-design-spec.md` — if the spec is wrong or ambiguous, note it in the
  feature's `SYNTAX.md` under an "Open against spec" heading and resolve locally.
- Don't add features, lints, or checks the spec didn't ask for (§ "Do NOT add features").
- Keep generated Go idiomatic — it must look like code a Go developer wrote (§8.3 keystone).
- Output directory naming: `features/<NN-name>/` where `<NN-name>` matches the TODO heading
  (e.g. `features/01-enums/`).
