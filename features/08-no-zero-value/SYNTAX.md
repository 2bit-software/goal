# 08-no-zero-value — required-field struct construction

## Final surface syntax

Constructing a struct **requires every declared field be set** — explicitly, or via the
explicit-defaults escape hatch. Forgetting a field is a located compile error (the checker's job),
**not** a silent Go zero value. This is the **default posture**, not opt-in (§3.5).

There is no new construction *operator*: a goal struct literal is spelled exactly like a Go keyed
composite literal. The only addition is the escape hatch:

```goal
User{ name: "a", email: "b@c", ...defaults }
```

`...defaults` is the trailing **explicit-defaults form** — "set every field I did not list to its
default (zero) value." It is the language's sanctioned way to say "I really do want the zeros for
the rest," so that the strictness is satisfiable without ceremony and the model doesn't route
around the whole feature (§3.5 implementation notes).

### Chosen spelling: `...defaults` (user-selected)

Among the candidates offered, the user chose `...defaults` over `_` (rest marker), `default` (bare
member), and `..Default` (Rust struct-update). Rationale, per the two principles:

- **Familiarity:** the leading `...` is a visual cousin of Go's own `...` (variadic / slice spread)
  and of Rust's `..` struct-update tail — readers parse "and the rest" immediately. No new sigil is
  introduced that means something unrelated elsewhere.
- **Feedback (located error):** the keyword **names the intent** ("defaults") and is greppable.
  Because opting into defaults is *explicit and visible*, a genuinely **forgotten** field cannot
  masquerade as a deliberate default — the checker still flags the missing field when `...defaults`
  is absent. That is the whole point of making completeness the default: the silent-zero-value
  footgun (a struct field left unset reading back as `0`/`""`/`nil`) is converted into a located
  "missing field `role`" error.

## Grammar

Struct construction reuses Go's keyed composite-literal grammar, with one optional trailing element:

```ebnf
StructLit   = TypeName "{" [ ElemList [ "," ] ] "}" .
ElemList    = Elem { "," Elem } .
Elem        = KeyedField | DefaultsForm .
KeyedField  = FieldName ":" Expression .
DefaultsForm = "..." "defaults" .              (* at most one per literal *)
```

- `DefaultsForm` may appear at most once in a literal. Conventionally it is **last** (it reads as
  "…and the rest: defaults"); other positions are accepted but discouraged.
- Positional (unkeyed) literals are **not** part of the goal surface for this feature — required-field
  construction is expressed with keyed fields so the "which field is missing" diagnostic is precise.
  (The reference transpiler does not enforce this; it is a checker rule.)
- `...defaults` nests anywhere a struct literal nests (return position, assignment RHS, an element of
  an outer literal, a call argument), since it is just an element of the literal.

## Worked examples

### 1. Complete literal — every field set (common case)

```goal
type User struct { name string; email string; role Role; admin bool }

func newAdmin() User {
    return User{ name: "root", email: "root@x", role: RoleAdmin, admin: true }
}
```

No `...defaults`; nothing to expand. (An *incomplete* literal here — say, omitting `admin` — is the
checker's located error, not a transpiler concern.)

### 2. Explicit-defaults over primitives

```goal
type User struct { name string; email string; role Role; active bool; logins int }

func newMember(name string) User {
    return User{ name: name, role: RoleMember, ...defaults }
}
```

`name` and `role` are set explicitly (an enum has no safe zero, so it *must* be set); `...defaults`
fills `email`, `active`, `logins` with their zero values.

### 3. Explicit-defaults over reference and named types

```goal
type Addr struct { host string; port int }
type Name = string
type Settings struct {
    primary  Addr
    fallback *Addr
    meta     Addr
    tags     []string
    env      map[string]string
    label    Name
    retries  int
}

func defaultSettings() Settings {
    return Settings{
        primary:  Addr{ host: "localhost", port: 8080 },
        fallback: nil,
        env:      map[string]string{},
        ...defaults,
    }
}
```

`fallback` (`*Addr`) and `env` (`map[…]`) have **unsafe** zeros, so they are set explicitly (see
below). `...defaults` recovers each remaining field's zero from its declared type: the named struct
`Addr` → `Addr{}`, the slice `tags` → `nil` (a nil slice is safe), the string alias `Name` → `""`,
`int` → `0`.

## Unsafe defaults are rejected

`...defaults` only fills a field whose zero value is **safe** — usable as-is. A field whose zero is a
latent hazard is a **located compile error** instead, converting the silent-zero footgun into a
diagnostic even inside the escape hatch. A zero is *unsafe* when normal use panics, deadlocks, or no
valid value exists:

| Field type | Verdict | Why |
| --- | --- | --- |
| `*T` pointer | **reject** | nil derefs panic — use `Option[T]` for an optional value, or set it explicitly |
| `map[K]V` | **reject** | a nil map panics on write |
| `chan …` | **reject** | nil send/recv blocks forever |
| `func …` | **reject** | calling a nil func panics |
| `enum` / sealed-interface sum type | **reject** | no valid zero variant |
| method-bearing named `interface{…}` | **reject** | a nil method call panics |
| primitives, `[N]T`, named struct | allow | the zero is a usable value |
| `[]T` slice | allow | a nil slice is safe (`range`/`len`/`append` all work) |
| `error`, `any`, bare `interface{}` | allow | a nil `error` is success; bare `any` has no methods |
| `type Role int` (int-backed enum) | allow | resolves to `int`; its zero is a real variant |

The escape for a genuinely-optional reference is `Option[T]` (feature 04), not a nil pointer ridden in
on `...defaults`. The check is **type-directed and scoped to the fields `...defaults` fills**: an
author who writes a field explicitly — even `fallback: nil` — has made a deliberate choice and is
taken at their word.

A rejected literal reports the first offending field, with its type and the `...defaults` position:

```
`...defaults` at 9:27 cannot default field `entries` of type `map[string]int`: a nil map panics on write — set it explicitly (e.g. `map[string]int{}`)
```

## Rationale, tied to the two principles

- **Default, not opt-in** (§3.5): "an opt-in strictness the model forgets to opt into catches
  nothing." Required-field construction is on by default; `...defaults` is the *opt-out for a single
  literal*, not an opt-in for the feature. Same safe-path-as-default logic that made `_` the
  rest-arm marker over a reflexive `default` (feature 02).
- **Attacking Go's actual weakness:** this is not generic rigor — it targets the named Go footgun
  that an unset struct field is silently its zero value. The familiarity spend (`...defaults`) is
  justified because the feature is "pure #1 error-catching on a named footgun" (§7 budget note,
  §3.5).
- **Conventional spelling preserved:** keyed composite literals are spelled exactly as in Go;
  `...defaults` is the only addition, and it leans on the existing `...` reading.

## Resolved open questions (§9)

- **Explicit-defaults form** → **`...defaults`** (trailing keyword-spread element). Decided via
  `AskUserQuestion`; alternatives `_`, bare `default`, and `..Default` were refused (see
  `DECISIONS.md` for the per-option justification). This closes the §9 "explicit-defaults form"
  item.

## Open against spec

- **"Defaults" means Go zero values, written explicitly.** The spec phrases the escape hatch as "I
  really do want **zero/defaults**" (§3.5) and says `...defaults` "lowers to explicit per-field
  default values so the generated literal is also complete" (§8.5). There is **no per-field
  declared-default syntax** in the spec (e.g. `name string = "anon"`), so this feature does **not**
  introduce one — `...defaults` fills each unset field with its *type's zero value*, spelled
  explicitly. A declared-default facility would be a separate, additive feature; flagged here rather
  than invented (per "Do NOT add features").
- **Unsafe / no-safe-zero fields are rejected by the defaults pass.** A closed sum type (feature 01)
  has no zero variant, and several reference-type zeros (`nil` map/pointer/chan/func, method
  interface) are latent runtime hazards. `...defaults` defaulting such a field is now a **located
  error from the pass itself** (see "Unsafe defaults are rejected" above and `TRANSPILE.md`), not a
  silent expansion — the one bit of checking this pass performs rather than erases, because letting
  an unsafe zero through the escape hatch would reopen the exact footgun the feature closes. This
  goes slightly beyond the original §3.5 framing ("`...defaults` = zero values"), narrowly: it
  rejects *unsafe* zeros while still filling safe ones, and stays scoped to defaulted fields —
  pervasive nil-elimination across all pointers remains the deferred §5 decision. Decided with the
  user; rationale in `DECISIONS.md`.
