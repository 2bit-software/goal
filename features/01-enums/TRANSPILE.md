# 01-enums — Transpile to Go

Governing contract: spec §8.1 (closed sum types) and §8.0 (erase checks, preserve runtime
semantics). Both surface forms (single-block `enum`, `sealed interface` + `implements`) target the
**same** Go encoding.

---

## 1. The encoding (§8.1)

A closed sum type lowers to **a sealed interface + one struct per variant + an unexported marker
method**:

```go
type NAME interface{ isNAME() }       // sealed: unexported marker keeps it closed in Go too

type NAME_Variant struct{ ...fields } // one per variant; empty struct if data-less
...
func (NAME_Variant) isNAME() {}       // marker impl per variant
```

- The **unexported** `isNAME()` is what makes the set genuinely closed in the generated Go: no
  other package can spell that method, so no other package can add a variant — mirroring goal's
  guarantee (§8.1).
- **Per-variant structs**, each carrying exactly its own payload (not one all-fields struct).
- **Data-less variants → empty structs** (`struct{}`). (Interning a shared value is a later
  optimization; not done here.)

---

## 2. Input → output pairs

### 2.1 Common case (`examples/status`)

```goal
enum Status {
    Pending
    Active { since: Time }
    Cancelled { reason: string, at: Time }
}

func examples() {
    p := Status.Pending
    a := Status.Active(since: now())
    c := Status.Cancelled(reason: "timeout", at: now())
    _, _, _ = p, a, c
}
```

```go
type Status interface{ isStatus() }

type Status_Pending struct{}
type Status_Active struct {
	Since Time
}
type Status_Cancelled struct {
	Reason string
	At     Time
}

func (Status_Pending) isStatus()   {}
func (Status_Active) isStatus()    {}
func (Status_Cancelled) isStatus() {}

func examples() {
	p := Status(Status_Pending{})
	a := Status(Status_Active{Since: now()})
	c := Status(Status_Cancelled{Reason: "timeout", At: now()})
	_, _, _ = p, a, c
}
```

### 2.2 All-data-less (`examples/traffic`)

```goal
enum Light {
    Red
    Yellow
    Green
}

func first() Light {
    return Light.Red
}
```

```go
type Light interface{ isLight() }

type Light_Red struct{}
type Light_Yellow struct{}
type Light_Green struct{}

func (Light_Red) isLight()    {}
func (Light_Yellow) isLight() {}
func (Light_Green) isLight()  {}

func first() Light {
	return Light(Light_Red{})
}
```

### 2.3 Sealed-interface form (`examples/shape`)

```goal
type Circle struct {
    Radius float64
}
type Rectangle struct {
    Width  float64
    Height float64
}

sealed interface Shape {}

implements Shape for Circle
implements Shape for Rectangle

func unit() Shape {
    return Circle{Radius: 1}
}
```

```go
type Circle struct {
	Radius float64
}
type Rectangle struct {
	Width  float64
	Height float64
}

type Shape interface{ isShape() }

func (Circle) isShape()    {}
func (Rectangle) isShape() {}

func unit() Shape {
	return Circle{Radius: 1}
}
```

---

## 3. Lowering rules (the general algorithm)

### 3.1 Single-block `enum NAME { ... }`

1. Emit the sealed interface: `type NAME interface{ isNAME() }`. The marker name is `is` + the
   enum name.
2. For each variant `V`:
   - **data-less** → `type NAME_V struct{}`.
   - **with payload** `{ f1: T1, f2: T2 }` → `type NAME_V struct { F1 T1; F2 T2 }`, where each Go
     field name `Fi` is the goal field name with its **first letter capitalized** (so the field is
     exported), and the type `Ti` is the goal type expression **passed through verbatim**.
3. For each variant `V`, emit the marker impl: `func (NAME_V) isNAME() {}`.

### 3.2 Construction `NAME.V(...)` / `NAME.V`

Recognized only when `NAME` is a declared enum and `V` one of its variants.

- **data-less** `NAME.V` → `NAME(NAME_V{})`.
- **with args** `NAME.V(label1: e1, label2: e2)` → `NAME(NAME_V{Label1: e1, Label2: e2})`, where
  each label is capitalized to match the exported Go field and each `ei` is the argument
  expression **passed through verbatim**. Argument order is preserved as written.

The outer `NAME(...)` conversion gives the value the **interface type** `NAME` (so it is usable
anywhere a `NAME` is expected), exactly per §8.1.

### 3.3 Sealed-interface form

1. `sealed interface NAME {}` → `type NAME interface{ isNAME() }` (identical interface to §3.1).
2. `implements NAME for T` → `func (T) isNAME() {}` (attach the marker to the existing standalone
   type `T`; the marker name is `is` + the interface name).
3. The standalone `type T struct { ... }` declarations and their composite-literal constructions
   (`T{...}`) are **ordinary Go** and pass through untouched. Because they now have the `isNAME`
   method, they satisfy `NAME`.

### 3.4 Everything else

Package clause, imports, plain `type`/`func`/`var`, comments, and any expression that is not an
enum-variant construction are **passed through verbatim**. Output is run through `go/format`, so
it is gofmt-clean regardless of the spacing the splicer produced.

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| **Closedness / exhaustiveness-readiness** | **Erased** as a *check* — but the **unexported marker method is preserved** | The checker already proved the set is closed; the generated Go does not re-verify it. The marker is kept because it *encodes* the closedness in Go's own type system (no other package can implement it), keeping the output honest against hand-edits and separate packages (§8.1). It is not a runtime check, it is a structural guarantee. |
| **Variant identity (the tag)** | **Preserved** at runtime | Which variant a value is *is* runtime data — it is the concrete struct type behind the interface, observed later by `match` (feature 02). The encoding must carry it. |
| **Payload values** | **Preserved** at runtime | The fields are real data the program uses. |
| **Field/variant *names*** | **Preserved** (as exported Go identifiers) | They are how `match` arms and field accesses refer to the data downstream. |

**No defensive `panic` is emitted by this feature.** The erasure-with-defensive-panic rule (§8.0)
applies where the checker *proves a point unreachable* — that first arises in `match`'s exhaustive
default (feature 02). Enum **declaration and construction** prove no unreachability, so there is no
unreachable point to guard. (This is noted so the absence is deliberate, not an oversight.)

---

## 5. Strategy forks

This feature has **one** encoding and therefore one lowering — the sum-type encoding of §8.1,
which §8.0 calls "the universal fallback." There is **no** immediate-vs-stored fork here: that
fork (§8.7) is a property of how a `Result`/`Option` *value* is consumed, and it selects *between*
the native-tuple/pointer strategy and *this* sum encoding. Enums always use this encoding; they
are the encoding the fork falls back to. Features 03/04 own that fork.

The only branch within this feature is **single-block vs sealed-interface form**, and it is not a
strategy fork in the §8.7 sense — both forms converge on the identical Go encoding. The selector
is purely syntactic: `enum { ... }` synthesizes `NAME_V` structs; `sealed interface` + standalone
types attach the marker to the user's own types and emit no synthesized structs.

---

## 6. Hygiene

This feature synthesizes no temporaries (no `:=` temporaries, no intermediate locals), so the
`__gop_` prefix (§8) does not appear in its output. The synthesized **type** names use the
`NAME_Variant` form mandated by §8.1, not the temporary prefix. Later features that introduce
locals (e.g. `match` bindings, `?` desugaring) own the `__gop_` hygiene.
