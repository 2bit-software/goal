# 12-derive-convert — transpile to Go

Governing model: §8.0 (erasure vs preservation) + the keystone discipline (§8.3) — emit the boring,
idiomatic Go a developer would hand-write.

## Erasure vs preservation

- **Erased (static guarantee):** the *completeness check* — "every target field is sourced." Like
  exhaustiveness (02), must-use (03), and field-completeness (08), the guarantee lives in the
  rejected programs and leaves no runtime trace.
- **Preserved (runtime):** the field-by-field assignment itself, and any tier-3 `Result`/error
  control flow (the `?`-threading) and tier-2 `assert` panics inside leaf conversions.
- **Stripped sugar:** `from` (leaf conversions lower to plain `func`, as in 06); `derive func` is
  replaced wholesale by the generated function; `...derive(src)` expands to explicit per-field
  assignments (the parallel of 08's `...defaults` expansion).

## Input → output pairs

### 1. Total derived conversion with exceptions

```goal
from func uuidToString(u UUID) string { return u.s }
from func ptrToNull(o *string) NullString { ... }

derive func toStorage(e EventExecution) StoredEvent {
    return StoredEvent{
        ExternalID: e.ID.String(),
        Internal:   _,
        ...derive(e)
    }
}
```
```go
func uuidToString(u UUID) string { return u.s }
func ptrToNull(o *string) NullString { ... }

func toStorage(e EventExecution) StoredEvent {
    var out StoredEvent
    out.ExternalID = e.ID.String()       // explicit override (verbatim)
    out.ID = uuidToString(e.ID)          // ...derive: registry UUID->string
    out.CustomerID = uuidToString(e.CustomerID)
    out.Railroad = e.Railroad            // ...derive: same type, direct
    out.TraceID = ptrToNull(e.TraceID)   // ...derive: registry *string->NullString
    return out                           // `Internal: _` => left zero, omitted
}
```

### 2. Fallible, fully derived

```goal
from func parseUUID(s string) (UUID, error) { ... }

derive func fromStorage(s StoredEvent) (EventExecution, error)
```
```go
func fromStorage(s StoredEvent) (EventExecution, error) {
    var out EventExecution
    __goal_v0, err := parseUUID(s.ID)
    if err != nil {
        return out, err
    }
    out.ID = __goal_v0
    // ... remaining fields ...
    return out, nil
}
```

### 3. Container recursion (slice)

```goal
from func uuidToString(u UUID) string { return u.s }
derive func toIDs(g Group) IDList   // Members []UUID -> Members []string
```
```go
func toIDs(g Group) IDList {
    var out IDList
    out.Members = make([]string, len(g.Members))
    for i := range g.Members {
        out.Members[i] = uuidToString(g.Members[i])
    }
    return out
}
```

## Lowering algorithm

Focused recognizer (`text/scanner`, span-splice, `go/format`); no full Go parser.

1. **Build the registry.** Scan `from func NAME(p Src) Ret`:
   - record `(Src → Tgt) ⇒ {name: NAME, fallible}`, where `fallible` is true iff `Ret` is
     `(Tgt, error)` (two results, second `error`).
   - strip the leading `from ` so the leaf becomes a plain Go `func` (reuses feature 06's technique).
2. **Collect struct fields** for every `type X struct {…}` (name + raw type, in order) — reuses
   feature 08's struct-body parser.
3. **Expand each `derive func`.** Parse its name, source param (name + type `S`), target type `T`,
   whether it is fallible (return `(T, error)`), and the optional body's overrides + `...derive(src)`.
   Emit a function:
   - `var out T`
   - **overrides first**, in written order: `Field: expr` → `out.Field = expr` (verbatim);
     `Field: _` → omitted (left zero).
   - **`...derive(src)` fields**, in `T`'s declared order, skipping any already named:
     for target field `F` of type `TF`, find the same-named (case-insensitive) source field `G` of
     type `SF` on `S`, then **resolve** `SF → TF` (below) and emit the assignment.
   - `return out` (total) or `return out, nil` (fallible).
4. **Resolve `SF → TF`:**
   - `SF == TF` → direct: `out.F = src.G`.
   - registry has `(SF → TF)` total → `out.F = NAME(src.G)`.
   - registry has `(SF → TF)` fallible → thread: `__goal_vN, err := NAME(src.G); if err != nil { return out, err }; out.F = __goal_vN` (requires the derived conversion to be fallible; else a located error).
   - **container recursion** (built-in): `SF = []A`, `TF = []B`, and `A → B` resolvable → emit a
     `make` + indexed loop applying the element resolution. (Map and nested-struct recursion follow
     the same rule; see scope.)
   - none of the above → **located error** (deferred, not silently skipped).
5. **Inline `...derive(src)` (return position).** The same expansion (steps 3–4) applies when a
   `...derive(src)` literal appears as the operand of a `return` in an ordinary function, not only
   in a `derive func` body: `return T{ overrides…, ...derive(src) }` (and `return T{ … }, nil` when
   the function returns `(T, error)`). It lowers to a scoping block `{ var out T; overrides; fills;
   return out[, nil] }` — the block lets `var out` shadow any user local of that name, and `src` is
   read off the enclosing function's parameter. The emitted conversion statements are identical to
   the equivalent `derive func`. Fallibility is taken from the return **shape** (a trailing `, nil`),
   and a `...derive` outside a return is a located error pointing at the `derive func` form.

## Erasure-with-defensive-feedback

Where a target field cannot be resolved, the reference transpiler emits a **located error** naming
the field and the missing `(SF → TF)` conversion — it never silently leaves the field zero (that
would resurrect the footgun the feature exists to kill). This mirrors the deferral discipline of
prior features (02/03/05/06 defer unhandled forms with a located message).

## Strategy forks

- **Tier (total / assert-total / fallible):** read from the leaf's return type (rule 1). Tiers 1–2
  are both *total* signatures, so they never make a derived conversion fallible; only a tier-3
  `(T,error)` leaf does. This is why the audited count fields (tier-2 narrowing) keep `toProto`
  total while only genuine parses (`fromStorage`) go fallible.
- **Dispatch:** target-directed `(SF → TF)`; concrete registry entry beats a built-in generic on
  overlap.

## Hygiene

Fallible threading uses `__goal_vN` temporaries (the `__goal_` prefix, per §8). `out` is the single
result accumulator (a local, so it dodges zero-literal synthesis the way feature 03 used named
returns).

## Foreign (out-of-package) source/target types

A `derive func` may convert to or from a struct declared in an imported Go package — the
common case being a generated protobuf message, e.g. `derive func FromProto(p
*hobv1.EnvironmentSpec) Spec`. The field set of such a type is not in the `.goal` source,
so the package driver enriches the name-keyed tables before lowering:

- `analyze.EnrichForeign` reads each `.goal` file's import block, finds the packages a
  `derive`/`from` references by qualifier, resolves each import path to a directory
  (same-module via `go.mod`, else `go list`), and parses that package's Go with stdlib
  `go/parser`. Every **exported** struct becomes `Tables.Structs["alias.Type"]`, with each
  field's type re-rendered qualified by the import alias (`*Workspace` → `*hobv1.Workspace`).
- The lowering and the checker strip a single leading pointer star before the lookup, so
  `*pkg.Message` resolves to its struct. A pointer **target** is built as a value and
  returned by address (`var out T; …; return &out`); a pointer **source** is read directly
  (Go auto-dereferences `p.Field`).
- This is enrichment, not type-checking — only the field set is read. Enrichment runs only
  at the **package** granularity (where a directory exists); a single-file `Transpile`
  stays foreign-blind and defers as before.

Both directions are supported. **Proto → internal** iterates the internal target's fields
(usually a curated subset, so fewer to source). **Internal → proto** iterates the proto
target's fields — every one must be sourced or overridden, so the larger, enum/oneof-laden
proto is the more demanding target.

**Foundation only.** Fields still resolve by the nominal model: same-named target field,
same type or a registered `from func` or a built-in recursion. Proto-specific shapes —
getter access, enum→sum, and oneof→sum — are **not** auto-bridged: an enum or oneof field
needs an explicit `Field: expr` override or a registered `from func`, exactly as any other
unbridged pair does. Unexported foreign fields (a proto's `state`/`sizeCache`) are never
read, so they never participate.

## Scope / not built (the checker's / a fuller frontend's job)

- **Completeness checking is not enforced as a type system** — the transpiler resolves what it can
  and defers the rest with a located message; it does not prove the registry is total. (Per the
  audit's no-checking-yet constraint.)
- **Container recursion**: slices are implemented; `map[K]A→map[K]B`, `Option[A]→Option[B]`, and
  nested-struct auto-recursion follow the identical rule and are noted but kept minimal in v1.
- **`Option[T]↔*T` generic bridge** and **user-facing generic `from`** are reserved (SYNTAX.md);
  examples use concrete registry entries.
- **Rename-with-conversion in the literal form** is written as an explicit verbatim expression; the
  registry auto-conversion applies only to same-named `...derive` fields.
