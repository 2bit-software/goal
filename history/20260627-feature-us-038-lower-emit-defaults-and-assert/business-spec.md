# Business Spec — US-038

## What

A goal developer needs two erased/lowered constructs to work on the AST backend:

1. **`...defaults`** inside a struct composite literal expands to explicit
   per-field zero values for every field not already set, so the generated Go
   literal is complete. A field whose zero is unsafe (nil pointer/map/chan/func,
   a sum type, a method-bearing interface) is a located error, not a silent zero.

2. **`assert <cond>`** lowers to a runtime `if !(cond) { panic("assertion
   failed: <cond text>") }`. The printf-message form `assert <cond>, <fmt>,
   <args>...` appends `: " + fmt.Sprintf(<fmt>, <args>...)` to the panic message
   and the `fmt` import is injected when needed.

## Acceptance

- lower+backend expand `...defaults` to explicit zero values and `assert` to an
  if-panic guard.
- The 08-no-zero-value and 10-assert transpile cases pass the behavioral tier
  (go build + go vet of the generated Go) through the new (AST) engine.
