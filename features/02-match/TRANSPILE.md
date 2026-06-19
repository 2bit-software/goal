# 02-match — Transpile to Go

Governing contract: spec §8.2 (`match` → type switch + erased exhaustiveness) and §8.0 (erase
checks, preserve runtime semantics, defensive `panic` on proven-unreachable points). `match`
reuses the 01-enums encoding (§8.1): the scrutinee is the sealed interface, the arms narrow to the
per-variant structs.

---

## 1. The lowering (§8.2)

`match` lowers to a **Go type switch** over the variant structs:

```go
switch __gop_v := <scrutinee>.(type) {
case <Enum>_<Variant>:
    <arm body, with bound fields rewritten to __gop_v.Field>
...
default:
    // proven-exhaustive -> defensive panic (erasure-with-defensive-panic, §8.0)
    panic("unreachable: non-exhaustive <Enum> (compiler invariant violated)")
}
```

- The type-switch guard variable is the hygienic `__gop_v` (§8 prefix). It is emitted **only when
  some arm actually references its binding**; otherwise the guard is omitted
  (`switch <scrutinee>.(type)`) so no unused variable is produced.
- **Payload binding maps onto type-switch narrowing.** The bound name (`a` in `Status.Active(a)`)
  *is* the narrowed value, so it is rewritten to `__gop_v`, and field reads on it are exported:
  `a.since` → `__gop_v.Since` (the goal field name capitalized to the exported Go field from §8.1).
- **Proven-exhaustive → `panic` default.** The checker proved every variant is handled; the
  `default` is unreachable, so per §8.0 it is a **loud** `panic`, never silent fall-through.
- **Explicit `_` → real default.** A `_` arm becomes an ordinary `default:` with that arm's body,
  and **no** panic is emitted. The transpiler carries the "has explicit rest-arm" bit through.

---

## 2. Input → output pairs

### 2.1 Statement position, exhaustive (`examples/status_match`)

```goal
match s {
    Status.Pending => startOnboarding()
    Status.Active(a) => render(a.since)
    Status.Cancelled(c) => logEvent(c.reason, c.at)
}
```

```go
switch __gop_v := s.(type) {
case Status_Pending:
	startOnboarding()
case Status_Active:
	render(__gop_v.Since)
case Status_Cancelled:
	logEvent(__gop_v.Reason, __gop_v.At)
default:
	panic("unreachable: non-exhaustive Status (compiler invariant violated)")
}
```

### 2.2 Explicit `_` → real default (`examples/status_rest`)

```goal
match s {
    Status.Active(a) => render(a.since)
    _ => showPlaceholder()
}
```

```go
switch __gop_v := s.(type) {
case Status_Active:
	render(__gop_v.Since)
default:
	showPlaceholder()
}
```

### 2.3 Expression position — return (`examples/status_return`)

```goal
return match s {
    Status.Pending => "pending"
    Status.Active(a) => "active"
    Status.Cancelled(c) => c.reason
}
```

```go
switch __gop_v := s.(type) {
case Status_Pending:
	return "pending"
case Status_Active:
	return "active"
case Status_Cancelled:
	return __gop_v.Reason
default:
	panic("unreachable: non-exhaustive Status (compiler invariant violated)")
}
```

Each arm becomes `return <armExpr>`. The `default: panic(...)` makes the type switch a terminating
statement, so the enclosing function needs nothing after it. (No IIFE, no temp — §8.2.)

### 2.4 Expression position — typed var (`examples/status_var`)

```goal
var d string = match s {
    Status.Pending => "pending"
    Status.Active(a) => "active"
    Status.Cancelled(c) => c.reason
}
```

```go
var d string
switch __gop_v := s.(type) {
case Status_Pending:
	d = "pending"
case Status_Active:
	d = "active"
case Status_Cancelled:
	d = __gop_v.Reason
default:
	panic("unreachable: non-exhaustive Status (compiler invariant violated)")
}
```

