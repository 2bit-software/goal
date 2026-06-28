# Plan Audit — Completeness

## Findings

No CRITICAL or MAJOR findings. The plan covers every acceptance criterion:
function-as-value registration, param binding in a fresh per-call scope,
multi-return via `[]Value` + the `execAssign` branch, recursion (root
registration before run), and the loud-refusal error cases. The minimum `if`/
`return` needed for factorial/fibonacci is explicitly in scope; full control flow
is correctly deferred to US-008.

- MINOR: The plan keeps the existing name-only `FuncVal(name)` constructor for
  value_test.go compatibility and adds a new decl-carrying constructor. Confirm
  during implement that no existing caller of `FuncVal` expected a callable.
- MINOR: `returnSignal` modelled as an `error` sentinel — standard tree-walker
  technique; ensure it never escapes `evalCallMulti` as a real error.

## Assumptions
- Top-level function registration happens in `New` (load-time), not lazily; this
  is what makes both recursion and the direct-`evalExpr` test seam work.
- goal has no anonymous/func-literal expressions in scope; only declared
  top-level functions are callable values (consistent with the corpus).
