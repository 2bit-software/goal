# 10-assert — runtime assertions

## Final surface syntax

`assert` is a statement that checks a boolean condition at runtime and panics on failure:

```goal
assert <cond>                          // bare — auto message only
assert <cond>, <fmt> [, <args>...]     // printf-style explanation appended
```

The panic message **always** includes the source text of `<cond>` (§8.6's located-feedback rule);
the optional message is a printf format string + args, appended after the auto text.

### Chosen form: printf-style message, bare fallback (user-selected)

The user chose the printf-style form with a bare fallback over "bare only" and "single string
message":

- **Bare `assert cond`** is valid on its own — no message argument required. This is the common case
  and matches the spec's sample (`assert amount > 0`).
- **`assert cond, "fmt", args...`** attaches a formatted human explanation (the `t.Errorf` idiom),
  so the offending values can be interpolated (`"n=%d not in [%d, %d]", n, lo, hi`).

Rationale, per the two principles:

- **Feedback:** the auto-included expression text already converts a bare runtime panic into located
  feedback ("which invariant failed, verbatim"). The printf message adds *why it matters* and the
  *actual values*, the highest-information failure a runtime check can give.
- **Familiarity:** `assert cond, "fmt", args...` is exactly Go's `t.Errorf` / `fmt.Errorf` shape and
  Python's `assert cond, msg` — no novel spelling. The bare fallback means the simple case stays
  one token + an expression.

## Grammar

```ebnf
AssertStmt = "assert" Expression [ "," FormatString { "," Expression } ] .
```

- `assert` is a statement keyword, valid only at statement position (the first token on its line in
  this reference). It is **not** an expression — it has no value.
- The condition is any boolean `Expression`. The message split is on the **first top-level comma**,
  so a condition containing a call with internal commas (`assert clamp(lo, hi, n) == n, …`) parses
  correctly.
- `FormatString` is a string literal; the following expressions are its printf arguments.

## Worked examples

### 1. Bare assert (common case)

```goal
func withdraw(balance int, amount int) int {
    assert amount > 0
    return balance - amount
}
```

### 2. Printf-style message

```goal
func setAge(age int) {
    assert age >= 0, "age must be non-negative, got %d", age
}
```

### 3. Mixed, with internal commas and a `%` in the condition

```goal
func check(lo int, hi int, n int) {
    assert lo <= hi
    assert clamp(lo, hi, n) == n, "n=%d not in [%d, %d]", n, lo, hi
    assert n%2 == 0
}
```

The `clamp(lo, hi, n)` commas are below top level, so only the comma before `"n=%d…"` splits
condition from message. The `%` in `n%2` is harmless — the expression text is emitted as a string
literal, never as a format string.

## Rationale, tied to the two principles

- **Priority #1 at the next-best band:** asserts catch errors at *runtime* — the band just below
  compile time (§4.3). They encode invariants the type system can't capture, cheaply and familiarly.
- **Located feedback is mandatory, message is optional:** the spec requires the source expression
  text in the failure (§8.6); the user-chosen printf message is layered on top for the values.
- **Conventional shape preserved:** the message form is the Go/Python assert idiom verbatim; no
  familiarity budget spent on novel punctuation.

## Resolved open questions (§9)

- **Message form** → **printf-style with a bare fallback** (`assert cond [, "fmt", args...]`).
  Decided via `AskUserQuestion`; "bare only" and "single string message" were the rejected
  alternatives (see `DECISIONS.md`). §9 has no other open item for asserts.

## Reserved (designed-in, not built)

- **Statically-checkable subset.** §4.3 reserves syntax for asserts the checker can later *discharge
  statically* (simple ranges, enum-exhaustiveness facts) instead of at runtime. No surface syntax is
  added for it now — the same `assert` keyword will carry it; a v1 assert is always runtime-checked.
  General Dafny-style static verification is refused (undecidable, slow feedback).
- **Pre/postconditions / contracts (§5)** share this posture: if added, runtime-checked in v1 with
  syntax reserved. Out of scope for this feature.

## Open against spec

- **Build-tag strip toggle.** §8.6 says the lowering should be *toggleable* via a build tag so
  release builds can strip asserts. NEXT-SESSION confirms v1 need not implement stripping. The
  reference transpiler therefore always emits the assert; the strip mode is documented in
  `TRANSPILE.md` as the reserved strategy, not built.
