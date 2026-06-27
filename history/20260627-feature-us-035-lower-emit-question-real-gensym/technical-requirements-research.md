# Technical Requirements / Research — US-035

- Work in internal/backend (emit.go, lower.go); the AST engine, not the splice passes.
- Scope-aware gensym: seed a collision set from the file's identifiers; allocate
  function-scoped names (success/error returns) and local temporaries (match, `?`)
  that never collide with source identifiers — the structural replacement for the
  fixed `__goal_` prefix used by US-034.
- `?` (ast.UnwrapExpr) is lowered at statement position: `name := expr?`,
  `_ := expr?`, and bare `expr?`. Result functions propagate the function's
  generated (ok, err) returns; Option functions return nil and deref into a fresh
  pointer temp. Destructure arity follows the callee's sema.FuncSig (a plain
  error-returning callee yields one value; a Result callee two).
- Behavioral tier (corpus.RunCompile) over features/05-question-prop is the witness.
