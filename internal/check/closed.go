package check

import "goal/internal/analyze"

// checkClosed enforces feature 06 (closed-E Result): for a closed-E `Result[T, E]`
// (E is a named sum, not the builtin error), the error side must stay closed — every
// error value flowing into it must be a known variant of E — and any `?` propagation
// across error types must have a total `from func` conversion registered. A `?` whose
// From-conversion is missing, or an Err of a type outside E, is an Error.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (06-error-e) and prompt.md for the loop.
//
// Reuse, don't reinvent:
//   - analyze.Tables.FuncSignatures gives each function's Mode (ModeResultClosed) and
//     its T/E; the closed pass (internal/pass/closed.go) shows how a `?` site and the
//     enclosing function's E are paired.
//   - analyze.Tables.FromRegistry is the (src, target) -> ConvEntry conversion table;
//     totality is "every E' that reaches a `?` has a registry entry to the enclosing E."
//   - analyze.Tables.Enums[E].VSet is the closed variant set for the Err side.
//
// Defer-boundary: when the propagated error's concrete type cannot be resolved at the
// `?` site, emit a located Warning rather than asserting a missing conversion.
func checkClosed(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}
