# 08-no-zero-value — Decisions

This file records the settled design decisions for required-field construction
(`...defaults`). It is referenced from `SYNTAX.md`.

## D1 — Explicit-defaults form: `...defaults`

The trailing keyword-spread element `...defaults` was chosen (via
`AskUserQuestion`) over the alternatives `_` (rest marker), a bare `default`
member, and `..Default` (Rust struct-update tail).

Rationale, tied to the two guiding principles:

- **Familiarity:** the leading `...` is a visual cousin of Go's own `...`
  (variadic / slice spread) and Rust's `..` struct-update tail — readers parse
  "and the rest" immediately, and no new sigil is introduced.
- **Feedback (located error):** the keyword *names the intent* ("defaults") and
  is greppable. Because opting into defaults is explicit and visible, a
  genuinely forgotten field cannot masquerade as a deliberate default — the
  checker still flags a missing field when `...defaults` is absent.

`..Default`, bare `default`, and `_` were refused: `_` collides with the
rest-arm marker in `match` (feature 02) and reads as "ignore" rather than
"default the rest"; a bare `default` member looks like a field named `default`;
`..Default` imports Rust's struct-update semantics (copy from a value) that this
feature does not have.

## D2 — Unsafe zeros are rejected, and it is a `goal check` error

`...defaults` fills a field only when the field's zero value is **safe** —
usable as-is. A field whose zero is a latent hazard (a nil pointer, map, chan,
or func; a method-bearing named interface; an `enum`/sealed sum type with no
zero variant) is a **located compile error** instead. This narrowly exceeds the
original §3.5 framing ("`...defaults` = zero values"): it rejects *unsafe* zeros
while still filling safe ones, and stays scoped to the fields `...defaults`
fills — pervasive nil-elimination across all pointers remains the deferred §5
decision.

**Where it is enforced:** the guarantee is a **first-class checker diagnostic**,
code `[unsafe-default]` (feature 08, severity error), emitted by the lexical
checker (`internal/sema`) and surfaced by **`goal check`** — not only as a
`goal build` transpile failure. The unsafe-zero classification is the single
shared rule `sema.ZeroSafety`, which the backend also consults on emit, so
`goal check` and `goal build` agree on exactly which zeros are unsafe.
Consequences:

- `goal check <file>` on an unsafe `...defaults` exits non-zero and prints
  `file:line:col: error: [unsafe-default] …`, catching the footgun before any
  code is emitted; the same literal with the offending field set explicitly is
  accepted.
- The behavior is regression-tested by
  `testdata/check/08-no-zero-value/unsafe_default.goal` (a positive `// want`
  case) and remains enforced end-to-end by the existing `goal build` path.

Decided with the user; letting an unsafe zero through the escape hatch would
reopen the exact silent-zero footgun the feature closes.
