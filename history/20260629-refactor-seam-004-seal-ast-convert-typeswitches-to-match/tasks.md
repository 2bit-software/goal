# Tasks — SEAM-004

1. Seal the 5 category interfaces + add `implements` to all concrete + support
   types; drop marker methods (selfhost/ast/*.goal). Verify gates.
2. Convert AST-family type-switches to `match` in selfhost/sema (check, resolve,
   question, fields, mustuse, assert). Verify.
3. Convert AST-family type-switches to `match` in selfhost/parser (parser,
   goal_construct) and selfhost/backend (lower, emit; over-one-sealed only). Verify.
4. Document kept-plain switches (foreign go/ast, emit sibling-category, walk) and
   resolve the go/ast-mirror oracle in DECISIONS.md.
5. Run all gates: task check, task build, task fixpoint; confirm corpus tiers green.
