# 08-no-zero-value — transpile to Go

Governing contract: §8.5 (pure-erasure features) and §8.0 (erasure-with-defensive-panic).

## Erasure vs preservation (§8.0)

- **Erased (static guarantee):** field-completeness checking. The requirement that every field be
  set is a *compile-time check* the goal checker performs and then **discards** — it "only ever
  rejected source" (§8.5). A complete struct literal therefore lowers to **itself, verbatim**. The
  transpiler generates **nothing extra** for it.
- **Rewritten (surface-only sugar):** the `...defaults` escape hatch. It is not a runtime construct —
  it expands at transpile time into explicit per-field zero values so the emitted Go literal is
  complete. After expansion there is no trace of `...defaults` in the Go.
- **Rejected (static guarantee, in this pass):** an `...defaults` that would fill a field whose zero
  is *unsafe* — a `nil` map/pointer/chan/func, a method-bearing named interface, or a sum type with
  no valid variant. Rather than emit the hazardous zero, the pass raises a **located** compile error.
  This is the one piece of checking the defaults pass performs itself (rather than erasing): an
  unsafe zero is precisely the silent-zero footgun the feature exists to close, so it cannot be
  allowed to slip back in through the escape hatch.
- **Preserved (runtime):** nothing. There is no runtime behavior to preserve and no point where the
  checker proves unreachability, so **no defensive `panic` is emitted** by this feature.

This feature is the purest erasure case in the audit: its entire static value (no silent zero
values) lives in the rejected programs, which never reach codegen.

## Input → output pairs

### 1. Complete literal → verbatim passthrough

```goal
return User{ name: "root", email: "root@x", role: RoleAdmin, admin: true }
```
```go
return User{name: "root", email: "root@x", role: RoleAdmin, admin: true}
```

Identical (modulo gofmt). The completeness check is erased; nothing is added.

### 2. `...defaults` over primitives → explicit zeros

```goal
return User{ name: name, role: RoleMember, ...defaults }
```
```go
return User{name: name, role: RoleMember, email: "", active: false, logins: 0}
```

`email`, `active`, `logins` are the declared fields **not** present in the literal; each is filled
with its type's zero, in declared order, where `...defaults` stood.

### 3. `...defaults` over reference and named types

A field whose zero is unsafe (`fallback *Addr` derefs to a panic, `env map[…]` panics on write) must
be set explicitly; `...defaults` then fills only the safe remainder.

```goal
return Settings{ primary: Addr{host: "localhost", port: 8080}, fallback: nil, env: map[string]string{}, ...defaults }
```
```go
return Settings{primary: Addr{host: "localhost", port: 8080}, fallback: nil, env: map[string]string{}, meta: Addr{}, tags: nil, label: "", retries: 0}
```

`meta` (named struct `Addr`) → `Addr{}`; `tags` (`[]string`) → `nil` (a nil slice is safe); `label`
(alias `Name = string`) → `""`; `retries` (`int`) → `0`.

### 4. `...defaults` over an unsafe field → located rejection

```goal
return Store{ name: name, ...defaults }   // entries is a map[string]int
```
```
pass defaults: `...defaults` at L:C cannot default field `entries` of type `map[string]int`: a nil map panics on write — set it explicitly (e.g. `map[string]int{}`)
```

A `nil` map reads fine but panics on write, so it has no safe zero. The pass reports the first
offending field (named, with its type and the `...defaults` position) instead of expanding it.

## Lowering rules

The transpiler is a focused recognizer (lex with `text/scanner`, span-splice, `go/format`); it does
**not** parse full Go. It performs exactly one rewrite — expanding `...defaults` — and passes all
other bytes through.

1. **Collect type info.** Scan top-level `type` declarations once:
   - `type X struct { … }` → ordered field list (`name`, raw type text); `a, b int` yields two
     fields. (Field-list parse is line/`;`-delimited; embedded fields and grouped `type ( … )`
     decls are out of scope.)
   - `type X = Y` (alias), `type X Y` (defined type) → record the underlying type text; `type X
     struct`/`interface` → record the kind. Used only to resolve a named field's zero value.
2. **Find each `...defaults`.** It lexes as four tokens `.` `.` `.` `defaults`. For each, walk
   backward (depth-aware) to the enclosing literal's `{`; the token just before it is the struct
   type name.
