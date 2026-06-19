# 07-implements — Transpile to Go

Governing contract: spec §8.5 (pure-erasure features) and §8.0. `implements` is a **static
guarantee**: the checker proves the type satisfies the interface at the declaration site, then it is
**erased**. The only generated Go is the free, runtime-cost-free compile-time assertion that keeps
the output self-verifying (§8.5, recommended).

---

## 1. The lowering (§8.5)

| goal | Go |
|---|---|
| `implements X for T` (T has value-receiver methods) | `var _ X = T{}` |
| `implements X for T` (T has a pointer-receiver method) | `var _ X = (*T)(nil)` |

The assertion is the *entire* output — `implements` itself produces no runtime behavior. Because
`var _ X = …` only compiles when the value's method set satisfies `X`, the assertion is itself the
shadow of the checker's proof: if the methods don't line up, Go rejects it (located at the
assertion, mirroring goal's declaration-site error).

---

## 2. Input → output pairs

### 2.1 Value receiver (`examples/value_recv`)

```goal
implements Stringer for Point
func (p Point) String() string { return "point" }
```

```go
var _ Stringer = Point{}

func (p Point) String() string { return "point" }
```

### 2.2 Pointer receiver (`examples/pointer_recv`)

```goal
implements Resetter for Counter
func (c *Counter) Reset() { c.n = 0 }
```

```go
var _ Resetter = (*Counter)(nil)

func (c *Counter) Reset() { c.n = 0 }
```

`Reset` is a pointer-receiver method, so only `*Counter` satisfies `Resetter`; the assertion uses a
typed nil pointer so it compiles.

### 2.3 Qualified interface (`examples/qualified_iface`)

```goal
implements io.Writer for Discard
```

```go
var _ io.Writer = Discard{}
```

---

## 3. Lowering rules

1. **Parse** `implements X for T`: `X` is the (possibly qualified) interface text between
   `implements` and `for`; `T` is the type text after `for` to end of line.
2. **Receiver kind.** Scan the file for `func (recv *T) …`. If any method of `T` uses a pointer
   receiver, emit `var _ X = (*T)(nil)`; otherwise emit `var _ X = T{}`. (Only `*T`'s method set
   includes pointer-receiver methods, so the value form would not satisfy `X` then.)
3. **Replace** the whole `implements` declaration with the assertion. Everything else is passed
   through and `go/format`-ed.

The reference transpiler **emits** the assertion but does **not** verify the methods exist or match
(the checker's job, §8.5). If they don't, Go's own compiler rejects the emitted assertion — a
fortunate property: the erased check leaves behind a real, compilable proof obligation.

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| The `implements` assertion | **Erased** | A static guarantee — Go's structural typing already satisfies the interface at use sites; nothing needs to re-check it at runtime. |
| The free `var _ X = …` assertion | **Emitted** (compile-time only, zero runtime cost) | Keeps the output self-verifying and documents intent (§8.5, recommended); it is not runtime behavior. |
| The type and its methods | **Preserved** | Ordinary Go — passed through untouched. |

No defensive `panic`: `implements` introduces no control flow and no proven-unreachable point.

---

## 5. Strategy forks

The only branch is the **assertion target** — `T{}` (value receivers) vs `(*T)(nil)` (a
pointer-receiver method present) — selected by scanning the file's method receivers. There is no
relationship to the §8.7 immediate-vs-stored fork (that is a `Result`/`Option` concern).

Note the cross-feature fork on `implements` itself: with a **sealed** interface (feature 01) it
lowers to the unexported marker method (closing the set); with an **ordinary** interface (this
feature) it lowers to the erased assertion. Same surface, two lowerings — the §3.4/§2 shared
capability.

---

## 6. Hygiene

No temporaries are synthesized; the only emitted name is the blank identifier `_` in the assertion.
User code is otherwise untouched.
