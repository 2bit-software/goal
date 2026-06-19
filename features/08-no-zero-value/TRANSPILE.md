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

```goal
return Settings{ primary: Addr{ host: "localhost", port: 8080 }, ...defaults }
```
```go
return Settings{primary: Addr{host: "localhost", port: 8080}, fallback: nil, meta: Addr{}, tags: nil, env: nil, label: "", retries: 0}
```

`fallback` (`*Addr`), `tags` (`[]string`), `env` (`map[string]string`) → `nil`; `meta` (named struct
`Addr`) → `Addr{}`; `label` (alias `Name = string`) → `""`; `retries` (`int`) → `0`.

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
4. **Expand.** For each declared field **not** present, in declared order, emit `name: <zero>` where
   `<zero>` is `zeroLit(type)`. Replace the `...defaults` span with the comma-joined entries (gofmt
   then normalizes layout). If no fields are unset, `...defaults` expands to nothing.

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

No temporaries are synthesized, so the `__gop_` prefix is not needed here.

## Scope / not-checked (the checker's job, not built)

- **Does not reject incomplete literals.** A literal missing a field passes through unchanged;
  flagging it is the checker's located error. The reference transpiler assumes well-formed input.
- **Does not verify field names** in a literal against the struct declaration.
- **Does not judge default appropriateness.** Defaulting an enum-typed field (a closed sum type has
  no safe zero) is a *semantic* error the checker rejects; here it mechanically expands to the
  encoding's zero (`nil` for a sealed interface). Examples set enum fields explicitly.
- **Unknown named types** (declared in another package, not visible in the file) default to `T{}`.
  That is correct for a struct but wrong for an out-of-package interface (which wants `nil`) — not
  recoverable without a type system, so it is deferred rather than guessed-around. In-file types are
  always resolved correctly.
