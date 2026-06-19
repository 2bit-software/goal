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
type JSONWriter struct { ... }

implements io.Writer for JSONWriter   // asserts at least io.Writer; still free to satisfy others

func (w JSONWriter) Write(p []byte) (int, error) { ... }
// if Write were missing or mis-signed, the error points HERE, at the declaration
```

- **`implements X for T`** — a standalone top-level declaration. `X` is any interface (qualified
  like `io.Writer` or local); `T` is a local type.
- **Additive, not a gate.** Structural satisfaction remains the default everywhere; `implements`
  only *adds* a checkable assertion at the declaration site.

### 1.1 Inherited from feature 01

The `implements X for T` form was pinned in **feature 01** (the sealed-interface enum form,
`implements Status for Active`), chosen there over a suffix annotation
(`type T struct{…} implements X`). This feature reuses that exact form for the general additive
assertion over any interface — one `implements` spelling for both roles, per §2/§3.4 ("shares its
core capability with the closed-enum mechanism"). No new surface is introduced, so no separate
syntax choice was opened.

The difference is purely in lowering: when `X` is a **sealed** interface (feature 01), `implements`
contributes the unexported marker method that closes the set; when `X` is an **ordinary** interface
(this feature), it lowers to the erased-but-asserted form below.

---

## 2. Grammar

```ebnf
ImplementsDecl = "implements" InterfaceType "for" TypeName .
```

A top-level declaration alongside Go's `type`/`func`/`var`. `InterfaceType` may be qualified
(`io.Writer`); `TypeName` is the local type being asserted.

---

## 3. Worked examples

### 3.1 Value receiver, local interface (`examples/value_recv`)

```goal
type Stringer interface { String() string }
type Point struct { X int; Y int }

implements Stringer for Point

func (p Point) String() string { return "point" }
```

### 3.2 Pointer receiver (`examples/pointer_recv`)

```goal
type Resetter interface { Reset() }
type Counter struct { n int }

implements Resetter for Counter

func (c *Counter) Reset() { c.n = 0 }
```

### 3.3 Qualified interface (`examples/qualified_iface`)

```goal
implements io.Writer for Discard

func (Discard) Write(p []byte) (int, error) { return len(p), nil }
```

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| `implements X for T` declaration | a declared interface assertion (cf. Rust `impl Trait for T`, but additive) | "struct satisfies an interface invisibly; a wrong signature surfaces at a distant call site or never" → located error at the struct | **Additive (§7, cheap)** — adds a checkable assertion without changing what any Go construct means |
| Additive, not nominal | Go's structural typing (unchanged) | n/a | None — structural satisfaction stays the default |

This is one of the best value/friction ratios on the list (§3.4): near-zero friction, near-zero
divergence.

---

## 5. Resolved open questions

None specific to this feature. The §3.4 note that the syntax "could equally be an annotation on the
type" was settled in feature 01 in favor of the standalone `implements X for T` declaration.

---

## 6. Open against spec

None. The spec §3.4 sample is exactly `implements io.Writer for JSONWriter`. The lowering is the
§8.5 erase-and-assert form.
