# Match Lowering Including Value Position (US-036) — Business Specification

## Overview

The new AST backend already lowers a statement-position `match` over a Result or
Option. It does not yet lower a `match` over an enum, nor a `match` used in value
position. This feature completes `match` lowering: an enum `match` lowers to a Go
type-switch over the §8.1 sum encoding, in both statement position and value
position (`return match …` and `var name T = match …`). This finally makes
value-position `match` work on the new front-end (the legacy splice engine
refused the untyped value form; the typed forms here are supported).

## Functional Requirements

### FR-1: Statement-position enum match
An enum `match` used as a statement SHALL lower to a Go type-switch on the
scrutinee's dynamic type, with one `case <Enum>_<Variant>:` per matched variant
and each arm body emitted as the case body.

### FR-2: Exhaustiveness default
A `match` without a `_` rest arm SHALL emit a `default:` clause that panics with
the message `unreachable: non-exhaustive <Enum> (compiler invariant violated)`.
A `match` with a `_` rest arm SHALL emit that arm's body as the `default:`
clause body instead.

### FR-3: Payload bindings
When an arm binds a variant payload (e.g. `Status.Active(a) => render(a.since)`),
references to the binding SHALL resolve to the type-switch guard variable, and
field accesses on the binding SHALL use the exported Go field name
(`a.since` -> `<guard>.Since`). The guard variable SHALL be introduced only when
at least one arm references its binding.

### FR-4: Value-position enum match
An enum `match` used as the operand of a `return` SHALL lower to the same
type-switch with each arm emitting `return <arm-body>`. An enum `match` used as
the initializer of a typed `var name T = match …` SHALL emit a `var name T`
declaration followed by the type-switch with each arm emitting `name = <arm-body>`.

### FR-5: Behavioral conformance
The generated Go for every 02-match case and for a new value-position-match case
SHALL compile and vet cleanly (the behavioral tier).

## Acceptance Criteria

- [ ] An enum `match` statement transpiles to a `switch <scrut>.(type)` with a
      `case` per variant.
- [ ] A `match` with no rest arm produces a panicking `default`; a `match` with
      a `_` arm produces a `default` running that arm's body.
- [ ] A payload binding and its field access resolve to the guard variable with
      the exported field name; the guard is omitted when unused.
- [ ] `return match …` produces a type-switch whose arms `return` their bodies.
- [ ] `var name T = match …` produces a `var name T` plus a type-switch whose
      arms assign `name = <body>`.
- [ ] The four 02-match transpile cases pass the behavioral tier through the new
      backend.
- [ ] A new value-position-match corpus case exists and passes the behavioral
      tier through the new backend.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

Indirect: goal developers write `match` and run `goal` (transpile/build). No new
CLI surface. The lowering is exercised through the corpus behavioral runner.

## Error Handling

A `match` shape the backend cannot lower (e.g. a closed-E Result match, or a
value-position match in an unsupported position) continues to surface a
descriptive backend error rather than emitting wrong code. Enum and the
supported value positions no longer error.

## Out of Scope

- Exact golden-text parity (gensym names differ from the splice engine); that is
  US-042.
- The untyped `name := match …` inference form (legacy refused it; not required).
- Closed-E Result match (US-037) and any non-enum value-position match.
- `match` as a sub-expression nested inside a larger expression.

## Open Questions

None — the encoding mirrors the documented §8.2 lowering and the shipped legacy
pass.
