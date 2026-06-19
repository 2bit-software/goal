# 01-enums — Syntax

Closed sum types (real enums): a closed tagged union with per-variant data. The complete variant
set is known to the compiler and is **not externally extensible** — the declaring module owns the
full set, permanently (spec §2.1). This is the spine: `match`, `Result`, and `Option` all reuse
the encoding it lowers to (§8.1).

This document pins the **surface syntax**. The Go it lowers to is in `TRANSPILE.md`.

---

## 1. Final surface syntax

### 1.1 Single-block form (the common path)

```goal
enum Status {
    Pending
    Active { since: Time }
    Cancelled { reason: string, at: Time }
}
```

- The keyword `enum` is itself the **closedness marker** — an `enum` block is self-evidently a
  closed set, co-located in one place.
- **Variants are newline-separated**, no trailing punctuation (Go block idiom — struct fields,
  `const` blocks).
- A **data-less variant** is a bare identifier (`Pending`).
- A variant **payload** is a brace block of `name: Type` fields, comma-separated on one line
  (`Active { since: Time }`), or across lines for many fields. Field names are lowercase `goal`
  identifiers; the type is any type expression.

### 1.2 Construction

```goal
p := Status.Pending
a := Status.Active(since: now())
c := Status.Cancelled(reason: "timeout", at: now())
```

- **Qualified** by the enum name (`Status.`) — disambiguates variants that share a name across
  enums and keeps the variant visually tied to its enum.
- **Call syntax with labeled arguments** (`since:`) — labels are the payload field names. Labels
  convert "wrong argument order" into a located error.
- A **data-less variant** is constructed by naming it with no call: `Status.Pending` (no `()`).

### 1.3 Sealed-interface form (the opt-in path)

For cases where variants must be **independently-existing types** (standalone structs reused
elsewhere), the same closed set is expressed with a `sealed interface` plus per-variant
`implements`:

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
```

- `sealed interface Shape {}` declares a **closed contract**: the compiler gathers every
  `implements Shape for ...` in the module and that set is frozen — no external package may add a
  satisfier. `sealed` is the closedness marker here (the `enum` keyword plays that role in the
  single-block form).
- `implements Shape for Circle` is the assertion "Circle is a variant of Shape." It reuses the
  **same `implements` mechanism** as the open-contract assertion in feature 07 (`implements
  io.Writer for JSONWriter`); the only difference is whether the target interface is `sealed`.
- Variants in this form are **ordinary Go-shaped struct types** and are constructed as ordinary
  composite literals (`Circle{Radius: 1}`), because they are standalone types — there is no
  `Shape.Circle(...)` sugar in this form. This asymmetry with the single-block form is deliberate:
  the sealed form is the escape hatch for "I need real, separately-usable types," so it speaks
  plain Go for the type and its construction.

**Both forms lower to the same Go encoding** (§8.1; see `TRANSPILE.md`). Prefer the single-block
form as the common path (spec §2.6).

---

## 2. Grammar

EBNF fragment (`{ x }` = zero-or-more, `[ x ]` = optional). It nests inside Go-shaped top-level
declarations — an `EnumDecl` / `SealedDecl` / `ImplementsDecl` is a new kind of top-level
declaration alongside Go's `type` / `func` / `var`.

```ebnf
EnumDecl       = "enum" identifier "{" { Variant } "}" .
Variant        = identifier [ "{" FieldList "}" ] .          (* newline-separated *)
FieldList      = Field { "," Field } [ "," ] .
Field          = identifier ":" Type .

SealedDecl     = "sealed" "interface" identifier "{" "}" .
ImplementsDecl = "implements" identifier "for" identifier .

Construction   = identifier "." identifier [ "(" [ ArgList ] ")" ] .
ArgList        = Arg { "," Arg } [ "," ] .
Arg            = identifier ":" Expression .

