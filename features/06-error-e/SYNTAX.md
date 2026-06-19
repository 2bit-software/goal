# 06-error-e — Syntax

Error type `E`: open *and* closed, one mechanism (spec §3.3). `E` may be the open `error` interface
(feature 03) **or** a **closed error enum** — exhaustively matchable, with the failure set visible
in the signature. Both are the **same** `Result` / `match` / `?` / must-use mechanism, differing
**only** in whether `E` is `error` or a closed enum. A function moves open→closed by *narrowing
`E`*, nothing else. **This one-mechanism-one-knob constraint is the line to protect** (§3.3).

This feature pins the closed-`E` surface and resolves the §9 `From`-conversion shape. The Go it
lowers to (the §8.1 sum encoding) is in `TRANSPILE.md`.

---

## 1. Final surface syntax

A closed error enum is **just an `enum`** (feature 01) used as the `E` of a `Result`:

```goal
enum ParseError {
    Empty
    BadKey { key: string }
}

func parse(s string) Result[Config, ParseError] {
    if s == "" {
        return Result.Err(ParseError.Empty)
    }
    return Result.Ok(Config{Raw: s})
}

match parse(input) {
    Result.Ok(cfg) => run(cfg)
    Result.Err(e) => report(e)
}
```

- **No new construction/match/`?` syntax.** `Result[T, E]`, `Result.Ok`/`Result.Err`, `match`, and
  `?` are exactly as pinned in 01–05. The *only* difference from open-`E` is that `E` is an enum
  (here `ParseError`) instead of `error`. That is the "one knob."
- **The failure set is visible** in the signature (`Result[Config, ParseError]`) and
  **exhaustively matchable**.

### 1.1 `From`-conversion for `?` across mismatched error enums

When `?` propagates a callee's error enum into a function whose error type differs, the conversion
is a **`from func`** — an ordinary function marked with a `from` modifier (the same modifier shape
as `pure func`, feature 09). `?` auto-invokes it, resolved by its `(Src) → Dst` signature:

```goal
enum AppError {
    Wrapped { cause: ParseError }
}

from func toApp(e ParseError) AppError {
    return AppError.Wrapped(cause: e)
}

func load(s string) Result[Config, AppError] {
    cfg := parse(s)?      // parse fails as ParseError; ? auto-applies toApp -> AppError
    return Result.Ok(cfg)
}
```

- The `from` keyword **marks** the function as a `?`-conversion and **erases** in the generated Go
  (it becomes a plain `func`). It is the explicit, on-the-page signal that `?` will reach for this
  function — no hidden discovery.
- `?` selects the `from func` whose signature is `(calleeError) callerError`. Open-`E` `?` needs no
  conversion (any `error` flows up as `error`); the conversion is the friction cost of typed errors.

---

## 2. Grammar

```ebnf
ClosedResult = "Result" "[" Type "," EnumName "]" .   (* E is an enum, not `error` *)
FromConv     = "from" "func" identifier "(" identifier Type ")" Type Block .
```

Everything else (`enum`, `Result.Ok`/`Result.Err`, `match`, `?`) is unchanged from 01–05. A
`from func` is an ordinary function declaration prefixed with the `from` modifier; its single
parameter's type is the conversion *source* and its return type is the *target*.

---

## 3. Worked examples

### 3.1 Closed-`E` Result + flat `match` (`examples/qclosed_match`)

```goal
match parse(input) {
    Result.Ok(cfg) => run(cfg)
    Result.Err(e) => report(e)
}
```

The `Err` arm binds the whole error enum value (`e`); to branch on its variants, `match e { ... }`
(feature 02) — flat `Ok`/`Err` here, with nested error-variant patterns left to plain composition.

### 3.2 `?` with matching error type (`examples/qclosed_prop_same`)

```goal
func loadFirst(a string) Result[Config, ParseError] {
    cfg := parse(a)?           // caller and callee both error as ParseError -> no conversion
    return Result.Ok(cfg)
}
```

### 3.3 `?` with `From`-conversion (`examples/qclosed_prop_from`)

```goal
from func toApp(e ParseError) AppError { return AppError.Wrapped(cause: e) }

func load(s string) Result[Config, AppError] {
    cfg := parse(s)?           // ParseError -> AppError via toApp
    return Result.Ok(cfg)
}
```

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| Closed `E` = an `enum` as the `Result` error type | Rust/Swift typed errors; reuses goal's own enum | "the failure set is invisible / unmatchable" → visible in the signature, exhaustively matched | None new — it's `enum` + `Result`, both already pinned |
| One mechanism, one knob (open↔closed = `E` only) | — | keeps `Result`/`match`/`?`/must-use identical so lint-as-policy works (§3.3) | None — *refuses* to add a second error system |
| `from func` modifier for conversion | Rust `From`; goal's own `pure func` modifier | "`?` can't bridge mismatched typed errors" → an explicit, visible conversion `?` applies | Small: one `from` keyword, erasing, on an ordinary func |

`Ok`/`Err`, `=>`, `?` are kept verbatim (§7).

---

## 5. Resolved open questions

- **§9 `From`-style conversion** — resolved as the `from func` modifier (see §1.1). Rejected: a
  dedicated `from … to` block (more novel) and an unmarked func discovered by signature (implicit /
  ambiguous). The two `?` modes (open with no conversion, closed with a `from func`) are the **same**
  `?` mechanism with/without a conversion step — the §3.3 line-to-protect.
- **Lint-level open-vs-closed policy** — **not** a transpile concern (§3.3 / TODO). Only the two
  lowerings (native tuple for open, sum encoding for closed) live here; which is the *default* is a
  lint posture, decided empirically, out of scope.

---

## 6. Open against spec

None. The spec §3.3 sample wrote bare `Ok`/`Err` and showed **nested** `Err(BadKey(k))` patterns;
this audit uses the qualified `Result.Ok`/`Result.Err` from 03 and **flat** `Err(e)` arms (nested
error-variant matching is left to composing `match e { … }`, an explicit later extension). The §9
`From` shape, undefined in the spec, is pinned as `from func`. The encoding is the §8.1 sum the spec
mandates for closed `E`, so no spec semantics change.
