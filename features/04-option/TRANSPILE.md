# 04-option — Transpile to Go

Governing contract: spec §8.4 (the `Option` pointer strategy) and §8.0 / §8.7. `Option` exists to
*eliminate* nil-deref yet transpiles *to* nil — correct, because the checker proved every access
goes through a `match`, so the generated nil-check is provably present. Safety lives in the source;
the output is its proven-safe shadow (§8.4).

---

## 1. The lowering (§8.4)

| goal | Go |
|---|---|
| `Option[T]` (type) | `*T` |
| `return Option.None` | `return nil` |
| `return Option.Some(v)` | `return &v` (v an identifier) / box to a temp otherwise |
| `match opt { Option.Some(x) => A; Option.None => B }` | `if __goal_o := opt; __goal_o != nil { x := *__goal_o; A } else { B }` |

`None` → `nil`; `Some(v)` → a non-nil pointer; the `match` becomes the nil-check the model already
knows.

---

## 2. Input → output pairs

### 2.1 Reference type (`examples/option_find`)

```goal
func find(id ID) Option[User] {
    u, ok := lookup(id)
    if !ok {
        return Option.None
    }
    return Option.Some(u)
}

func handle(id ID) {
    match find(id) {
        Option.Some(u) => greet(u)
        Option.None => prompt()
    }
}
```

```go
func find(id ID) *User {
	u, ok := lookup(id)
	if !ok {
		return nil
	}
	return &u
}

func handle(id ID) {
	if __goal_o := find(id); __goal_o != nil {
		u := *__goal_o
		greet(u)
	} else {
		prompt()
	}
}
```

### 2.2 Value type — boxing (`examples/option_int`)

```goal
func first(xs []int) Option[int] {
    if len(xs) == 0 {
        return Option.None
    }
    return Option.Some(xs[0])
}
```

```go
func first(xs []int) *int {
	if len(xs) == 0 {
		return nil
	}
	__goal_some := xs[0]
	return &__goal_some
}
```

`Option[int]` → `*int`. `xs[0]` is not a bare identifier, so it is boxed through a temp (§3.2).

### 2.3 `Some` arm ignores its binding (`examples/option_exists`)

```goal
func exists(id ID) bool {
    match find(id) {
        Option.Some(u) => return true
        Option.None => return false
    }
}
```

```go
func exists(id ID) bool {
	if __goal_o := find(id); __goal_o != nil {
		return true
	} else {
		return false
	}
}
```

The deref alias (`u := *__goal_o`) is omitted because the `Some` arm does not use `u` — no unused
variable.

---

## 3. Lowering rules (the general algorithm)

### 3.1 Type

`Option[T]` → `*T`, wherever the type appears. For **reference types** this is the natural pointer.
For **value types** (`Option[int]`) the pointer *boxes* the value (`*int`) — the v1 strategy (§8.4);
a non-allocating sum encoding is a later optimization.

### 3.2 Construction (return position)

- `return Option.None` → `return nil`.
- `return Option.Some(v)`:
  - if `v` is a bare **identifier** (addressable) → `return &v` (the spec's `Some(u) -> &u`);
  - otherwise (a literal, index, call, …) → box through a temp so the address is valid Go:
    `__goal_some := v` then `return &__goal_some`. (Go forbids `&5`, `&f()`; the temp is the idiomatic
    `v := …; return &v`.) Boxing also gives the Option its own copy, avoiding aliasing the source.

Recognized only in `return` position (the immediate case). A stored `Option.Some/None` is the §8.7
case and is out of scope.

### 3.3 Consumption — `match` on an Option

For `match <opt> { Option.Some(x) => A; Option.None => B }`:

1. Capture and nil-test in one `if`-init: `if __goal_o := <opt>; __goal_o != nil { … } else { … }`.
2. **`Some` → the non-nil branch; `None` → the `else`** (regardless of source order).
3. In the `Some` branch, if the arm binds and uses `x`, dereference: `x := *__goal_o` — so the inner
   value is reachable *only* inside this branch. If `x` is unused, the deref is omitted.
4. Arm bodies are emitted verbatim (the user's binding name `x` is preserved via the `:= *__goal_o`
   alias — no renaming needed).

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| **Must-destructure guarantee** | **Erased** | A static guarantee — it proved every access goes through a `match`. It produces no runtime behavior; the generated nil-check is the *shadow* of that proof, not a re-check. |
| **The `Option` sum wrapper** | **Erased into `*T`** | In the pointer strategy there is no runtime sum value — `None`/`Some` are `nil`/non-nil. The "sum type" exists only in the source's type system. |
| **The nil branch / control flow** | **Preserved** | Which branch runs (`Some` vs `None`) is runtime semantics — preserved exactly as `if p != nil` / `else`. |
| **The inner value** | **Preserved** | Real runtime data behind the pointer; reached via `*__goal_o`. |

No defensive `panic` arises: the Option `match` is a total two-way `if`/`else` over a pointer —
there is no proven-unreachable point to guard. (The defensive-panic rule applies to exhaustive enum
`match`, feature 02, and sum-encoded matches, feature 06.)

---

## 5. Strategy forks (§8.7)

| Case | Strategy | Status |
|---|---|---|
| Option consumed immediately (returned + match) | **pointer `*T`** (box value types) | **implemented (this feature)** |
| Option stored as a first-class value | pointer `*T` is itself storable; full stored/value-position handling | out of scope here — deferred |
| Value-type Option without allocation | sum encoding | later optimization (§8.4) |

The transpiler implements the first row, detecting the immediate case structurally (`Option[T]` as
a type; `Option.Some/None` in return position; `match` directly on an Option). Value-position
Option `match` (`x := match …`) is **deferred** with a located message.

---

## 6. Hygiene

Synthesized names use the `__goal_` prefix (§8): `__goal_o` (the captured pointer at a match site) and
`__goal_some` (the box for a non-addressable `Some` payload). User identifiers, the copied type `T`,
and arm bodies are otherwise untouched; the `Some` binding name is preserved as a real local alias.
