# Idiom Catalogue — manual go->goal transforms

These are the upgrades `goal fix` does **not** perform. They were proven across
SEAM-002..006 of the self-host idiomatization PRD. Each entry says when to
CONVERT and — just as important — when to REFUSE and record a documented non-fit.

goal is a Go superset, so valid Go is (almost) valid goal. The idioms below turn
Go shapes into goal's native sum-type / exhaustive-match / Result forms.

## Carry-forward gotchas (apply to every transform)

- **enum/sealed zero value is nil.** A goal `enum` lowers to a sealed interface,
  so its zero value is `nil`, not the first variant. Anywhere Go relied on the
  zero value being the 0th iota constant, set the field EXPLICITLY (e.g.
  `Mode.ModeNone`) — do not assume a default variant.
- **enums cannot carry methods.** An enum lowers to an interface; Go forbids
  declaring a method on an interface type. A `(e E) String() string` method on a
  would-be enum must become a FREE function (e.g. `SeverityLabel(s Severity)
  string`) whose body is a `match`.
- **value-position `match` lowering is restricted.** A `match` used as a value
  lowers only in three forms: `x := match ...`, `var x T = match ...`, or
  `return match ...`. It cannot appear as an arbitrary sub-expression. Restructure
  to one of those three shapes.
- **cross-package idioms need whole-program facts.** Cross-package enum-match and
  sealed-match resolution rely on the compiler's whole-program enum/sealed-fact
  propagation (SEAM-CAP / CAP-2 / CAP-3c). Within a SINGLE file or SINGLE package
  scope, a consumer of the converted type that lives in ANOTHER package is a
  documented non-fit: "cross-package consumer not in scope". Convert the
  definition only if every consumer is in scope, or leave it and record the
  refusal.

## 1. iota const block -> goal `enum`

CONVERT when the const block is a closed set of tags used only for
identity/branching:

```go
type Severity int
const (
    Error Severity = iota
    Warning
)
```
becomes
```goal
enum Severity {
    Error
    Warning
}
```
Variants with payload: `enum Shape { Circle { r: float64 }; Rect { w, h: float64 } }`.

REFUSE (keep iota, document the reason) when the value carries:
- **numeric identity** — used as an array index, bit flag, or arithmetic operand;
- **wire/serialization value** — the integer is persisted or sent on the wire;
- **ordering dependence** — compared with `<`/`>` for an inherent order that the
  code relies on (and you do not want to add an explicit ordering function).

Real examples from the self-host: `token.Kind` and `litClass` were KEPT as iota
(numeric/wire/ordering identity); `FuncMod`, `ChanDir`, `Mode`, and `Severity`
were CONVERTED (pure tags).

## 2. type-switch over a closed/sealed scrutinee -> sealed interface + match

CONVERT when a `switch x := v.(type)` ranges over a CLOSED set of concrete types
implementing one interface — seal the interface and make the switch an exhaustive
`match`:

```go
type Node interface{ Pos() Position }
// ... AssignStmt, ReturnStmt, ExprStmt all implement Node
switch s := n.(type) {
case *AssignStmt: ...
case *ReturnStmt: ...
}
```
becomes
```goal
sealed interface Node { Pos() Position }
type AssignStmt struct implements Node { ... }
type ReturnStmt struct implements Node { ... }

match n {
    AssignStmt(s) => ...
    ReturnStmt(s) => ...
}
```
- Sealing makes the set closed so the checker can PROVE exhaustiveness.
- Nested hierarchies (an interface that embeds another sealed interface) use the
  embedding cascade: `sealed interface Expr { Node }` — an implementor of `Expr`
  emits both `isExpr()` and `isNode()` markers (CAP-3d).
- A plain `switch` over a sealed type is a compile error in goal (§9
  switch-coexistence) — every such switch MUST become a `match`.

REFUSE when the interface is genuinely OPEN/extensible (third parties implement
it), or when not all type-switch arms' concrete types are in scope.

## 3. method on a would-be-enum -> free label function

When a type becomes an enum (idiom 1) and it had methods, move each method to a
FREE function (enums can't carry methods):

```go
func (s Severity) String() string { ... }
```
becomes
```goal
func SeverityLabel(s Severity) string {
    return match s {
        Severity.Warning => "warning"
        _ => "error"
    }
}
```
Callers that relied on the `Stringer` (via `%s`) must call the free function
explicitly. This is mechanical once idiom 1 is applied.

## 4. exported fallible `(T,error)` -> `Result[T, error]` / `?`

CONVERT a function whose `(T, error)` is PURE single-value propagation (the
caller either uses T or returns the error):

```go
func parse(s string) (Config, error) {
    if s == "" { return Config{}, errors.New("empty") }
    return Config{Raw: s}, nil
}
```
becomes
```goal
func parse(s string) Result[Config, error] {
    if s == "" { return Result.Err(errors.New("empty")) }
    return Result.Ok(Config{Raw: s})
}
```
and call sites collapse via `?`: `cfg := parse(s)?`.

`goal fix` AUTO-converts the in-function `if err != nil { return ..., err }`
propagation shape. The MANUAL part is choosing which exported signatures to lift
and updating cross-package callers in lockstep.

REFUSE (documented non-fit) — these are SEMANTIC non-fits, not scope problems,
and will NOT be fixed by reaching further:
- **accumulator** — function gathers `[]error` (e.g. an EnrichForeign that
  collects many errors) — not a single Err branch;
- **multi-value return** — returns more than one success value plus error
  (e.g. a 3-/4-value return) — Result holds exactly one T (use a struct bundle
  only if that is a genuine idiom gain);
- **comma-ok control flow** — `(T, bool)` lookup / `v, ok :=` patterns are
  Option-or-keep, not Result; if the bool is real control flow, keep it;
- **cross-package consumer not in scope** — converting would break a caller in a
  package outside the chosen scope.

## Decision table

| Go shape | Convert to | Refuse & keep when |
|----------|------------|--------------------|
| iota const block | `enum` | numeric identity / wire value / ordering dependence |
| type-switch over closed types | sealed interface + `match` | open/extensible interface; arms not all in scope |
| method on would-be-enum | free label `func` + `match` | (mechanical; only if idiom 1 applies) |
| exported `(T,error)` pure-propagation | `Result[T,error]` + `?` | accumulator / multi-value / comma-ok / cross-pkg consumer out of scope |
