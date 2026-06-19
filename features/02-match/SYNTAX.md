# 02-match — Syntax

Pattern-matching `match` with exhaustiveness over closed enums (spec §3.1). A syntactically
distinct construct (not Go's `switch`) with `=>` arms, payload binding in the arm, a deliberate
`_` rest-arm, and — enforced by the checker, not this audit — exhaustiveness. Builds directly on
the 01-enums encoding: arms narrow to the per-variant struct and bound fields become field
accesses.

This document pins the **surface syntax**. The Go it lowers to is in `TRANSPILE.md`.

---

## 1. Final surface syntax

```goal
match s {
    Status.Pending => startOnboarding()
    Status.Active(a) => render(a.since)
    Status.Cancelled(c) => logEvent(c.reason, c.at)
}
```

- **`match <scrutinee> { <arms> }`** — a distinct keyword, never `switch`. Plain `switch` on a
  closed enum is a **compile error** (see §5).
- **Arms are `Pattern => Body`**, newline-separated, no trailing punctuation (the 01-enums block
  idiom, carried forward).
- **`=>` is the arm arrow** — conventional, non-negotiable (§7); not Go-ified.
- **Patterns are qualified**: `Status.Active` — the enum-qualified variant, matching the
  construction form `Status.Active(...)` from 01-enums.
- **Payload binding binds the whole narrowed variant to one name**: `Status.Active(a)` binds `a`
  to the `Active` value; fields are read off it (`a.since`). A data-less variant takes no binding
  (`Status.Pending`). A variant whose payload you don't need can also be written without a binding
  (`Status.Active` matches and ignores the payload).
- **`_` is the rest-arm** — conventional, non-negotiable; a *deliberate* opt-out of exhaustiveness,
  never a reflexive Go `default:`.

### 1.1 Statement vs expression — one unified construct

`match` is a single construct usable in **both** positions:

```goal
// statement position — arms are statements
match s {
    Status.Active(a) => render(a.since)
    _ => showPlaceholder()
}

// value position — the same construct yields a value
label := match s {
    Status.Pending => "pending"
    Status.Active(a) => "active"
    Status.Cancelled(c) => c.reason
}

// value position via return
func label(s Status) string {
    return match s {
        Status.Pending => "pending"
        Status.Active(a) => "active"
        Status.Cancelled(c) => c.reason
    }
}
```

In statement position an arm body is a statement (or a `{ block }`); in value position each arm
body is an expression that yields the match's value. (Lowering: §8.2 — `var x T` + per-arm
assignment, or per-arm `return`; never an IIFE.)

---

## 2. Grammar

EBNF fragment (`{ x }` = zero-or-more, `[ x ]` = optional). A `MatchExpr` is both an expression
and, in statement position, an expression-statement.

```ebnf
MatchExpr  = "match" Expression "{" { Arm } "}" .     (* arms newline-separated *)
Arm        = Pattern "=>" ArmBody .
Pattern    = "_"
           | identifier "." identifier [ "(" identifier ")" ] .   (* Enum.Variant[(binding)] *)
ArmBody    = Expression | Block .
```

Notes:
- The leading `identifier "." identifier` of a non-rest `Pattern` is the **qualified variant**
  (enum name `.` variant name). The optional `"(" identifier ")"` binds the narrowed variant value
  to that name for the arm body.
- Exactly one `_` arm is permitted; it is the rest-arm.
- `ArmBody` is an `Expression` in value position and a statement/`Block` in statement position.

---

## 3. Worked examples

### 3.1 Statement position, fully exhaustive, payload binding

```goal
match s {
    Status.Pending => startOnboarding()
    Status.Active(a) => render(a.since)
    Status.Cancelled(c) => logEvent(c.reason, c.at)
}
```
All variants handled → the checker proves exhaustiveness; the lowering emits a defensive
`panic` default (§8.2). (`examples/status_match`.)

### 3.2 Deliberate partial handling with `_`

```goal
match s {
    Status.Active(a) => render(a.since)
    _ => showPlaceholder()
}
```
`_` is a conscious opt-out → a **real** default, not a panic. (`examples/status_rest`.)

### 3.3 Expression position (return)

