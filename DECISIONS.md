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
