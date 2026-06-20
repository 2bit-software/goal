# 07-implements — Transpile to Go

Governing contract: spec §8.5 (pure-erasure features) and §8.0. `implements` is a **static
guarantee**: the checker proves the type satisfies the interface at the declaration site, then it is
**erased**. The only generated Go is the free, runtime-cost-free compile-time assertion that keeps
the output self-verifying (§8.5, recommended).

---

## 1. The lowering (§8.5)

| goal | Go |
|---|---|
| `type T struct implements X { … }` (T has value-receiver methods) | `var _ X = T{}` after the struct |
| `type T struct implements X { … }` (T has a pointer-receiver method) | `var _ X = (*T)(nil)` after the struct |

The clause is stripped from the declaration (leaving a plain Go struct) and the assertion is emitted
just after the struct's closing brace. The assertion is the *entire* added output — `implements`
itself produces no runtime behavior. Because
`var _ X = …` only compiles when the value's method set satisfies `X`, the assertion is itself the
shadow of the checker's proof: if the methods don't line up, Go rejects it (located at the
assertion, mirroring goal's declaration-site error).

---

## 2. Input → output pairs

### 2.1 Value receiver (`examples/value_recv`)

```goal
type Point struct implements Stringer { X int; Y int }
func (p Point) String() string { return "point" }
```

```go
type Point struct { X int; Y int }

var _ Stringer = Point{}

func (p Point) String() string { return "point" }
```

### 2.2 Pointer receiver (`examples/pointer_recv`)

```goal
type Counter struct implements Resetter { n int }
func (c *Counter) Reset() { c.n = 0 }
```

```go
type Counter struct { n int }

var _ Resetter = (*Counter)(nil)

func (c *Counter) Reset() { c.n = 0 }
```

`Reset` is a pointer-receiver method, so only `*Counter` satisfies `Resetter`; the assertion uses a
typed nil pointer so it compiles.

### 2.3 Qualified interface (`examples/qualified_iface`)

```goal
type Discard struct implements io.Writer {}
```

```go
type Discard struct{}

var _ io.Writer = Discard{}
```

---

## 3. Lowering rules

1. **Parse** the clause `type T struct implements X, Y { … }`: `T` is the declared type name; the
   interface list is the comma-separated (possibly qualified) text between `implements` and the
   struct body `{`.
2. **Receiver kind.** Scan the file for `func (recv *T) …`. If any method of `T` uses a pointer
   receiver, emit `var _ X = (*T)(nil)`; otherwise emit `var _ X = T{}`. (Only `*T`'s method set
   includes pointer-receiver methods, so the value form would not satisfy `X` then.)
3. **Strip** the clause (`struct implements X, Y {` → `struct {`) and **emit** one assertion per
   interface immediately after the struct's closing brace. Everything else is passed through and
   `go/format`-ed.

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

Note the cross-feature fork on `implements` itself: a **sealed** interface in the list (feature 01)
lowers to the unexported marker method (closing the set); an **ordinary** interface (this feature)
lowers to the erased assertion. Same clause, two lowerings — the §3.4/§2 shared capability. A single
clause may list both, and each interface is lowered independently.

---

## 6. Hygiene

No temporaries are synthesized; the only emitted name is the blank identifier `_` in the assertion.
User code is otherwise untouched.