```goal
func label(s Status) string {
    return match s {
        Status.Pending => "pending"
        Status.Active(a) => "active"
        Status.Cancelled(c) => c.reason
    }
}
```
Each arm yields the function's return value. (`examples/status_return`.)

### 3.4 Expression position (typed var)

```goal
func describe(s Status) string {
    var d string = match s {
        Status.Pending => "pending"
        Status.Active(a) => "active"
        Status.Cancelled(c) => c.reason
    }
    return d
}
```
The match initializes `d`. (`examples/status_var`.) The untyped short form `d := match s { ... }`
is identical *surface* syntax and equally valid goal; the reference transpiler defers it because
recovering `d`'s type needs the checker's inference (see `TRANSPILE.md` §5).

---

## 4. Rationale (tied to the two principles)

| Choice | Idiom it lands on | Error class it converts | Familiarity spend |
|---|---|---|---|
| Distinct `match` keyword (not `switch`) | Rust/Swift/Scala `match`/`case` | "reflexive Go `default:` silently swallows omitted enum cases" → exhaustiveness is enforceable | New keyword, justified: it is *the* construct that makes exhaustiveness reachable (§3.1) |
| `=>` arrow, `_` rest-arm | Rust/Scala/the cross-language match idiom | "is this catch-all deliberate?" → `_` is a *conscious* opt-out, visually distinct from `default:` | None net — `=>`/`_` are the widely-seen forms; diverging from Go here is a familiarity *win* |
| Newline-separated arms | 01-enums block idiom / Go block decls | "trailing-comma churn" avoided | None |
| Qualified `Status.Active` | matches 01-enums construction | keeps the enum visible at the match site; avoids ambiguity if the same variant name recurs | Minor verbosity; consistency with construction |
| Bind-the-value `Status.Active(a)` → `a.since` | Go's own type-switch idiom (`case T: v.Field`) | binds the narrowed variant so data extraction and exhaustiveness compose (§3.1) | Lowest — this *is* how Go's type-switch binding reads |
| Unified statement/expression `match` | Rust expression-match | "match can't produce a value, forcing a mutable two-step" → value-returning match | Small: value-position match isn't Go, but lowers to plain `var x T` + assignment (no IIFE, §8.2) |

**Conventional names preserved** (§7): `=>` and `_` stay as-is; not Go-ified.

---

## 5. Resolved open questions

### Switch-coexistence rule (§9 / §3.1) — resolved

**Plain `switch` applied to a closed-enum value is a compile error that redirects to `match`.**
Plain `switch` survives unchanged for non-enum switching (ints, strings, arbitrary type switches);
it is rejected *only* on the closed-enum types that need exhaustiveness. This makes the unsafe,
exhaustiveness-free form **unreachable on exactly the types where omitted cases hide** — the
spec's "option 3" (§3.1 implementation notes), and consistent with §3.1 "Refused: plain `switch`
on a closed enum."

Error message (the exhaustiveness diagnostic *is* the product, §1.4 — list the missing variants):

```
error: cannot 'switch' on closed enum 'Status'; use 'match'
  --> app.goal:42:5
   |
42 |     switch s {
   |     ^^^^^^ plain 'switch' gives no exhaustiveness check
   = note: 'match' requires every variant to be handled (or dismissed with '_')
   = missing if you switch today: Pending, Active, Cancelled
   = help: replace 'switch s {' with 'match s {' and '=>' arms
```

Rejected alternatives (see `DECISIONS.md`): **allow-but-unchecked** (the reflex-`switch` failure
§3.1 calls out); **allow-only-with-explicit-default** (a present `default:` is precisely what
defeats exhaustiveness, so it legitimizes the reflex we want to block).

This is a **checker** rule. The reference transpiler does no checking and simply does not transform
plain `switch` (it passes through); enforcement lands with the checker.

---

## 6. Open against spec

None. The spec's §3.1 sample used positional binding (`Active(since)`, `Cancelled(reason, at)`);
this audit adopts bind-the-value (`Active(a)` + `a.since`) per the user's choice and qualifies
variants (`Status.Active`) to match 01-enums construction. Both lower to the §8.2 type-switch the
spec mandates (bound names → field accesses on the narrowed type), so no spec semantics change.
