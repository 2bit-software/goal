package check

import "goal/internal/analyze"

// checkAssert enforces the static-provable subset of feature 10 (assert): an `assert`
// whose condition the checker can decide at compile time and prove false is an Error
// (a guaranteed panic), and a tautological always-true assert may be reported as a
// dead-code Warning. Conditions that are not statically decidable are left entirely to
// the runtime check the lowering already emits.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (10-assert) and prompt.md for the loop. The feature audit
// deliberately *reserved* (did not build) the static-checkable subset — keep this
// minimal and conservative; only flag what is provably constant.
//
// Reuse, don't reinvent:
//   - The assert pass (internal/pass/assert.go) locates each `assert cond, msg, …`.
//   - Only constant-foldable conditions are in scope here (literal comparisons,
//     boolean constants). Anything referencing a variable is out of scope by design.
//
// Defer-boundary: do not attempt general theorem proving. If the condition is not a
// trivially decidable constant, emit nothing — the runtime assert stands.
func checkAssert(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}
