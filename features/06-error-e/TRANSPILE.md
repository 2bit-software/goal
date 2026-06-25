# 06-error-e — Transpile to Go

Governing contract: spec §8.3 (closed-`E` fork) and §8.1 (the sum encoding) / §8.0. A closed-`E`
`Result` is **the universal sum-type fallback** (§8.0): it does *not* use the open-`E` native
tuple. `match` is a type switch; `?` is type-switch-and-return with a `From`-conversion in the
`Err` arm.

---

## 1. The encoding

A closed-`E` `Result[T, E]` lowers to a generic sum encoding, injected once per file:

```go
type Result[T, E any] interface{ isResult() }
type Ok[T, E any] struct{ Value T }
type Err[T, E any] struct{ Value E }

func (Ok[T, E]) isResult()  {}
func (Err[T, E]) isResult() {}
```

A `Result[Config, ParseError]` **return type stays as written** (it now names the injected generic
interface). The error enum `E` is itself the §8.1 sealed-interface encoding (feature 01).

| goal | Go |
|---|---|
| `Result[T, E]` (closed) | `Result[T, E]` (the injected generic interface) — unchanged |
| `Result.Ok(v)` | `Ok[T, E]{Value: v}` (T, E from the enclosing function's return type) |
| `Result.Err(e)` | `Err[T, E]{Value: e}` |
| `match call { Ok(x) => A; Err(e) => B }` | `switch __goal_e := call.(type) { case Ok[T,E]: …; case Err[T,E]: …; default: panic(…) }` |
| `name := call?` (same E) | type switch: `Ok` assigns `name`, `Err` returns `Err[T2,E]{Value: __goal_e.Value}` |
| `name := call?` (E → E2) | as above, but `Err` returns `Err[T2,E2]{Value: conv(__goal_e.Value)}` |
| `from func c(e Src) Dst {…}` | `func c(e Src) Dst {…}` (`from` erased) |

T, E for construction come from the **enclosing** function; for `match`/`?` the **callee's** Result
type (looked up from its signature) supplies the `Ok[T,E]`/`Err[T,E]` case types, and the **caller's**
supplies the `?` early-return type.

---

## 2. Input → output pairs

### 2.1 Construction + flat match (`examples/qclosed_match`)

```goal
func parse(s string) Result[Config, ParseError] {
    if s == "" { return Result.Err(ParseError.Empty) }
    return Result.Ok(Config{Raw: s})
}
match parse(input) {
    Result.Ok(cfg) => run(cfg)
    Result.Err(e) => report(e)
}
```

```go
func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Err[Config, ParseError]{Value: ParseError(ParseError_Empty{})}
	}
	return Ok[Config, ParseError]{Value: Config{Raw: s}}
}

func handle(input string) {
	switch __goal_e := parse(input).(type) {
	case Ok[Config, ParseError]:
		cfg := __goal_e.Value
		run(cfg)
	case Err[Config, ParseError]:
		e := __goal_e.Value
		report(e)
	default:
		panic("unreachable: non-exhaustive Result[Config, ParseError] (compiler invariant violated)")
	}
}
```

`Result.Err(ParseError.Empty)` lowers in two composed steps: the `Result.Err(…)` wrap, and the
inner enum construction `ParseError.Empty` → `ParseError(ParseError_Empty{})` (feature 01).

### 2.2 `?` with matching error type (`examples/qclosed_prop_same`)

```goal
func loadFirst(a string) Result[Config, ParseError] {
    cfg := parse(a)?
    return Result.Ok(cfg)
}
```

```go
func loadFirst(a string) Result[Config, ParseError] {
	var cfg Config
	switch __goal_e := parse(a).(type) {
	case Ok[Config, ParseError]:
		cfg = __goal_e.Value
	case Err[Config, ParseError]:
		return Err[Config, ParseError]{Value: __goal_e.Value}
	default:
		panic("unreachable: non-exhaustive Result[Config, ParseError] (compiler invariant violated)")
	}
	return Ok[Config, ParseError]{Value: cfg}
}
```

### 2.3 `?` with `From`-conversion (`examples/qclosed_prop_from`)

```goal
from func toApp(e ParseError) AppError { return AppError.Wrapped(cause: e) }

func load(s string) Result[Config, AppError] {
    cfg := parse(s)?
    return Result.Ok(cfg)
}
```

```go
func toApp(e ParseError) AppError {
	return AppError(AppError_Wrapped{Cause: e})
}

func load(s string) Result[Config, AppError] {
	var cfg Config
	switch __goal_e := parse(s).(type) {
	case Ok[Config, ParseError]:
		cfg = __goal_e.Value
	case Err[Config, ParseError]:
		return Err[Config, AppError]{Value: toApp(__goal_e.Value)}
	default:
		panic("unreachable: non-exhaustive Result[Config, ParseError] (compiler invariant violated)")
	}
	return Ok[Config, AppError]{Value: cfg}
}
```

The `Err` arm switches on the **callee's** `Err[Config, ParseError]` and returns the **caller's**
`Err[Config, AppError]`, wrapping the value in `toApp(...)` — the `from func` resolved by the
`(ParseError) → AppError` pair.

---

## 3. Lowering rules

1. **Encoding.** Inject the generic `Result`/`Ok`/`Err` once (after imports) when any closed-`E`
   Result is present. Closed-`E` is detected by `Result[T, E]` with `E != error`.
2. **Construction.** `Result.Ok(X)` / `Result.Err(X)` wrap to `Ok[T, E]{Value: X}` /
   `Err[T, E]{Value: X}`; `T, E` are the enclosing function's Result type. The inner `X` is lowered
   independently (so an enum construction inside `Err(…)` is also lowered).
3. **match** (statement position, scrutinee a direct call): a type switch with `case Ok[T, E]:` /
   `case Err[T, E]:` (T, E from the callee's signature) and a defensive `panic` default (the Ok/Err
   set is proven exhaustive, §8.0). A used `Ok`/`Err` binding becomes `name := __goal_e.Value`.
4. **`?`** (`name := call?`): `var name T_callee` then a type switch — the `Ok` arm assigns `name`,
   the `Err` arm `return`s the caller's `Err[T_caller, E_caller]`. If `E_callee == E_caller` the
   value passes through; otherwise it is wrapped in the `from func` for `(E_callee) → E_caller`. The
   default is the defensive `panic`.
5. **`from func`.** The `from` keyword is stripped (the function lowers as an ordinary `func`); the
   transpiler records `(Src, Dst) → name` for `?` to resolve.

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| Exhaustiveness of `Ok`/`Err` (and of error-enum match) | **Erased** check, **defensive `panic`** default | The checker proved both cases handled; the `default` is unreachable, so per §8.0 it fails loud, not silently. |
| The `from` marker | **Erased** to a plain `func` | A static designation; it produced no runtime behavior beyond the call `?` emits. |
| The sum value (`Ok`/`Err`), the error enum value, payloads | **Preserved** | Real runtime data — unlike open-`E`, closed-`E` *does* carry a runtime sum value (the §8.1 encoding). |
| The `?` / `match` control flow and the `From` call | **Preserved** | The branch taken and the conversion applied are program behavior. |

This is the key contrast with feature 03: **open-`E` erases the sum into a native tuple; closed-`E`
preserves it as the sum encoding** (§8.0's "universal fallback").

---

## 5. Strategy forks (§8.3)

| Case | Strategy | Status |
|---|---|---|
| Closed-`E` `Result` (E is an enum) | **sum encoding** + type-switch match + type-switch-and-return `?` | **implemented (this feature)** |
| Open-`E` `Result[T, error]` | native `(T, error)` + `if err != nil` | feature 03 / 05 |
| `?` across mismatched closed `E` | sum encoding + `from func` conversion in the `Err` arm | **implemented** |

Open- and closed-`E` `?` are the **same** mechanism with/without a conversion step (§3.3). The
transpiler resolves the callee's Result type and the enclosing function's Result type from
signatures, so the `match`/`?` scrutinee must be a **direct call**. Nested `Err`-variant patterns,
value-position match, and stored Results are out of scope.

---

## 6. Hygiene

The type-switch guard is the hygienic `__goal_e` (§8). Synthesized type names follow §8.1
(`Enum_Variant`); the injected `Result`/`Ok`/`Err` are the sum encoding. User identifiers, payload
expressions, and arm bodies are otherwise untouched.
