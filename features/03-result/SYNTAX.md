# 03-result — Syntax

`Result[T, E]` as the error channel (spec §3.2): fallible functions return a `Result` as the
**whole** return value — a sum of `Result.Ok(T)` / `Result.Err(E)`. The success value lives
*inside* the sum, so it cannot be reached without going through the error path. This eliminates
Go's biggest silent-failure class (`cfg, _ := parse(s)`): there is no separate error return to
`_`-discard, because the error is a *branch of the value you hold*.

**Scope of this feature:** the **open-`E`** common case, `Result[T, error]`, consumed
**immediately** (returned and `match`-ed at the use site) — the §8.3 native-`(T, error)` keystone.
Closed-`E` (sum encoding) is feature 06; `?` propagation is feature 05; the immediate-vs-stored
fallback (§8.7) is noted but not built here.

This document pins the **surface syntax**. The Go it lowers to is in `TRANSPILE.md`.

---

## 1. Final surface syntax

```goal
func parse(s string) Result[Config, error] {
    if s == "" {
        return Result.Err(errors.New("empty input"))
    }
    return Result.Ok(Config{Raw: s})
}

// Must handle — the success value is unreachable without going through the Result:
match parse(input) {
    Result.Ok(cfg) => run(cfg)
    Result.Err(e) => report(e)
}
```

- **`Result[T, E]`** — Go-generics bracket form, both type arguments **always explicit**. The open
  case is written `Result[T, error]`; there is no `Result[T]` shorthand (the error channel stays
  visible in every signature).
- **`Result` is the whole return.** A fallible function returns exactly `Result[T, E]`, never a
  tuple containing a `Result` (so it cannot be destructured around — §3.2).
- **Construction is qualified**: `Result.Ok(v)` / `Result.Err(e)` — same `Type.Variant(...)`
  qualification as 01-enums construction (`Status.Active(...)`). `Ok`/`Err` remain the
  conventional variant names (§7); only the qualification is added, for one uniform sum-type
  construction rule.
- **Consumed via `match`** (feature 02): arms `Result.Ok(binding)` / `Result.Err(binding)` bind the
  success value / the error (bind-the-value, the 02 binding form).

### 1.1 The `Result`-as-whole-return rule

```goal
func parse(s string) Result[Config, error]        // OK — Result is the entire return
func parse(s string) (Result[Config, error], int) // REJECTED — Result mixed into a tuple
func dims() (int, int)                             // OK — genuine multi-infallible return, no Result
```

Mixing a `Result` into a tuple is refused (§3.2): the moment fallibility is one slot in a tuple,
it can be destructured around, back to Go's ignored-error problem. (Checker-enforced; see §5.)

---

## 2. Grammar

```ebnf
ResultType   = "Result" "[" Type "," Type "]" .          (* E is `error` in the open case *)
OkExpr       = "Result" "." "Ok"  "(" Expression ")" .
ErrExpr      = "Result" "." "Err" "(" Expression ")" .
ResultMatch  = "match" Expression "{" OkArm ErrArm "}" . (* arms in either order *)
OkArm        = "Result" "." "Ok"  [ "(" identifier ")" ] "=>" ArmBody .
ErrArm       = "Result" "." "Err" [ "(" identifier ")" ] "=>" ArmBody .
```

`ResultType` appears only as a whole function return type. `OkExpr`/`ErrExpr` appear in return
position (the immediate case). `ResultMatch` is the §3.2 consumer; it is an instance of the
02-match construct specialized to the `Ok`/`Err` variants.

---

## 3. Worked examples

### 3.1 Producer — both Ok and Err returns (`examples/result_parse`)

```goal
func parse(s string) Result[Config, error] {
    if s == "" {
        return Result.Err(errors.New("empty input"))
    }
    return Result.Ok(Config{Raw: s})
}
```

### 3.2 Consumer — `match` on the Result (`examples/result_match`)

```goal
func handle(input string) {
    match parse(input) {
        Result.Ok(cfg) => run(cfg)
        Result.Err(e) => report(e)
    }
}
```

A consumer that does not need the success value still goes through the Result; the binding is
simply unused:

```goal
func handle2(input string) {
    match parse(input) {
        Result.Ok(cfg) => done()
        Result.Err(e) => report(e)
    }
}
```

### 3.3 Non-struct `T`, multiple `Err` returns (`examples/result_int`)

```goal
func parsePositive(s string) Result[int, error] {
    n, err := strconv.Atoi(s)
    if err != nil {
        return Result.Err(err)
    }
    if n <= 0 {
        return Result.Err(errors.New("not positive"))
    }
    return Result.Ok(n)
}
```

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| `Result[T, E]` as whole return | Rust/Swift `Result`/`throws`-as-value | "`cfg, _ := parse(s)` silently drops the error" → the error is a branch you must handle | **Replacing** (§7) — justified: kills Go's single biggest silent-failure class |
| Both args explicit `Result[T, error]` | plain Go generics | n/a (visibility) | None beyond the bracket; no defaulting magic added |
| Qualified `Result.Ok` / `Result.Err` | 01-enums `Type.Variant(...)` construction | n/a (consistency) | Small: spends a little of §7's conventional-name budget for one uniform sum-type construction rule (see DECISIONS) |
| `Ok` / `Err` variant names | universal Result idiom | n/a | None — conventional names kept, not Go-ified |
| Consume via `match` | 02-match | binds the success value only through the handled branch | reuses 02; no new spend |

`Ok`/`Err` are conventional names (§7) — kept verbatim; only the `Result.` qualifier is added.

---

## 5. Resolved open questions / checker-side rules (not transpiled here)

- **Must-use.** Ignoring a `Result` is a compile error unless explicitly discarded. This is the
  checker's job (it tracks `Result`-typed expression results and flags unused ones); the reference
  transpiler does **not** implement it. The **explicit-discard surface** (how to intentionally drop
  a Result) is deferred to land together with must-use enforcement — it has no meaning until the
  check exists, and its lowering (a single-value `_ =` must become `_, _ =` over the native tuple)
  is best decided with that machinery. Noted here so the gap is deliberate.
- **`Result`-as-whole-return enforcement.** Rejecting `Result` mixed into a tuple is a checker
  rule; the transpiler simply only recognizes `Result` in whole-return position.
- **Open-vs-closed `E` policy.** Out of scope here (open-`E` only); the lint policy and the
  closed-`E` lowering are feature 06.

---

## 6. Open against spec

None. The spec §3.2 sample wrote bare `Ok(...)` / `Err(...)`; this audit qualifies them as
`Result.Ok(...)` / `Result.Err(...)` per the user's choice, for one uniform construction rule
across all sum types (01-enums, Result). The variant names and the `Result[T, error]` shape are
unchanged, and the lowering is the §8.3 native-`(T, error)` keystone, so no spec semantics change.
