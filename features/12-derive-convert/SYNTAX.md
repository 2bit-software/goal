# 12-derive-convert — type-directed, completeness-checked struct conversion

> **Origin.** Not from `goal-design-spec.md`. This feature comes from the design exploration in this
> session: auditing the `telegraph/public-api` 3-layer codebase (transport/proto ↔ domain ↔
> persistence/boiler) and its goverter usage. The audit showed the friction is **not** field-name
> boilerplate (1:1 fields are the minority) but the *repeating type-pair conversions* — UUIDs,
> three optionality representations, timestamps, int widths, JSON blobs, enums — and the lack of a
> **completeness guarantee** (a forgotten field is a silent zero value). See `DECISIONS.md` §12.

## What it is

Two related constructs:

1. **Leaf conversion (the registry).** A named `from func` declaring how to turn one type into
   another. Generalizes feature 06's `from func` (which was scoped to error types for `?`) to *any*
   type pair. Every in-scope `from func` is part of the conversion registry, indexed by its
   `(source type → target type)` signature.

2. **Derived conversion.** A `derive func` whose body the compiler fills **field-by-field**,
   resolving each target field through the registry, and **erroring if any target field cannot be
   sourced** — the no-zero-value completeness posture (feature 08) applied to conversion. A
   forgotten field is a located compile error, not a silent zero.

## Final surface syntax

### Leaf conversions (registry) — three tiers

Tier is encoded in the **return type**:

```goal
// 1. lossless-total — cannot fail
from func uuidToString(u puid.UUID) string { return u.String() }

// 2. invariant-checked total — narrowing; asserts the invariant, panics on the impossible
//    (loud-but-local, NO Result ripple). Default tier for an ambiguous narrowing.
from func intToInt32(i int) int32 {
    assert i >= math.MinInt32 && i <= math.MaxInt32, "int32 overflow: %d", i
    return int32(i)
}

// 3. recoverable-fallible — expected bad input; returns Result, propagated by `?`
from func parseUUID(s string) Result[puid.UUID, error] { return puid.Parse(s)? }
```

The tier is a **semantic statement**: *is a failure here a bug (tier 2, assert) or expected bad
input (tier 3, Result)?* It is decided once per type-pair in the shared registry (~15 decls for a
whole codebase), not per entity.

### Derived conversions

```goal
// bodyless — derive every field. Sugar for `{ return StoredEvent{ ...derive(e) } }`.
derive func toStorage(e EventExecution) StoredEvent

// fallible derived conversion — return type carries Result (per feature 03); the deriver
// threads `?` through any tier-3 field conversions.
derive func fromStorage(s StoredEvent) Result[EventExecution, error]

// with exceptions — a returned partial literal; `...derive(src)` fills the rest, checked.
derive func toStorage(e EventExecution) StoredEvent {
    return StoredEvent{
        ExternalID: e.ID.String(),   // explicit (and rename): verbatim target-typed expression
        DeletedAt:  _,               // _ = intentionally skipped (left zero)
        ...derive(e)                 // remaining target fields, registry-resolved + completeness-checked
    }
}
```

- **`...derive(src)`** is to derived conversion what `...defaults` (feature 08) is to zero-
  construction: it fills the *unmentioned* target fields. Each is sourced from the same-named field
  of `src` (case-insensitive), converted via the registry; an unbridged or unsourced field is a
  **located error**.
- **`Field: expr`** is an explicit override, emitted verbatim — `expr` must already be the target
  field's type (it is the escape hatch; the author writes any needed conversion). A **rename that
  also needs a conversion** is written as an explicit expression (e.g. `ExternalID: e.ID.String()`),
  since the literal form has no separate rename operator — the registry auto-conversion applies only
  to same-named fields filled by `...derive`.
- **`Field: _`** marks a target field intentionally unset (the deriver leaves it zero and the
  completeness check is satisfied) — e.g. boiler's `R`, `L`, `DeletedAt`.

### Grammar

```ebnf
LeafConv   = "from" "func" Name "(" Param ")" ReturnType Body .
DeriveDecl = "derive" "func" Name "(" Param ")" ReturnType [ DeriveBody ] .
ReturnType = Type | "Result" "[" Type "," Type "]" | "(" Type "," Type ")" .
DeriveBody = "{" "return" CompositeLit "}" .
CompositeLit = TypeName "{" { Override "," } "...derive" "(" ident ")" [ "," ] "}" .
Override   = FieldName ":" ( Expression | "_" ) .
```