This is the spec's preferred value-position lowering verbatim: `var x T` before the switch, an
assignment per arm — no closure.

---

## 3. Lowering rules (the general algorithm)

1. **Scrutinee.** The text between `match` and the arm-block `{` becomes the type-switch operand:
   `<scrutinee>.(type)`.
2. **Arm patterns.** `Enum.Variant` → `case Enum_Variant:` (the §8.1 struct name). `Enum.Variant(b)`
   additionally binds `b` (see 4). `_` → `default:` (real default; suppresses the panic).
3. **Arm bodies.** Each body is rewritten (binding/field rewrite, below) and wrapped per position:
   - statement → emitted as-is,
   - `return match` → `return <body>`,
   - `var name T = match` → `name = <body>` (with `var name T` emitted before the switch).
4. **Binding + field rewrite.** Within an arm with binding `b` of variant `V`: every reference to
   `b` becomes `__gop_v`; a field read `b.f` where `f` is a declared field of `V` becomes
   `__gop_v.F` (exported). Non-field selectors on `b` keep their selector, only the base is
   renamed.
5. **Default.** If a `_` arm exists, its body is the `default`. Otherwise emit the defensive
   `panic("unreachable: non-exhaustive <Enum> (compiler invariant violated)")`.
6. **Guard variable.** Emit `__gop_v :=` only if step 4 produced at least one `__gop_v` reference;
   else `switch <scrutinee>.(type)`.

Everything outside a `match` (and the `enum` declarations, lowered via the reused §8.1 path) is
passed through verbatim and the whole output is `go/format`-ed.

---

## 4. Erasure vs preservation (§8.0)

| Aspect | Fate | Why |
|---|---|---|
| **Exhaustiveness check** | **Erased** | The checker proved every variant is handled; the generated Go does not re-verify it. It constrained which source was legal, nothing more. |
| **The proven-unreachable default** | **Erased check, but a defensive `panic` is emitted** | Per §8.0: we do not re-check the proof, but we do not trust the universe to honor it silently. If an unsafe escape or checker bug ever lets control reach it, it fails **loud**, not as a silent wrong-zero-value. |
| **The match value / control flow** | **Preserved** | Which arm runs, the value an expression-match yields, and `return`/branch control flow are runtime semantics — the type switch preserves them exactly. |
| **Payload binding** | **Preserved** (as field access) | The bound fields are real runtime data; binding lowers to field reads on the narrowed struct. |
| **Explicit `_` default** | **Preserved** | A deliberate rest-arm is runtime behavior (it *does* something), so it becomes a real `default`, distinct from the erased-but-panicking exhaustive default. |

---

## 5. Strategy forks

The single fork is **match position**, selected purely syntactically:

| Position | Detected by | Lowering |
|---|---|---|
| statement | `match` not preceded by `return`/`=` | arms emitted as statements; panic-or-real default |
| return | `match` preceded by `return` | each arm `return <expr>`; the switch is the function tail |
| typed var | `var NAME TYPE = match ...` | `var NAME TYPE` then per-arm `NAME = <expr>` |

**Deferred:** the untyped value form `NAME := match ...`. Its lowering is *identical* to the typed
`var` case, but recovering `NAME`'s type requires the checker's type inference, which the
no-checking reference transpiler does not have. The transpiler therefore rejects it with a located
message pointing to `var NAME T = match` / `return match`. This is the §8.2 "carry the result type
from the checker" requirement showing through: with the checker present, `:=` lowers like §2.4.

There is **no** immediate-vs-stored fork here (that is a `Result`/`Option` concern, §8.7). The
scrutinee is always the §8.1 sealed-interface value.

---

## 6. Hygiene

The only synthesized temporary is the type-switch guard `__gop_v` (§8 prefix), emitted only when a
binding is used. No other locals are introduced. Arm bodies are copied from source with only the
binding/field rewrites applied, so user identifiers are untouched.
