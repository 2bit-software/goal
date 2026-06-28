# Audit: Completeness — US-019

## Findings

### MINOR — `<-` operator role
The spec lists `<-` as a prefix/unary operator (FR-2) and excludes it as a binary
op. This matches Go (channel receive is unary; send is a statement). Documented;
no action needed.

### MINOR — `?` after a non-postfixable operand
Spec says `?` follows "an operand (after its postfix chain)". A `?` after a bare
literal (e.g. `5?`) is syntactically accepted by the grammar but semantically
meaningless; semantic rejection is a checker concern, out of scope here. No
action.

### MINOR — Precedence of `?` vs unary
`-x?` — does `?` bind to `x` or to `-x`? Since `?` is part of the postfix chain
(tightest) and unary is parsed around the postfix operand, `?` binds to `x`,
giving `-(x?)`. This is the natural Go-like reading. Acceptable; can be pinned by
a test if desired but not required by the AC.

## Conclusion

No CRITICAL or MAJOR findings. The spec is complete and testable. Recommend PASS.

## Assumptions

- `?` binds tighter than unary (it is in the innermost postfix chain), so `-x?`
  is `-(x?)`. Matches the existing UnwrapExpr node design.
- `<-` is unary-only in the expression grammar; channel send remains a statement.
- Left associativity for all binary levels (Go semantics).
