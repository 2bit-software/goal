# Parse match and patterns — Business Specification

## Overview

The goal front-end parser must turn `match` expressions into AST. `match`
selects on an enum/sealed value and runs the arm whose pattern matches. Unlike
the legacy splice transpiler, `match` must be representable in *value position*
(e.g. `var x = match s { ... }` or `return match s { ... }`) as well as
statement position, because the AST models `match` as an expression.

## Functional Requirements

### FR-1: Statement-position match
The parser SHALL parse a `match` that appears where a statement is expected
(e.g. a bare `match s { ... }` inside a function body) into the match AST node.

### FR-2: Value-position match
The parser SHALL parse a `match` that appears where an expression is expected
(`var x = match s { ... }`, `return match s { ... }`) into the same match AST
node, so value-position match is representable.

### FR-3: Variant patterns with bindings
The parser SHALL parse an arm pattern of the form `Enum.Variant` (data-less) and
`Enum.Variant(binding)` (payload bound to an identifier), recording the enum
reference, the variant tag, and the optional binding.

### FR-4: Rest pattern
The parser SHALL parse the catch-all arm pattern `_` as a distinct rest pattern
(not an ordinary identifier).

### FR-5: Arm bodies
The parser SHALL parse each arm as `Pattern => Body`, where the body is either an
expression (value position) or a block.

## Acceptance Criteria

- [ ] A statement-position `match` (e.g. `features/02-match/examples/status_match.goal`)
      parses with no errors and yields the expected number of arms with the
      expected variant/binding structure.
- [ ] A value-position `match` (e.g. `status_var.goal` and `status_return.goal`)
      parses with no errors and yields the expected arms.
- [ ] A binding pattern `Status.Active(a)` records the enum, variant, and the
      bound identifier `a`.
- [ ] A rest pattern `_` (e.g. `status_rest.goal`) parses as the rest pattern,
      distinct from an identifier.
- [ ] The project verify gates remain green: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`.

## User Interactions

None directly — this is a compiler-internal parsing capability consumed by later
lowering/checking stories.

## Error Handling

Consistent with the rest of the parser: a malformed arm records a parse error
but the parser always advances so it makes progress and returns a partial AST.

## Out of Scope

- Lowering / code generation for match (later stories US-036).
- Exhaustiveness checking of match arms (US-029).
- Match guards or nested/destructuring sub-patterns beyond a single payload
  binding.
- Construction `Enum.Variant(field: v)` and spread (`...defaults`) — US-022.

## Open Questions

None.