3. **Find the already-set fields.** Within that literal's own brace depth, every `IDENT :` is a
   present key.
4. **Classify, then expand.** For each declared field **not** present, in declared order:
   `zeroSafety(type)` first asks whether the field's zero is safe to fill. If not (a `nil`
   map/pointer/chan/func, a method-bearing named interface, or a sum type with no valid variant),
   the pass returns a **located** error naming that field — reporting the first offender. Otherwise
   it emits `name: <zero>` where `<zero>` is `zeroLit(type)`. Replace the `...defaults` span with the
   comma-joined entries (gofmt then normalizes layout). If no fields are unset, `...defaults` expands
   to nothing.

### `zeroSafety(type)` — which zeros `...defaults` refuses

A field's zero is *unsafe* when normal use of the value panics, deadlocks, or no valid value exists.
`zeroSafety` mirrors `zeroLit`'s traversal (prefix checks, then alias/named resolution).

| Declared type form | Verdict |
| --- | --- |
| `*T` pointer | **reject** — nil derefs panic (use `Option[T]`, or set explicitly) |
| `map[K]V` | **reject** — a nil map panics on write |
| `chan …` | **reject** — nil send/recv blocks forever |
| `func …` | **reject** — calling a nil func panics |
| named `enum` / sealed interface (sum type) | **reject** — no valid zero variant |
| method-bearing named `interface{…}` | **reject** — a nil method call panics |
| `string`/`bool`/numeric, `[N]T`, named struct | allow — the zero is a usable value |
| `[]T` slice | allow — a nil slice is safe (`range`/`len`/`append` all work) |
| `error`, `any`, bare `interface{}` | allow — nil error is success; bare any has no methods |
| `type Role int` (int-backed enum) | allow — resolves to `int`; zero is a real variant |
| unknown named type (no local decl) | allow — assumed struct-like, as `zeroLit` does |

### `zeroLit(type)` — the type → zero mapping

Recovered syntactically from the declared type; no type system. Untyped constants (`0`, `""`,
`false`) are assignable to defined types, so `type Role int` correctly defaults to `0`.

| Declared type form | Zero literal |
| --- | --- |
| `*T`, `[]T`, `map[K]V`, `chan …`, `func …`, `interface{…}`, `any`, `error` | `nil` |
| `[N]T` (array) | `[N]T{}` |
| `string` | `""` |
| `bool` | `false` |
| `int`/`uint`/`byte`/`rune`/`float*`/`complex*` (and sized variants) | `0` |
| named local `type X struct{…}` | `X{}` |
| named local `type X interface{…}` | `nil` |
| named local alias / defined type | resolve underlying, recurse (depth-guarded) |
| unknown named type (no local decl) | `T{}` (best-effort; see note) |

## Strategy forks

There is **one** lowering (verbatim passthrough + `...defaults` expansion) — no open/closed or
immediate/stored fork applies to struct construction. The only branching is inside `zeroLit`, the
type → zero table above.

## Hygiene

No temporaries are synthesized, so the `__goal_` prefix is not needed here.

## Scope

- **Unsafe-default rejection is performed here** (see `zeroSafety` above): an `...defaults` that
  would fill an unsafe or no-safe-zero field is a located error. This is the one check the pass does
  rather than erases, because an unsafe zero reaching codegen is the exact footgun the feature
  closes. The classification is type-directed; it does **not** inspect explicitly-written values, so
  an author who writes `env: nil` (or any explicit value) for an unsafe field is taken at their word.

Still the **checker's job, not built here**:

- **Does not reject incomplete literals.** A literal missing a field (with no `...defaults`) passes
  through unchanged; flagging it is the checker's located error. The reference transpiler assumes
  well-formed input.
- **Does not verify field names** in a literal against the struct declaration.
- **Unknown named types** (declared in another package, not visible in the file) default to `T{}` and
  are classified *safe*. That is correct for a struct but cannot distinguish an out-of-package
  interface (which wants `nil` and could be unsafe) — not recoverable without a type system, so it is
  deferred rather than guessed-around. In-file types are always resolved correctly.
- **The standalone reference transpiler has no `enum`/`sealed` table**, so it recognizes a sum type's
  no-safe-zero only when the field's type resolves to a named `interface`. The real pipeline pass
  consults `analyze.Sealed`/`analyze.Enums` and rejects both `enum` and sealed-interface sum fields.
