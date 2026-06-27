# Plan Audit: Completeness — US-019

## Coverage of spec requirements

- FR-1 (binary precedence) → `precedence` + `parseBinary`. Covered.
- FR-2 (unary/prefix) → `parseUnary` (+ StarExpr for `*x`). Covered.
- FR-3 (postfix binds tightest) → preserved existing `parsePostfix` chain,
  `parseUnary` wraps the postfix operand. Covered.
- FR-4 (postfix `?`) → `QUESTION` case in `parsePostfix` → `UnwrapExpr`. Covered.
- FR-5 (no regression) → `exprLev` preserved, single `parseExpr` entry point.
  Covered.

## Findings

### MINOR — `startsExpr` broadening scope
`parseReturnStmt` uses `startsExpr(p.kind())` to decide whether results follow.
The plan broadens `startsExpr` to include unary/`*`/`&` starts; verify this does
not misclassify a `*`/`&` that begins a *type* in some other caller. Audit:
`startsExpr` is only consulted by `parseReturnStmt`, so the blast radius is just
"does an expression follow `return`," which is exactly correct. No issue.

### MINOR — `?` after literal
`5?` parses to `UnwrapExpr` over a BasicLit. Semantically meaningless but
syntactically harmless; rejection is a checker concern. Out of scope. No action.

## Conclusion

No CRITICAL or MAJOR findings. Plan is complete and traces every requirement.
Recommend PASS.

## Assumptions

- Left-associative binaries via `parseBinary(opPrec+1)`.
- `<-` is unary-only; channel send stays a statement.
- `?` lives in the postfix chain (tightest), so `-x?` is `-(x?)`.
