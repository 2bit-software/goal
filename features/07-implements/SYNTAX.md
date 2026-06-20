# 07-implements — Syntax

Explicit `implements` (spec §3.4): a struct may **declare** it satisfies (at least) an interface.
The declaration is **additive** — it does not preclude satisfying other interfaces structurally,
and it does **not** convert goal to nominal typing. It converts Go's invisible, unchecked-at-the-
declaration interface satisfaction into a **located compile error at the struct** when a method is
missing or mis-signed.

This document pins the surface syntax (inherited from feature 01). The Go it lowers to is in
`TRANSPILE.md`.

---

## 1. Final surface syntax

```goal
type JSONWriter struct implements io.Writer {   // asserts at least io.Writer; still free to satisfy others
    ...
}

func (w JSONWriter) Write(p []byte) (int, error) { ... }
// if Write were missing or mis-signed, the error points HERE, at the declaration
```

- **`type T struct implements X, Y { … }`** — an inline clause on the struct declaration, between
  `struct` and the body `{`. Each interface in the comma-separated list is asserted; `X`/`Y` may be
  qualified (like `io.Writer`) or local.
- **Additive, not a gate.** Structural satisfaction remains the default everywhere; `implements`
  only *adds* a checkable assertion at the declaration site.
- **Structs only, for now.** Only struct types carry the clause today. Extending it to any concrete
  type, as Go allows (e.g. `type Celsius float64 implements Stringer`), is future work.

### 1.1 Shared with feature 01

The same inline clause carries the sealed-interface assertion (feature 01,
`type Active struct implements Status { … }`) — one `implements` spelling for both roles, per
§2/§3.4 ("shares its core capability with the closed-enum mechanism").

The difference is purely in lowering: when an interface in the list is **sealed** (feature 01), it
contributes the unexported marker method that closes the set; when it is an **ordinary** interface
(this feature), it lowers to the erased-but-asserted form below. A single clause may mix both.

---

## 2. Grammar

```ebnf
StructType     = "struct" [ ImplementsClause ] StructBody .
ImplementsClause = "implements" InterfaceType { "," InterfaceType } .
```

The clause sits between `struct` and the struct body. Each `InterfaceType` may be qualified
(`io.Writer`); the asserted type is the local type being declared.

---

## 3. Worked examples

### 3.1 Value receiver, local interface (`examples/value_recv`)

```goal
type Stringer interface { String() string }
type Point struct implements Stringer { X int; Y int }

func (p Point) String() string { return "point" }
```

### 3.2 Pointer receiver (`examples/pointer_recv`)

```goal
type Resetter interface { Reset() }
type Counter struct implements Resetter { n int }

func (c *Counter) Reset() { c.n = 0 }
```

### 3.3 Qualified interface (`examples/qualified_iface`)

```goal
type Discard struct implements io.Writer {}

func (Discard) Write(p []byte) (int, error) { return len(p), nil }
```

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| `type T struct implements X { … }` clause | a declared interface assertion on the type (cf. Java `class T implements X`, but additive) | "struct satisfies an interface invisibly; a wrong signature surfaces at a distant call site or never" → located error at the struct | **Additive (§7, cheap)** — adds a checkable assertion without changing what any Go construct means |
| Additive, not nominal | Go's structural typing (unchanged) | n/a | None — structural satisfaction stays the default |

This is one of the best value/friction ratios on the list (§3.4): near-zero friction, near-zero
divergence.

---

## 5. Resolved open questions

None specific to this feature. The §3.4 note that the syntax "could equally be an annotation on the
type" is realized: the assertion is an inline clause on the type declaration,
`type T struct implements X { … }`, shared with feature 01.

---

## 6. Open against spec

None. The spec §3.4 sample interface assertion (`io.Writer` on `JSONWriter`) is expressed as the
inline clause `type JSONWriter struct implements io.Writer { … }`. The lowering is the §8.5
erase-and-assert form.
