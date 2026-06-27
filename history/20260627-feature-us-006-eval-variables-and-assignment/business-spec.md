# US-006 Eval variables and assignment — Business Specification

## Overview

The goscript tree-walking interpreter must let program state evolve during
evaluation. This story adds declaration and assignment to the interpreter so a
running goal program can introduce named variables and constants and then
mutate them. Without it, the interpreter can only evaluate pure expressions.

## Functional Requirements

### FR-1: Variable declaration
A `var` declaration SHALL bind each declared name to the evaluated initializer
value in the current scope. A `var` declaration with no initializer SHALL bind
the name to the safe zero value for its kind (numeric 0, empty string, false).

### FR-2: Short variable declaration
A short variable declaration (`name := expr`) SHALL evaluate the right-hand side
and bind the name in the current scope.

### FR-3: Constant declaration
A `const` declaration SHALL evaluate its value and bind the name in the current
scope, readable like any other binding.

### FR-4: Plain assignment
A plain assignment (`name = expr`) to an already-declared variable SHALL update
the binding where that variable was declared, without creating a new shadowing
binding. Parallel assignment (`a, b = b, a`) SHALL evaluate all right-hand
sides before performing any binding.

### FR-5: Compound assignment
A compound assignment (`+=`, `-=`, `*=`, `/=`, `%=`) SHALL read the variable's
current value, apply the corresponding binary operator with the right-hand
side, and write the result back to the existing binding.

### FR-6: Variable reads
Referencing a declared variable in an expression SHALL yield its current value.
Referencing an undefined name, or assigning to an undeclared name, SHALL produce
a descriptive, located error — never a silent nil or zero.

## Acceptance Criteria

- [ ] A program that declares variables with `var` (with and without an
      initializer), `:=`, and `const`, reassigns them with `=`, and
      compound-assigns them (`+=`, `-=`, etc.) yields the expected final values.
- [ ] Reading a declared variable returns the most recently assigned value.
- [ ] A plain `=` updates the existing binding rather than shadowing it (the
      change is visible through the original scope).
- [ ] Assigning to an undeclared name returns a descriptive error.
- [ ] Reading an undefined name returns a descriptive error.
- [ ] The project verify gates pass: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`.

## User Interactions

No direct user interface. The behavior is exercised by goal programs run
through the interpreter and asserted by unit tests in `package interp`.

## Error Handling

- Reading an undefined name → a descriptive not-found error naming the symbol.
- Assigning to an undeclared name → a descriptive not-found error naming the
  symbol.
- Applying a compound operator to incompatible operand kinds → the same
  descriptive operator/kind-mismatch error the expression evaluator already
  produces.

## Out of Scope

- Composite (struct/slice/map) zero values and composite-target assignment
  (e.g. `s[i] = x`, `p.field = x`) — these arrive with US-009/US-010.
- Bitwise/shift compound operators beyond the arithmetic set (deferred; they
  are not in the current Go-subset corpus and yield a descriptive error).
- Function calls and multi-value RHS from a call (`a, b = f()`) — US-007.
- Typed constant semantics / iota — out of scope for the runtime model.

## Open Questions

None — the scope mirrors the established US-005 eval seam and US-003 Env design.
