# 04-option — Syntax

`Option[T]` / nil-safety (spec §3.6): a sum of `Some(T)` / `None` that **must be destructured** (via
`match`) to reach the inner value. This kills nil-dereference — Go's *other* great silent-failure
class. It is nearly free: `Option` is just a library enum over the §2 sum-type machinery, the same
as `Result`.

**Scope of this feature:** the **immediate** case (§8.7) — an `Option` returned and `match`-ed at
the use site — lowered with the §8.4 **pointer strategy** (`*T`). Value types box to `*int` for v1.
A stored `Option` value and value-position `match` are noted but not built here.

This document pins the **surface syntax**. The Go it lowers to is in `TRANSPILE.md`.

---

## 1. Final surface syntax

```goal
func find(id ID) Option[User] {
    u, ok := lookup(id)
    if !ok {
        return Option.None
    }
    return Option.Some(u)
}

// Must destructure — the inner value is unreachable without going through the match:
match find(id) {
    Option.Some(u) => greet(u)
    Option.None => prompt()
}
```

- **`Option[T]`** — Go-generics bracket form, the single type argument always explicit; consistent
  with `Result[T, error]` (03) and the spec §3.6/§8.4 samples. `?` is **not** used for the optional
  type — it is reserved exclusively for propagation (feature 05).
- **Construction is qualified**: `Option.Some(v)` / `Option.None` — the same `Type.Variant(...)`
  qualification as 01-enums and 03-result. `None` is data-less, so no parens (like `Status.Pending`).
- **Consumed via `match`** (feature 02): arms `Option.Some(binding)` / `Option.None`; the `Some` arm
  binds the inner value (bind-the-value, the 02 form). The inner value cannot be reached except
  inside the `Some` arm — that is the nil-deref elimination.

`Some` / `None` remain the conventional variant names (§7); only the `Option.` qualifier is added.

---

## 2. Grammar

```ebnf
OptionType   = "Option" "[" Type "]" .
SomeExpr     = "Option" "." "Some" "(" Expression ")" .
NoneExpr     = "Option" "." "None" .
OptionMatch  = "match" Expression "{" SomeArm NoneArm "}" .   (* arms in either order *)
SomeArm      = "Option" "." "Some" "(" identifier ")" "=>" ArmBody .
NoneArm      = "Option" "." "None" "=>" ArmBody .
```

`OptionType` is the optional type wherever a type may appear. `SomeExpr`/`NoneExpr` appear in
return position (the immediate case). `OptionMatch` is the §3.6 consumer — an instance of the
02-match construct specialized to `Some`/`None`.

---

## 3. Worked examples

### 3.1 Reference type — producer + consumer (`examples/option_find`)

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

### 3.2 Value type — boxing (`examples/option_int`)

```goal
func first(xs []int) Option[int] {
    if len(xs) == 0 {
        return Option.None
    }
    return Option.Some(xs[0])
}
```

`Option[int]` cannot use `nil` for a bare `int`, so it boxes to `*int` (§8.4; see `TRANSPILE.md`).

### 3.3 `Some` arm ignores its binding (`examples/option_exists`)

```goal
func exists(id ID) bool {
    match find(id) {
        Option.Some(u) => return true
        Option.None => return false
    }
}
```

The value is still reached only through the `match`; the binding is simply unused.

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| `Option[T]` must-destructure | Rust/Swift/Scala `Option`/optional | "nil-dereference" → forced visible `match`; the inner value is unreachable otherwise | **Replacing** (§7) — justified: kills Go's other big silent-failure class, and it's nearly free over the §2 machinery |
| `Option[T]` bracket (not `T?`) | plain Go generics; consistent with `Result[...]` | n/a | None; and it keeps `?` reserved for propagation, avoiding glyph overload |
| Qualified `Option.Some` / `Option.None` | 01-enums / 03-result `Type.Variant(...)` | n/a (consistency) | Small: one uniform sum-type construction rule (see DECISIONS) |
| `Some` / `None` names | universal optional idiom | n/a | None — conventional names kept, not Go-ified |
| Consume via `match` | 02-match | the inner value is bound only inside the handled `Some` arm | reuses 02; no new spend |

---

## 5. Resolved open questions / checker-side rules (not transpiled here)

- **Must-destructure / must-use.** That every access goes through a `match` (so the generated
  nil-check is provably present) is the checker's guarantee, not enforced by the reference
  transpiler. Same must-use considerations as `Result` (§3.6).
- **Immediate-vs-stored (§8.7).** This feature lowers the immediate case (pointer strategy). A
  stored `Option` value is the documented fallback (see `TRANSPILE.md` §5); the pointer
  representation `*T` is itself storable, but stored/value-position handling is out of scope here.

---

## 6. Open against spec

None. The spec §3.6 sample wrote bare `Some(...)` / `None`; this audit qualifies them as
`Option.Some(...)` / `Option.None` per the user's choice, for one uniform construction rule across
all sum types. The variant names and the `Option[T]` shape are unchanged, and the lowering is the
§8.4 pointer strategy, so no spec semantics change.
