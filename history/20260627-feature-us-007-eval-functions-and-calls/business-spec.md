# Eval functions and calls — Business Specification

## Overview

The goscript tree-walking interpreter (internal/interp) must execute user-defined
functions: a function declaration becomes a callable runtime value, a call binds
its arguments to the function's parameters and runs its body, the body can return
one or more values, and a function can call itself so recursive algorithms run.

This is what turns the interpreter from an expression evaluator into one that can
run structured programs.

## Functional Requirements

### FR-1: Function declarations are runtime values
Every top-level plain function declaration in the program is available as a
callable value, resolvable by its name during evaluation.

### FR-2: Calls bind arguments to parameters
Calling a function evaluates each argument and binds it, positionally, to the
function's declared parameters in a fresh scope for that call. Each invocation
gets independent parameter bindings.

### FR-3: Functions return values
A function body's `return` produces the function's result. `return e` yields the
value of `e`; a bare `return` (and falling off the end of the body) yields no
values. A function may declare and return multiple values via `return a, b`.

A multi-value call SHALL be usable in exactly two positions: (a) as the sole
right-hand side of a multi-target assignment/short-var declaration —
`a, b := divmod(x, y)` — binding each result positionally to a target, and (b) as
the sole operand list of a `return`. In a single-value position (e.g. an operand
of an operator, a single assignment target, or one argument among several), a
call SHALL resolve to exactly one value; a call that produces zero or more than
one value in a single-value position is a descriptive error.

### FR-4: Control flow needed for recursion
The interpreter SHALL evaluate `if`/`else` with a bool condition and `return` so
that recursive base cases work. A non-bool `if` condition is a descriptive error.
(The full control-flow suite — `for`, `switch`, `break`/`continue` — is US-008.)

### FR-5: Recursion is supported
A function may call itself (directly). A call resolves the function by name
through the scope chain, and all top-level functions are registered as values
before evaluation begins, so a function is visible to its own body and to
forward references — enabling recursive algorithms such as factorial and
fibonacci.

### FR-6: Loud refusals
Calling an undefined name yields the existing `*NotFoundError`
(`"undefined: <name>"`). Calling a non-function value yields a descriptive
`"interp: cannot call <kind>"` error. An argument-count mismatch (args vs.
declared parameters) and a result-count mismatch (returned values vs. multi-
assignment targets) each yield a descriptive error naming the expected and actual
counts. None of these is a silent nil or wrong result.

### FR-7: Observation seam for tests
Top-level functions are registered into the interpreter's root scope at
construction, so a `package interp` unit test can construct the interpreter from
a parsed + sema-resolved program and evaluate a call expression against the root
scope to observe the returned value — the same direct-`evalExpr` testing pattern
established by US-005/US-006.

## Acceptance Criteria

- [ ] The interpreter evaluates function declarations as values (a top-level
      function name resolves to a callable function value).
- [ ] A call binds its arguments to the callee's parameters in a fresh per-call
      scope; an arg-count mismatch is a descriptive error.
- [ ] A function with multiple return values (e.g. `divmod(a,b) (int,int)`)
      returns all of them, and `q, r := divmod(...)` binds each positionally;
      using such a call in a single-value position is a descriptive error.
- [ ] A recursive factorial goal function returns the correct factorial
      (e.g. factorial(5) == 120).
- [ ] A recursive fibonacci goal function returns the correct fibonacci number
      (e.g. fib(10) == 55).
- [ ] Calling an undefined name yields the `*NotFoundError` ("undefined: name");
      calling a non-function value yields a descriptive "cannot call" error; an
      argument-count mismatch yields a descriptive error naming the counts.

## User Interactions

No direct user interface. The behavior is observed by Go unit tests in
internal/interp that parse + sema-resolve a small goal program, run it through the
interpreter, and assert returned values.

## Error Handling

- Undefined callee: the existing `*NotFoundError` ("undefined: name").
- Non-function callee: a descriptive "interp: cannot call <kind>" error.
- Arity mismatch: a descriptive "interp: ... expects N args, got M" error.
All surface as the error returned from evaluation, never a panic or silent value.

## Out of Scope

- Methods / receiver dispatch (US-018).
- Closures over local function literals / anonymous functions — goal has no
  func-literal expressions in the corpus; only top-level declared functions.
- Host/stdlib function calls (US-011).
- The full control-flow suite — three-clause for, switch, break/continue (US-008).
  This story implements only the minimum `if`/`return` needed for recursive
  factorial and fibonacci.
- Variadic parameters.

## Open Questions

None.
