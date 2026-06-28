# US-017 Eval question-mark unwinding — Business Specification

## Overview

The goscript tree-walking interpreter gains the postfix `?` operator, the second
of the two genuinely non-Go runtime mechanics (the first being value-position
`match`). `?` propagates failures: it unwraps a success and continues, or it
short-circuits the enclosing function with its own error/none return. This lets
real goal programs that use `?`-based error propagation run under interpretation
with the same observable behavior as the Go backend.

## Functional Requirements

### FR-1: Unwrap-and-continue on success
`expr?` where `expr` evaluates to `Result.Ok(v)` yields the unwrapped value `v`,
and execution proceeds to the next statement. The same applies to
`Option.Some(v)`, yielding `v`.

### FR-2: Early return on Result.Err
`expr?` where `expr` evaluates to `Result.Err(e)` performs a non-local early
return from the enclosing function, which returns `Result.Err(e)` (its own
error result). No further statements in the enclosing function run.

### FR-3: Early return on Option.None
`expr?` where `expr` evaluates to `Option.None` early-returns the enclosing
function's `Option.None`.

### FR-4: Closed-E `from` conversion during propagation
When the enclosing function returns a closed-E `Result[T, E]` (E is an enum, not
`error`) and the propagated error's type differs from E, the registered
`from func` conversion is applied to the error before the enclosing function
returns `Result.Err(convertedError)`.

### FR-5: Loud refusal outside a propagating function
Using `?` in a function whose return shape is neither Result nor Option is a
located, descriptive refusal rather than a silent value or nil.

## Acceptance Criteria

- [ ] `expr?` on `Result.Ok(v)` evaluates to `v` and continues.
- [ ] `expr?` on `Option.Some(v)` evaluates to `v` and continues.
- [ ] `expr?` on `Result.Err(e)` early-returns the enclosing function's
      `Result.Err(e)`; statements after the `?` do not run.
- [ ] `expr?` on `Option.None` early-returns the enclosing function's
      `Option.None`.
- [ ] For a closed-E Result whose callee error type differs from the enclosing
      function's error type, the `from func` conversion is applied to the
      propagated error (verified over the 06-error-e `from` shape).
- [ ] For a closed-E Result with the SAME error type, the error propagates
      unchanged (no conversion).
- [ ] `?` in a non-Result/Option function yields a descriptive error.

## User Interactions

No new CLI surface. The behavior is exercised by goal source run through the
interpreter (`internal/interp`).

### Test harness and oracle (audit resolution)

The unit tests follow the established interp test convention (result_test.go /
option_test.go): INLINE `const program` source strings driven through
`newInterp` + the `evalFn` helper, asserting on the returned `Value` shape
(Kind/Variant.Tag/payload). The `.goal` corpus fixtures are NOT loaded verbatim,
and the `.go.expected` files are backend Go output — not an interpreter oracle.

The named 05-question-prop / 06-error-e fixtures return `Result.Ok` / `Option.None`
UNCONDITIONALLY, so loading them would never reach the Err / None / conversion
branches. The tests therefore use programs MODELED ON those shapes but adapted so
each branch actually fires (e.g. a `parse` that returns `Result.Err(...)` on a
sentinel input, mirroring `qclosed_prop_same`'s `if s == "" { return Result.Err(...) }`;
an Option function that returns `None` for a missing key; and a closed-E `from`
program whose callee genuinely errs so `toApp` runs).

## Error Handling

- A `?` in a function whose return shape is neither Result nor Option (including
  a void function such as `main`, and a `?` with no enclosing propagating
  function on the signature stack) is a located, descriptive refusal carrying the
  `interp:` prefix. Tests assert on a substring, not exact wording.
- A `?` whose operand does not evaluate to a Result or Option variant is a
  located, descriptive refusal.
- A closed-E propagation whose required `from` conversion cannot be resolved
  (callee E differs from caller E and no `FromRegistry` entry exists) is a
  located, descriptive refusal — never a silent mistyped `Err`.
- The conversion trigger is resolved STATICALLY off the direct-call operand's
  callee `FuncSignatures[name].E` versus the enclosing caller `E`; when equal
  (or the callee E cannot be resolved and the caller is same-E) no conversion is
  applied. The interpreter otherwise trusts sema's well-typedness.

## Out of Scope

- The Go backend lowering of `?` (already implemented in internal/backend).
- `?` on a bare `func(...) error` callee (the `qprop_erronly` shape): handled
  best-effort (a nil error continues) but NOT an asserted acceptance shape.
- `?` used INSIDE a method body: methods push a none-shaped signature, so a `?`
  there hits the FR-5 refusal; method-position `?` propagation is out of scope.
- Capability routing of effects (US-023), implements dispatch (US-018), and the
  corpus behavioral gate (US-027).

## Open Questions

- None remaining. The propagation semantics and the `from`-conversion source are
  determined by existing sema facts (`FuncSignatures`, `FromRegistry`); the test
  oracle and branch-coverage approach are pinned in the Test harness section
  above following the iteration-1 audit.
