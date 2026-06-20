package check

import "goal/internal/analyze"

// checkConvert enforces feature 12 (derive-convert): a `derive func g(s S) T` must be
// total — every field of the target T must be reachable, either field-by-field from S
// or via a registered `from func`, honoring the exception clause (ignore / rename /
// explicit per-field). A target field with no resolvable source and no exception is an
// Error.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (12-derive-convert) and prompt.md for the loop.
//
// Reuse, don't reinvent:
//   - The derive pass (internal/pass/derive.go) already walks `derive func`, the
//     target/source field correspondence, and the exception clause.
//   - analyze.Tables.Structs gives both field lists; analyze.Tables.FromRegistry
//     resolves a field whose types differ. This generalizes the 06 totality check.
//
// Defer-boundary: feature-12 recursion into map/Option/nested-struct fields and the
// two bespoke shapes (pmk_upgrade, patterns JSON) are deferred per the audit — emit a
// located Warning for an unresolvable field rather than a false incompleteness Error.
func checkConvert(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}
