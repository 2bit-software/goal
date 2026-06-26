---
status: complete
updated: 2026-06-25
---

# Spike Results: `goal fix`

## Spike 1 — Validate the transformation target round-trips through the transpiler

**Question**: Is the idiomatic goal I intend `fix` to *produce* valid, and does it preserve
the semantics of the plain-Go input?

**Method**: Hand-wrote a plain-Go-style `before.goal` and its idiomatic `after.goal`, then
ran each through `go run ./cmd/goalc`.

`before.goal`:
```goal
func load(p string) ([]byte, error) {
	f, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
func describe(p string) (int, error) {
	data, err := load(p)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}
```

`after.goal` (the `fix` target):
```goal
func load(p string) Result[[]byte, error] {
	f := os.ReadFile(p)?
	return Result.Ok(f)
}
func describe(p string) Result[int, error] {
	data := load(p)?
	return Result.Ok(len(data))
}
```

**Findings (both transpiled, exit 0):**

1. `before.goal` is already valid goal (goal ⊃ Go) and lowers to itself.
2. `after.goal` lowers to:
   ```go
   func load(p string) (__goal_ok []byte, __goal_err error) {
       f, __goal_err := os.ReadFile(p)
       if __goal_err != nil { return __goal_ok, __goal_err }
       return f, nil
   }
   ```
   i.e. **the lowering of `after` reproduces `before`'s exact propagation shape.** `fix` is
   the *inverse* of the `?`/Result lowering. This is the correctness backbone: for each
   shape `fix` collapses, the transpiler's existing pass expands it right back, so golden
   tests can assert `lower(fix(before)) ≈ lower(before)`.
3. `?` collapse and `Result.Ok/Err` were confirmed against `internal/pass/question.go` and
   `result.go` templates — no guessing about output syntax.

**Conclusion**: The transformation target is correct and well-defined. Confirmed that the
plain-Go `(T,error)`+propagation → `?` case **requires the signature conversion** (the body
collapse is only legal once the function returns `Result`), so P2 (signature) and P1 (body
collapse) are coupled for the user's primary example; the caller ripple (`describe` calling
`load`) is contained because both convert together within the package set.

## Decision implied by the spike

The `Fix` orchestrator should apply fixers to a **fixed point** (re-lex + re-run until no
change, bounded iteration count): signature conversion in pass 1 marks a function `Result`,
which makes its body's propagation legal to collapse in pass 2. This yields FR-011
(fixed-point) for free and removes any need to hand-order interdependent fixers.
