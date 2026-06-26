# 05-question-prop — Syntax

Postfix `?` propagation (spec §3.7): if the value is `Err`/`None`, early-return it from the
enclosing function; otherwise unwrap the `Ok`/`Some`. `?` is what *earns* the divergence from
`if err != nil` — it makes the safe path the **short** path, so the model uses it instead of
routing around the safety. It composes 03-result and 04-option.

**Scope of this feature:** **open-`E`** only (`Result[T, error]`) and `Option[T]`. Closed-`E` `?`
needs the `From`-style conversion and is feature 06; it is out of scope here.

This document pins the **surface syntax**. The Go it lowers to is in `TRANSPILE.md`.

---

## 1. Final surface syntax

```goal
func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?   // if Err, returns it; else unwraps to the []byte
    cfg := parse(raw)?    // same
    return Result.Ok(cfg)
}
```

- **`?` is postfix** on a `Result`- or `Option`-typed expression — the conventional Rust/Swift
  glyph, non-negotiable (§7), not Go-ified.
- **`?` appears as `name := expr?`, `_ := expr?`, or a bare `expr?` statement.** Keep the
  unwrapped value with `name := expr?`; **discard** it (propagate only the failure) with
  `_ := expr?` or the bare `expr?` form. The bare form is the natural shape when the call's only
  output is the error — there is no value to keep — e.g. `os.MkdirAll(dir, 0o755)?`.
- **The propagation mode is the enclosing function's return type.** Inside a `Result[_, error]`
  function, `?` early-returns the `Err`; inside an `Option[_]` function, `?` early-returns `None`.
  (You cannot `?` a `Result` where the failure has nowhere compatible to go — open-`E` keeps this
  trivial: any `error` flows up as `error`, no conversion.)

### 1.1 The discard forms — `_ := expr?` and bare `expr?`

goal uses `_` as its **deliberate-discard** marker — the `match` rest-arm (§3.1) and the must-use
opt-out (§3.2, "`let _ =` spirit") — and `_ := expr?` reuses it to make a *value* discard explicit.
The bare `expr?` statement is also allowed: when the callee's only output is the error there is no
success value to discard, so the bare form hides nothing — `os.MkdirAll(dir)?` reads more cleanly
than `_ := os.MkdirAll(dir)?`. The failure (`Err`/`None`) is never silently dropped in any form;
`?` always propagates it.

The discard lowering is **arity-aware**: it emits one blank identifier per discarded value, so an
error-only callee (`func(…) error`) lowers to `if __goal_err := expr; …` and a `(T, error)` callee
to `if _, __goal_err := expr; …`. Keeping a value (`name := expr?`) requires a `(T, error)` callee;
a `name := expr?` on an error-only call is a compile error — there is nothing to bind. (Arity is
read from the callee's signature: in-file functions directly, imported functions by parsing the
package; an unresolved callee keeps the two-value form.)

---

## 2. Grammar

```ebnf
QuestionStmt = [ ( identifier | "_" ) ":=" ] Expression "?" .
```

`?` is a postfix operator that may appear only as the final token of a `QuestionStmt`. The optional
left-hand side is a single binding name (keep the value) or `_` (discard it); with no left-hand side
the statement is a bare `expr?` that discards the value and propagates only the failure. The
enclosing function must return `Result[_, error]` or `Option[_]`.

---

## 3. Worked examples

### 3.1 Result `?` chain (`examples/qprop_result`)

```goal
func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?
    cfg := parse(raw)?
    return Result.Ok(cfg)
}
```

### 3.2 Option `?` chain (`examples/qprop_option`)

```goal
func grandparent(name string) Option[User] {
    u := find(name)?
    p := parent(u.Name)?
    return Option.Some(p)
}
```

### 3.3 Discard with `_` (`examples/qprop_discard`)

```goal
func sync() Result[int, error] {
    _ := flush()?        // propagate the error; ignore the int
    return Result.Ok(1)
}
```

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| Postfix `?` | Rust/Swift `?` | "`Result`/`Option` without sugar is a pyramid of matches → model routes around it" → safe path becomes the *short* path | **Replacing** (§7) — justified: it's what makes `Result`/`Option` cheaper than `if err != nil`, so they get used |
| `?` always on an assignment; discard via `_` | Go's `_` discard; goal's own `_` (match rest-arm, must-use opt-out) | "implicit drop of the unwrapped value" → explicit, visible `_` | None — reuses `_`; uniform `lhs := expr?` shape |
| Mode from enclosing return type | Rust/Swift (`?` returns the function's error/none type) | n/a | None — no annotation needed |

`?` is a conventional glyph (§7) and is kept verbatim.

---

## 5. Resolved open questions / deferred

- **Bare `expr?`** — refused; `?` must be `name := expr?` or `_ := expr?` (see §1.1).
- **Closed-`E` `?`** — deferred to feature 06: it must convert the callee's error enum into the
  caller's via a `From`-style conversion auto-invoked by `?`. Open-`E` (`error`) needs no
  conversion and ships here (§3.7 "ship this in v1"). The two modes are the **same** `?` with/without
  a conversion step — kept one mechanism.
- **Inline / sub-expression `?`** (`g(f()?)`) — out of scope for the reference transpiler; `?` is
  handled at statement level (`lhs := expr?`). The surface rule (§2) already restricts `?` to that
  position.
- **Must-use / value-position match** — checker / earlier-feature concerns, unchanged.

---

## 6. Open against spec

None. The spec §8.3 sample wrote `raw := readFile(p)?` (the bind form) and bare `Ok(...)`; this
audit keeps the bind form, adds the explicit `_ := expr?` discard (rather than a bare `expr?`), and
uses the qualified `Result.Ok` / `Option.Some` established in 03/04. The `?` semantics and the §8.3
lowering are unchanged, so no spec semantics change.