A bodyless `DeriveDecl` is sugar for `{ return T{ ...derive(src) } }` (no overrides).

## Worked examples

1. **Total with exceptions** (`to_storage`): rename (`ExternalID: e.ID.String()`), skip
   (`Internal: _`), Option→null and UUID→string bridges auto-filled by `...derive(e)`.
2. **Fallible, fully derived** (`from_storage`): bodyless, returns `(EventExecution, error)`; the
   tier-3 `string → puid.UUID` parse threads the error.
3. **Container recursion** (`slice`): `[]UUID → []string` filled automatically from the registered
   `UUID → string` — the deriver recurses into the slice (built-in rule).

## Rationale, tied to the two principles

- **Feedback:** the dominant silent-error classes the audit found — a forgotten field reading back as
  zero, a `default:`-armed enum silently mapping to a fallback, `safeIntToInt32` silently clamping —
  all become **located errors**: completeness checking (a field with no conversion fails to compile),
  exhaustive `match` enum mappings (feature 02), and tier-2 `assert` narrowing (feature 10). Today's
  transport hop in the audited code has **zero** completeness guarantee; this gives it one.
- **Familiarity:** leaf conversions reuse `from func` (06); enum mappings reuse `match` (02);
  fallibility reuses `Result`/`?` (03/05); `...derive` mirrors `...defaults` (08); the exception
  literal mirrors ordinary keyed construction. The output is the boring field-by-field Go a developer
  would hand-write — and what goverter generates.
- **Spend justified:** the new surface is just `derive func` + `...derive` + tier discipline. The
  recount on `configurable_execution` cut its own footprint from ~10 funcs / ~140 lines of manual
  transport code + an 8-method goverter interface to **6 `derive` lines + 3 enum mappings**, with the
  type-pair plumbing in a ~15-line shared registry.

## Resolved design decisions (from the exploration)

- **Default narrowing tier = invariant-checked-total (`assert`)**, not `Result`. A silent-clamp
  replacement should fail loud-but-local unless the author explicitly opts into recoverable `Result`.
- **Container recursion is a built-in deriver rule** (`[]`, `map`, `Option[A]→Option[B]`, nested
  struct), not a user-written generic conversion — so the user writes only the leaf `A→B`.
- **`Option[T] ↔ *T` is a built-in generic bridge** (unconstrained; maps to a Go generic). All other
  optional bridges (`Option[T] ↔ null.String/Time/Int`) are concrete `from func`s.
- **Target-directed dispatch**: a conversion is chosen by `(source-field-type → target-field-type)`;
  the target type is always known from the destination field. **Concrete beats generic** on overlap.
- **One canonical conversion per ordered `(A,B)` pair** in the registry. A site needing a different
  behavior for the same pair calls a named conversion explicitly (not via `...derive`).

## Reserved (designed-in, not built)

- **User-facing generic `from func [A,B] where convert(A,B)`** — power-user constrained generics.
  v1 ships built-in container recursion + concrete + the one `Option[T]↔*T` bridge.
- **Refinement / range types** (`int<0..2³¹>`) that would make narrowing compile-provable — too
  heavy a type-system spend (same trap §4.3 refused for static asserts); tier-2 `assert` is the
  stand-in.

## Open against spec / scope notes

- **`json.RawMessage` blobs stay first-class opaque fields.** A registered blob↔blob (or blob↔string)
  conversion handles them; the feature does **not** force typing them. Completeness checks structural
  completeness of the conversion, never the *contents* of an opaque field — and that is correct, not
  a limitation. (A blob that genuinely has a schema *may* be modeled as a struct/sum type, at which
  point container recursion covers it — but that is the author's modeling choice.)
- **Surface uses `Result[T,error]`; examples use the lowered `(T, error)` tuple.** To keep this
  feature's reference transpiler standalone (no dependency on feature 03's Result lowering or 04's
  Option lowering), the examples write conversions in their already-lowered Go forms (`(T, error)`,
  `*string` for `Option[string]`, local `UUID`/`NullString` stand-ins) — the same self-containment
  discipline prior features used (e.g. `type Time = int64`).
