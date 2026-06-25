# 03-result — Transpile to Go

Governing contract: spec §8.3 (the `Result` keystone) and §8.0 / §8.7. This feature implements the
**open-`E`, consumed-immediately** path only: `Result[T, error]` → native Go `(T, error)`. That
single lowering is what makes goal's output *idiomatic* Go with full stdlib interop — the highest-
leverage lowering in the language (§8.0).

---

## 1. The lowering (§8.3)

| goal | Go |
|---|---|
| `func f(...) Result[T, error]` | `func f(...) (__goal_ok T, __goal_err error)` |
| `return Result.Ok(v)` | `return v, nil` |
| `return Result.Err(e)` | `return __goal_ok, e` |
| `match call { Result.Ok(x) => A; Result.Err(e) => B }` | `__goal_v, __goal_err := call` + `if __goal_err != nil { B } else { A }` |

The success type `Result[T, error]` becomes the native pair `(T, error)`; `?` (feature 05) and a
`match` both consume that pair with the `if err != nil` shape the model already knows.

---

## 2. Input → output pairs

### 2.1 Producer (`examples/result_parse`)

```goal
func parse(s string) Result[Config, error] {
    if s == "" {
        return Result.Err(errors.New("empty input"))
    }
    return Result.Ok(Config{Raw: s})
}
```

```go
func parse(s string) (__goal_ok Config, __goal_err error) {
	if s == "" {
		return __goal_ok, errors.New("empty input")
	}
	return Config{Raw: s}, nil
}
```

### 2.2 Consumer (`examples/result_match`)

```goal
match parse(input) {
    Result.Ok(cfg) => run(cfg)
    Result.Err(e) => report(e)
}
```

```go
__goal_v, __goal_err := parse(input)
if __goal_err != nil {
	report(__goal_err)
} else {
	run(__goal_v)
}
```

When the Ok arm does not use its binding, the success value is discarded so no unused variable is
produced:

```goal
match parse(input) {
    Result.Ok(cfg) => done()
    Result.Err(e) => report(e)
}
```

```go
_, __goal_err := parse(input)
if __goal_err != nil {
	report(__goal_err)
} else {
	done()
}
```

### 2.3 Non-struct `T`, multiple `Err` returns (`examples/result_int`)

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

```go
func parsePositive(s string) (__goal_ok int, __goal_err error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return __goal_ok, err
	}
	if n <= 0 {
		return __goal_ok, errors.New("not positive")
	}
	return n, nil
}
```

Both `Err` returns reuse the same zero (`__goal_ok`, an `int` here) with no per-return declaration —
see §3.2.

---

## 3. Lowering rules (the general algorithm)

### 3.1 Return type → native pair

`func ... Result[T, error]` → `func ... (__goal_ok T, __goal_err error)`. T is copied verbatim from
the first type argument; the error return is `__goal_err error`.

### 3.2 Why named returns

`return Result.Err(e)` must produce `(zero of T, e)`, but a no-type-inference reference transpiler
cannot synthesize a type-correct zero **literal** (`Config{}` vs `0` vs `nil` vs …) from a bare
type name. **Named returns** sidestep this entirely: `__goal_ok` *is* the zero value of `T`, for any
`T`, with no literal to spell and no per-return `var` to declare (which would otherwise collide
across multiple `Err` returns). So `return Result.Err(e)` → `return __goal_ok, e`. A checker-backed
compiler with full type information could instead emit the spec's literal form
(`(Config, error)` + `Config{}`); the named-return shape is the type-agnostic equivalent and is
itself idiomatic Go. (Recorded as an assumption in `DECISIONS.md`.)

### 3.3 Construction in return position

- `return Result.Ok(X)` → `return X, nil` (X copied verbatim).
- `return Result.Err(X)` → `return __goal_ok, X`.

Only recognized in `return` position (the immediate case). `Result.Ok/Err` used to build a
**stored** value (`xs := []Result[...]{Result.Ok(1)}`) is the §8.7 stored case → sum encoding,
out of scope here.

### 3.4 Statement-position `match` on a Result (native if/else)

For `match <call> { Result.Ok(x) => A; Result.Err(e) => B }`:

1. Capture the pair: `<okLHS>, __goal_err := <call>`, where `okLHS` is `__goal_v` if the Ok arm uses
   its binding, else `_` (the error LHS is always `__goal_err` — it is the branch discriminant).
2. Branch on the error, Err first (the idiomatic `if err != nil`): `if __goal_err != nil { B } else
   { A }`.
3. Rewrite bindings: the Ok binding → `__goal_v` (the whole success value), the Err binding →
   `__goal_err`, throughout their arm bodies. Field reads keep their selector (`cfg.Raw` →
   `__goal_v.Raw`); no capitalization, since the payload is an ordinary value, not an enum variant.

This is **not** a type switch (contrast 02-match on an enum): there is no sum value at runtime in
the open-`E` strategy — only `(T, error)` — so matching lowers to the error branch, which is the
whole point of the native-tuple keystone.

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| **Must-use** | **Erased** (and not implemented here) | A static guarantee — it constrained which source was legal; it produces no runtime behavior. |
| **The `Result` sum wrapper (open-E)** | **Erased into `(T, error)`** | In the open-E immediate case there is no runtime sum value at all; the success/error split *is* Go's native pair. The "sum type" exists only in the source's type system. |
| **The error branch / control flow** | **Preserved** | Which path runs (`Ok` vs `Err`), and the early-return shape, are runtime semantics — preserved exactly as the `if err != nil` / `else`. |
| **Payload values** (`T`, the `error`) | **Preserved** | Real runtime data carried by the native pair. |

No defensive `panic` arises here: the open-`E` `match` is a two-way `if`/`else` over `err`, which
is total by construction (either `err != nil` or not) — there is no proven-unreachable point to
guard. (The defensive-panic rule applies to the closed-`E` sum-encoded `match`, feature 06, and to
exhaustive enum `match`, feature 02.)

---

## 5. Strategy forks (§8.7)

| Case | Strategy | Status |
|---|---|---|
| Open-`E`, consumed immediately (returned + match/`?`) | **native `(T, error)`** | **implemented (this feature)** |
| `Result` stored as a first-class value (slice/map/struct field/passed around) | **sum encoding** (the §8.1 universal fallback) | out of scope — feature 06 / later; the transpiler does not handle stored Results |
| Closed-`E` `Result` | **sum encoding**, `?` via type-switch-and-return | feature 06 |

The transpiler implements the first row. It detects the immediate case structurally (Result in
whole-return position; `Result.Ok/Err` in return position; `match` directly on a Result-returning
call). Value-position Result `match` (`x := match ...`) is **deferred** with a located message, as
is any stored Result.

---

## 6. Hygiene

Synthesized names use the `__goal_` prefix (§8): `__goal_ok` / `__goal_err` (named returns) and
`__goal_v` / `__goal_err` (the captured pair at a match site). User identifiers and the copied
`T`, `Ok`/`Err` payload expressions, and arm bodies are otherwise untouched.
