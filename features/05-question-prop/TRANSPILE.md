# 05-question-prop — Transpile to Go

Governing contract: spec §8.3 (`Result` and `?` dual strategy) and §3.7. This feature lowers the
**open-`E`** `?`: `?` becomes the `if err != nil` block the model was trained on (Result) or the
nil-check early return (Option). It composes the 03-result and 04-option lowerings for the
signatures and constructions the `?` sites depend on.

---

## 1. The lowering (§8.3)

**Result `?`** (enclosing function returns `Result[_, error]` → named `(__goal_ok T, __goal_err error)`):

| goal | Go |
|---|---|
| `name := expr?` | `name, __goal_err := expr` + `if __goal_err != nil { return __goal_ok, __goal_err }` |
| `_ := expr?` | `if _, __goal_err := expr; __goal_err != nil { return __goal_ok, __goal_err }` |

**Option `?`** (enclosing function returns `Option[_]` → `*T`):

| goal | Go |
|---|---|
| `name := expr?` | `__goal_oN := expr` + `if __goal_oN == nil { return nil }` + `name := *__goal_oN` |
| `_ := expr?` | `if __goal_oN := expr; __goal_oN == nil { return nil }` |

The Result form *is* textbook idiomatic Go — `?` becomes `if err != nil`, and the result
interoperates with the entire stdlib (§8.3 keystone).

---

## 2. Input → output pairs

### 2.1 Result `?` chain (`examples/qprop_result`)

```goal
func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?
    cfg := parse(raw)?
    return Result.Ok(cfg)
}
```

```go
func loadConfig(p string) (__goal_ok Config, __goal_err error) {
	raw, __goal_err := readFile(p)
	if __goal_err != nil {
		return __goal_ok, __goal_err
	}
	cfg, __goal_err := parse(raw)
	if __goal_err != nil {
		return __goal_ok, __goal_err
	}
	return cfg, nil
}
```

### 2.2 Option `?` chain (`examples/qprop_option`)

```goal
func grandparent(name string) Option[User] {
    u := find(name)?
    p := parent(u.Name)?
    return Option.Some(p)
}
```

```go
func grandparent(name string) *User {
	__goal_o1 := find(name)
	if __goal_o1 == nil {
		return nil
	}
	u := *__goal_o1
	__goal_o2 := parent(u.Name)
	if __goal_o2 == nil {
		return nil
	}
	p := *__goal_o2
	return &p
}
```

### 2.3 Discard with `_` (`examples/qprop_discard`)

```goal
func sync() Result[int, error] {
    _ := flush()?
    return Result.Ok(1)
}
```

```go
func sync() (__goal_ok int, __goal_err error) {
	if _, __goal_err := flush(); __goal_err != nil {
		return __goal_ok, __goal_err
	}
	return 1, nil
}
```

### 2.4 Nested `?` in expression position (`examples/qprop_nested_arg`)

`?` also composes *inside* an expression — a call argument, a binary operand, or a
return payload. Each nested `?` is **hoisted** onto its own statement just before the
enclosing one, bound to a fresh temp, and reuses the same unwrap-or-early-return
lowering the statement-position `?` already produces.

```goal
func g() Result[int, error] {
    return Result.Ok(1)
}

func f(x int) int {
    return x + 1
}

func use() Result[int, error] {
    y := f(g()?)
    return Result.Ok(y)
}
```

```go
func g() (ok int, err error) {
	return 1, nil
}

func f(x int) int {
	return x + 1
}

func use() (ok int, err error) {
	q, err := g()
	if err != nil {
		return ok, err
	}
	y := f(q)
	return y, nil
}
```

### 2.5 Evaluation order preserved (`examples/qprop_nested_multi`)

When an operand evaluated *before* a nested `?` has a side effect (it calls a
function), it is bound to a temp first, so hoisting the `?` never reorders effects.
For `add(plain(), g()?)`, `plain()` is bound before the `g()?` temp — source order
is preserved. Pure operands (literals, `x + 1`, field reads) are left inline.

```goal
func plain() int {
    return 1
}

func g() Result[int, error] {
    return Result.Ok(2)
}

func add(a int, b int) int {
    return a + b
}

func use() Result[int, error] {
    y := add(plain(), g()?)
    return Result.Ok(y)
}
```

