# Eval literals and operators — Business Specification

## Overview

The goscript tree-walking interpreter must be able to evaluate ordinary
expressions so real computation runs under interpretation. This story adds
evaluation of primitive literals and the arithmetic, comparison, logical, and
unary operators, each producing the same runtime value Go would compute for the
equivalent expression.

## Functional Requirements

### FR-1: Literal evaluation
The interpreter SHALL evaluate integer, floating-point, string, and boolean
literals to their corresponding runtime values.

### FR-2: Arithmetic operators
The interpreter SHALL evaluate `+`, `-`, `*`, `/`, and `%` on numeric operands
with Go semantics (integer division truncates, `%` is integer remainder). `+`
SHALL also concatenate strings.

### FR-3: Comparison operators
The interpreter SHALL evaluate `==`, `!=`, `<`, `<=`, `>`, `>=`, producing a
boolean result with Go ordering/equality semantics for numbers and strings, and
equality for booleans.

### FR-4: Logical operators with short-circuit
The interpreter SHALL evaluate `&&` and `||` on booleans, short-circuiting: when
the left operand alone determines the result, the right operand SHALL NOT be
evaluated.

### FR-5: Unary operators
The interpreter SHALL evaluate unary `-` (numeric negation) and `!` (boolean
negation), and SHALL transparently evaluate parenthesized expressions.

## Acceptance Criteria

- [ ] An integer literal evaluates to the matching integer value.
- [ ] A float literal evaluates to the matching float value.
- [ ] A string literal evaluates to its unquoted string value.
- [ ] `true` / `false` evaluate to the matching boolean value.
- [ ] `+ - * / %` produce Go-correct results on integers and floats; `+`
      concatenates strings.
- [ ] `== != < <= > >=` produce Go-correct boolean results for numbers and
      strings.
- [ ] `&&` and `||` produce Go-correct boolean results AND short-circuit (a
      right operand with an observable side effect / error is not evaluated when
      the left operand decides the result).
- [ ] Unary `-` and `!` produce Go-correct results; parentheses regroup.
- [ ] A table-driven unit test evaluates at least 12 distinct expression
      programs and asserts each result value.

## User Interactions

No direct user surface. Expression evaluation is consumed internally by the
interpreter's statement execution and by later evaluation stories.

## Error Handling

- Division or remainder by zero SHALL yield a named, descriptive error rather
  than a panic.
- An operator applied to operands of an unsupported kind SHALL yield a
  descriptive error naming the operator and kinds.

## Out of Scope

- Variables, declarations, and assignment (US-006).
- Function calls and identifiers resolving to values (US-007).
- Bitwise operators, shifts, and complex/imaginary literals.
- Composite literals, indexing, and method/field access (US-009+).

## Open Questions

- None. The behavior mirrors Go's expression semantics over the interpreter's
  existing Value model.
