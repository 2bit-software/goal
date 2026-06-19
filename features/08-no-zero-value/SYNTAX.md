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
    return Settings{ primary: Addr{ host: "localhost", port: 8080 }, ...defaults }
}
```

`...defaults` recovers each unset field's zero from its declared type: pointer / slice / map → `nil`,
the named struct `Addr` → `Addr{}`, the string alias `Name` → `""`, `int` → `0`.

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
- **Enum-typed fields have no safe default.** A closed sum type (feature 01) has no zero variant, so
  `...defaults` defaulting an enum field is a *semantic* error the checker should reject. The
  reference transpiler does no checking, so it mechanically expands such a field to the encoding's
  zero (`nil` for the sealed-interface form) — noted in `TRANSPILE.md` as a checker responsibility,
  not a transpiler one. Examples therefore always set enum fields explicitly.
