# 05-question-prop — Transpile to Go

Governing contract: spec §8.3 (`Result` and `?` dual strategy) and §3.7. This feature lowers the
**open-`E`** `?`: `?` becomes the `if err != nil` block the model was trained on (Result) or the
nil-check early return (Option). It composes the 03-result and 04-option lowerings for the
signatures and constructions the `?` sites depend on.

---

## 1. The lowering (§8.3)

**Result `?`** (enclosing function returns `Result[_, error]` → named `(__gop_ok T, __gop_err error)`):

| goal | Go |
|---|---|
| `name := expr?` | `name, __gop_err := expr` + `if __gop_err != nil { return __gop_ok, __gop_err }` |
| `_ := expr?` | `if _, __gop_err := expr; __gop_err != nil { return __gop_ok, __gop_err }` |

**Option `?`** (enclosing function returns `Option[_]` → `*T`):

| goal | Go |
|---|---|
| `name := expr?` | `__gop_oN := expr` + `if __gop_oN == nil { return nil }` + `name := *__gop_oN` |
| `_ := expr?` | `if __gop_oN := expr; __gop_oN == nil { return nil }` |

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
func loadConfig(p string) (__gop_ok Config, __gop_err error) {
	raw, __gop_err := readFile(p)
	if __gop_err != nil {
		return __gop_ok, __gop_err
	}
	cfg, __gop_err := parse(raw)
	if __gop_err != nil {
		return __gop_ok, __gop_err
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
	__gop_o1 := find(name)
	if __gop_o1 == nil {
		return nil
	}
	u := *__gop_o1
	__gop_o2 := parent(u.Name)
	if __gop_o2 == nil {
		return nil
	}
	p := *__gop_o2
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
func sync() (__gop_ok int, __gop_err error) {
	if _, __gop_err := flush(); __gop_err != nil {
		return __gop_ok, __gop_err
	}
	return 1, nil
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

The enclosing function's signature is rewritten to **named returns** `(__gop_ok T, __gop_err error)`
(reused from 03-result), so the early return is `return __gop_ok, __gop_err` for any `T` with no
zero literal needed.

- `name := expr?` → `name, __gop_err := expr` then `if __gop_err != nil { return __gop_ok, __gop_err }`.
  Reusing `__gop_err` across several `?` is valid Go: each `:=` introduces a new `name`, so the
  short declaration redeclares the (param-scoped) `__gop_err` — exactly the spec's `cfg, err := ...`
  pattern.
- `_ := expr?` → the if-init form `if _, __gop_err := expr; __gop_err != nil { ... }`, scoping
  `__gop_err` to the `if` so repeated discards never collide and nothing is unused.

### 3.3 Option mode

The enclosing function returns `*T` (Option `[T]` rewritten by the 04 path). `None` propagates as
`nil`.

- `name := expr?` → `__gop_oN := expr` then `if __gop_oN == nil { return nil }` then
  `name := *__gop_oN`. Each `?` uses a **fresh** `__gop_oN` (monotonic counter) so multiple `?` in
  one block do not redeclare the pointer temp.
- `_ := expr?` → the if-init form `if __gop_oN := expr; __gop_oN == nil { return nil }` (value
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
| Inline `?` (`g(f()?)`) / stored Result/Option | hoist / sum encoding | deferred |

Closed-`E` and open-`E` `?` are the **same** mechanism with/without a conversion step (§3.3
line-to-protect); only the open path is built here.

---

## 6. Hygiene

Synthesized names use the `__gop_` prefix (§8): `__gop_ok` / `__gop_err` (Result named returns and
the per-`?` error), `__gop_oN` (per-`?` Option pointer temp, monotonic). User identifiers, the
copied `T`, and the operand expressions are otherwise untouched.
