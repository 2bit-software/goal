# Parse Expressions with Pratt and Postfix ? — Business Specification

## Overview

The goal parser must produce precedence-correct expression trees. Today its
expression grammar is minimal: an operand followed by a selector/call/index/
composite-literal postfix chain, with no binary operators, no unary operators,
and no postfix `?`. This makes any expression with operators parse incorrectly
(or not at all). This feature upgrades expression parsing to honor Go's operator
precedence and associativity, recognize prefix/unary operators, and treat goal's
postfix `?` unwrap as a first-class expression node, so downstream tools reason
about one faithful structure.

## Functional Requirements

### FR-1: Binary operator precedence
Binary expressions SHALL parse with Go's operator precedence and left
associativity. The five precedence levels (highest to lowest) are:
`* / % << >> & &^`; then `+ - | ^`; then `== != < <= > >=`; then `&&`; then `||`.
For example `a + b * c` SHALL parse as `a + (b * c)` and `a - b - c` as
`(a - b) - c`.

### FR-2: Unary / prefix operators
Prefix operators `+ - ! ^ & <-` SHALL parse as a unary expression over their
operand, and `*x` SHALL parse as a pointer-dereference expression. Unary binds
tighter than any binary operator.

### FR-3: Postfix chains bind tightest
Selector (`a.b`), call (`f(x)`), and index (`a[i]`) postfix forms SHALL continue
to bind tighter than unary and binary operators.

### FR-4: Postfix `?` unwrap
A `?` immediately following an operand (after its postfix chain) SHALL parse as a
dedicated unwrap expression node wrapping that operand. `f(x)?` SHALL unwrap the
call result; `a.b?` SHALL unwrap the selector. `?` binds tighter than binary
operators, so `f(x)? + 1` unwraps the call, then adds.

### FR-5: No regression
All previously supported expression, statement, and declaration parsing SHALL
continue to work, and composite-literal brace suppression in control-clause
headers SHALL be preserved.

## Acceptance Criteria

- [ ] `f(x)?` parses to an unwrap node whose operand is the call `f(x)`.
- [ ] `a.b?` parses to an unwrap node whose operand is the selector `a.b`.
- [ ] A mixed-precedence binary expression (e.g. `a + b * c == d`) parses to the
      correctly nested tree (`(a + (b * c)) == d`).
- [ ] A left-associative same-precedence chain (e.g. `a - b - c`) nests left.
- [ ] A prefix-unary operand (e.g. `-a * b`) applies the unary tighter than the
      binary operator.
- [ ] The existing parser test suite and the full project test suite remain
      green.

## User Interactions

No direct end-user interaction. The surface is the internal parser API
(`ParseFile`), consumed by later sema/backend/formatter stories.

## Error Handling

Malformed expressions SHALL be reported as accumulated parse errors (existing
mechanism) while the parser continues to make forward progress (never panics,
never loops). A missing operand after an operator is reported and recovery
continues.

## Out of Scope

- Lowering or emitting `?` (later stories US-035, US-019 only parses it).
- match / variant-construction / spread expressions (US-021, US-022).
- Generic type-argument lists in expressions.
- Ternary or other non-Go operators.

## Open Questions

None.