Type           = (* any Go type expression — passed through verbatim *) .
Expression     = (* any Go expression *) .
```

Notes:
- In `Construction`, the leading `identifier . identifier` is recognized as a variant
  construction **only when** the first identifier names an enum and the second names one of its
  variants; otherwise it is an ordinary selector and is left alone.
- A variant with no `"{" FieldList "}"` is data-less; its `Construction` has no `"(" ... ")"`.

---

## 3. Worked examples

### 3.1 Common case — payloads + labeled construction + a data-less variant

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

### 3.2 Awkward edge — every variant is data-less

A closed enum with no payloads at all (the case `iota` constants usually cover, but closed and
exhaustiveness-checkable):

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

### 3.3 Sealed-interface form — standalone variant types

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

---

## 4. Rationale (tied to the two principles)

Every divergence from Go-shape is justified by landing on a **widely-seen idiom** (familiarity
principle, §7) and/or converting a **silent error class** into a located error (feedback
principle).

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| `enum NAME { ... }` block | Rust/Swift/Scala `enum`; Go's own block-decl shape | "open `iota` enum can't be exhaustiveness-checked" → closed set enables exhaustiveness | New keyword `enum`; cheap — it reads exactly like the enums those languages already have |
| Newline-separated variants | Go `const`/struct-field blocks | "forgot a comma" churn avoided | None — this *is* Go's block idiom |
| `Variant { field: Type }` payload | Rust struct-variants / Swift associated values / TS object types | "which positional field is which" → named fields | Small: `{ }` + `:` rather than Go's `name Type` — buys self-documenting payloads and labeled construction |
| `Status.Active(since: now())` labeled call | constructor-call shape; labels echo Swift/Python keyword args | "wrong argument order at construction" → labeled, order-independent | Small: labeled call args are not Go, but they make the payload contract explicit at the call site |
| `Status.Pending` (no parens, data-less) | enum-member access (Swift `.pending`, Java `Status.PENDING`) | n/a (ergonomic) | None notable |
| `sealed interface` + `implements ... for` | Kotlin/Scala/Java `sealed`; reuses goal's own `implements` | "open interface silently breaks exhaustiveness when an outside package adds an implementor" → closed, enforced satisfier set | `sealed` keyword; justified below |

**Conventional names are preserved** (§7): this feature introduces no `Some`/`None`/`Ok`/`Err`/`?`
/`=>`/`_` — those belong to later features and are not Go-ified here.

### Why `sealed` is not redundant with `implements` (familiarity-budget justification)

`implements X for Y` asserts *"Y satisfies X"* and nothing more; it says nothing about whether
X's satisfier set is closed. The two concerns are **orthogonal**, and they come apart in exactly
the case feature 07 exists for: `implements io.Writer for JSONWriter` is a useful assertion over
an **open** contract — anyone may add another `io.Writer`, and `match` over `io.Writer` must
**not** be exhaustiveness-checked. Without a marker on the interface, the compiler cannot tell an
open contract (don't check, don't restrict implementors) from a closed enum (check exhaustively,
forbid outside implementors). `sealed` is precisely that one marker. It is the bit that (a)
forbids external implementors — without it Go's structural typing leaves the set open and any
consumer package silently breaks exhaustiveness the moment it adds a variant (§2.2) — and (b)
licenses `match`'s `panic("unreachable")` default, which is only sound because the set is provably
complete. The keyword spend is justified: it buys the entire reason enums exist. (User confirmed
`sealed` for now; revisit if a different word is preferred.)

---

## 5. Resolved open questions

This feature owns no §9 open question that bears on syntax. (`From`-conversion, explicit-defaults,
switch-coexistence, and the `E` lint default belong to features 05/06, 08, and 02 respectively.)

The closedness commitment itself (§2.6 asks it be decided at spec level) is resolved here for the
two surface forms: **closedness is signalled by an explicit keyword** — `enum` in the single-block
form, `sealed` in the standalone-type form — never inferred from "all implementors happen to be
local." Inference-from-absence is rejected: it is non-local and silently breaks exhaustiveness
when a satisfier is added elsewhere with no declared intent and no located error.

---

## 6. Open against spec

None. The spec's illustrative sample (§2.5) used Go-style positional payloads
(`Active(since Time)`) and the single-block form; this audit adopts brace-named payloads
(`Active { since: Time }`) per the user's choice and pins the sealed-interface form's surface
(`sealed interface` + `implements ... for`), which §2.6 left unspecified. Both lower to the
encoding the spec mandates in §8.1, so no spec semantics change.
