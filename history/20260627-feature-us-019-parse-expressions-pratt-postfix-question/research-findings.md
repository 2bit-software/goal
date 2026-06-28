# Research Findings — US-019

## Summary

Precedence-correct expression parsing for the Go subset is a solved, standard
problem; the canonical reference is Go's own `go/parser` (`parseBinaryExpr`
precedence-climbing loop) and the Go spec's operator precedence table. No
third-party research is needed — the AST nodes already exist in-repo and the
approach is the textbook precedence-climbing algorithm.

## Go operator precedence (authoritative — Go spec, "Operators")

Five binary precedence levels, all left-associative:

| Prec | Operators            |
|------|----------------------|
| 5    | `*  /  %  <<  >>  &  &^` |
| 4    | `+  -  |  ^`          |
| 3    | `==  !=  <  <=  >  >=` |
| 2    | `&&`                 |
| 1    | `||`                 |

Unary/prefix operators: `+  -  !  ^  *  &  <-`. Postfix selector/call/index bind
tighter than any binary op. goal adds postfix `?` (unwrap), which binds tightest
(applies to the fully-built postfix operand).

## Algorithm (precedence climbing — mirrors go/parser)

```
parseExpr()            -> parseBinary(1)
parseBinary(minPrec):
  x = parseUnary()
  loop:
    op = current; opPrec = precedence(op)
    if opPrec < minPrec: return x
    advance
    y = parseBinary(opPrec + 1)   // left-assoc
    x = BinaryExpr{x, op, y}
parseUnary():
  if current is unary op: advance; return UnaryExpr/StarExpr over parseUnary()
  else: return parsePostfix(parseOperand())  // includes ? in the chain
```

`<-` (ARROW) is treated as a unary receive operator, not a binary operator, to
match Go (channel send is a statement, not an expression in this subset).

## Confidence

High. The reference implementation is the Go standard library, the AST nodes
(`BinaryExpr`, `UnaryExpr`, `StarExpr`, `UnwrapExpr`) already exist, and the
existing parser already owns the `parseOperand`/`parsePostfix`/`exprLev`
machinery this layers onto.

## Open Questions

None blocking. The `?` precedence (tightest, postfix) is fixed by
REWRITE-ARCHITECTURE and the existing `UnwrapExpr` node doc comment.
