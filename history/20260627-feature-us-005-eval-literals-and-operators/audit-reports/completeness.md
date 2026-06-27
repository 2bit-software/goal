# Audit: Completeness — US-005

## Findings

- MINOR: The spec lists CHAR literals out of scope implicitly (only int/float/
  string/bool named). Go semantics for `'a'` would map to an int rune. Decision:
  evaluation of CHAR literals is a harmless bonus; not required by the AC and not
  blocking. Implementation may decode CHAR to an int value but tests need not
  cover it.
- MINOR: Mixed int/float arithmetic (e.g. `1 + 2.0`) is untyped-constant
  territory in real Go. The corpus expressions in tests use same-kind operands;
  the spec scopes operand kinds to numbers/strings/bools. Not blocking.
- None CRITICAL or MAJOR. Every FR maps to a verifiable AC; error cases
  (divide-by-zero, unsupported operand kinds) are specified.

## Assumptions

- Integer literals decode as signed int64 (base 0, so hex/octal/binary forms
  also accepted), floats as float64 — matching the Value model's Int/Float.
- Same-kind operands for binary numeric/comparison ops; cross-kind promotion is
  out of scope for this story.