```go
func plain() int {
	return 1
}

func g() (ok int, err error) {
	return 2, nil
}

func add(a int, b int) int {
	return a + b
}

func use() (ok int, err error) {
	q := plain()
	q1, err := g()
	if err != nil {
		return ok, err
	}
	y := add(q, q1)
	return y, nil
}
```

---

## 3. Lowering rules (the general algorithm)

### 3.1 Mode selection

`?`'s mode is the **enclosing function's return type**: `Result[_, error]` → Result mode;
`Option[_]` → Option mode. The transpiler records each function's body span and mode while
scanning signatures, then maps each `?` (by source offset) to its function. A `?` outside a
Result/Option function is a located error (open-`E` only here).

### 3.2 Result mode

The enclosing function's signature is rewritten to **named returns** `(__goal_ok T, __goal_err error)`
(reused from 03-result), so the early return is `return __goal_ok, __goal_err` for any `T` with no
zero literal needed.

- `name := expr?` → `name, __goal_err := expr` then `if __goal_err != nil { return __goal_ok, __goal_err }`.
  Reusing `__goal_err` across several `?` is valid Go: each `:=` introduces a new `name`, so the
  short declaration redeclares the (param-scoped) `__goal_err` — exactly the spec's `cfg, err := ...`
  pattern.
- `_ := expr?` → the if-init form `if _, __goal_err := expr; __goal_err != nil { ... }`, scoping
  `__goal_err` to the `if` so repeated discards never collide and nothing is unused.

### 3.3 Option mode

The enclosing function returns `*T` (Option `[T]` rewritten by the 04 path). `None` propagates as
`nil`.

- `name := expr?` → `__goal_oN := expr` then `if __goal_oN == nil { return nil }` then
  `name := *__goal_oN`. Each `?` uses a **fresh** `__goal_oN` (monotonic counter) so multiple `?` in
  one block do not redeclare the pointer temp.
- `_ := expr?` → the if-init form `if __goal_oN := expr; __goal_oN == nil { return nil }` (value
  discarded).

### 3.4 Statement position

`?` is recognized only as the final token of a `lhs := expr?` statement (the surface rule, §2). The
whole statement line is replaced. Inline/sub-expression `?` is out of scope.

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| **`Result`/`Option` sum wrapper** | **Erased** into `(T, error)` / `*T` | Open-`E` `?` has no runtime sum value — the keystone native pair / pointer carry success-or-failure. |
| **The propagation control flow** | **Preserved** | The early-return-on-failure / unwrap-on-success *is* the program's behavior — preserved exactly as `if err != nil { return … }` / `if p == nil { return nil }`. |
| **The unwrapped value** | **Preserved** | Real runtime data, bound to `name` (or discarded via `_`). |

No defensive `panic`: `?` lowers to a total two-way branch on `err`/`nil` — no proven-unreachable
point. (The defensive panic belongs to exhaustive enum `match`, feature 02.)

---

## 5. Strategy forks (§8.3, §8.7)

| Case | Strategy | Status |
|---|---|---|
| Open-`E` Result `?` | native `(T, error)` + `if err != nil` | **implemented** |
| Option `?` | pointer `*T` + nil-check early return | **implemented** |
| Closed-`E` Result `?` | sum encoding + type-switch-and-return with a `From`-conversion in the `Err` arm | **feature 06** (out of scope) |
| Nested `?` in expression position (`f(g()?)`, `g()? + 1`, `return Ok(f(g()?))`) | hoist onto its own statement (§2.4–2.5) | **implemented** |
| Stored Result/Option | sum encoding | deferred |

Closed-`E` and open-`E` `?` are the **same** mechanism with/without a conversion step (§3.3
line-to-protect); only the open path is built here.

---

## 6. Hygiene

Synthesized names use the `__goal_` prefix (§8): `__goal_ok` / `__goal_err` (Result named returns and
the per-`?` error), `__goal_oN` (per-`?` Option pointer temp, monotonic). User identifiers, the
copied `T`, and the operand expressions are otherwise untouched.
