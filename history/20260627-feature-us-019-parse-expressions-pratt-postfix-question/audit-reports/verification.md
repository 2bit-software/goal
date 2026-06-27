# Implementation Verification — US-019

## Gates (prd.json verifyCommands)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages green)

## Acceptance Criteria (business-spec.md)
- [x] `f(x)?` → `*ast.UnwrapExpr` wrapping `*ast.CallExpr` — asserted by
      `TestParseExpressionPrecedence/unwrap_call`.
- [x] `a.b?` → `*ast.UnwrapExpr` wrapping `*ast.SelectorExpr` — asserted by
      `unwrap_selector`.
- [x] `a + b * c == d` → `(a + (b * c)) == d` — asserted by `mixed_precedence`.
- [x] `a - b - c` left-nested — asserted by `left_associative`.
- [x] `-a * b` → `(-a) * b` (unary tighter than binary) — asserted by
      `unary_tighter_than_binary`.
- [x] No regression — full suite green; existing parser tests unchanged.

## Findings
None CRITICAL/MAJOR. The implementation matches the plan exactly: `precedence`
table, `parseBinary` precedence-climbing loop, `parseUnary` prefix handling,
`UnwrapExpr` postfix case, broadened `startsExpr`. `exprLev` composite
suppression untouched, so control-header parsing is unaffected.

## Assumptions (validated)
- `?` binds tightest (postfix chain), so `-x?` is `-(x?)`. Matches UnwrapExpr
  node design.
- `<-` is unary-only in expressions; channel send remains a statement.
- Left associativity for all binary levels (Go semantics).
- `*x`/`&x` map to `StarExpr`/`UnaryExpr` per go/ast conventions.
