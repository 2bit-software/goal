package check

import "goal/internal/analyze"

// checkExhaustive enforces feature 02 (match): a `match` over an enum (or
// Result/Option) must cover every variant, or supply an explicit `_` rest-arm. A
// match missing a variant and lacking `_` is an Error — the very case the lowering
// would otherwise turn into a silent `panic("unreachable: …")` default.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (02-match) and prompt.md for the loop.
//
// Reuse, don't reinvent:
//   - The match pass (internal/pass/match.go) already locates `match` blocks, their
//     scrutinee, arm qualifiers, and the `_` rest-arm; scan.MatchQualifier /
//     scan.MatchBodyBrace are the shared locators.
//   - Variant sets: analyze.Tables.Enums[…].VSet. Resolve the scrutinee's enum type
//     from the function signature / construction context.
//   - Must run pre-lowering: the type switch erases which variants were covered.
//
// Defer-boundary: when the scrutinee's type cannot be resolved lexically (untyped
// `x := match …`, value-position match needing an inferred type), emit a located
// Warning naming the unresolved scrutinee — do not assume exhaustiveness.
func checkExhaustive(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}
