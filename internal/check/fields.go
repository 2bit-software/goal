package check

import "goal/internal/analyze"

// checkFields enforces feature 08 (no-zero-value): every struct or enum-variant
// composite literal must name a value for every field, unless it opts out with the
// `...defaults` spread. A literal that omits a field without the spread is an Error.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (08-no-zero-value) and prompt.md for the loop.
//
// Reuse, don't reinvent:
//   - The defaults pass (internal/pass/defaults.go) already locates every `T{…}`
//     literal and recognizes the `...defaults` spread — lift its locating logic and
//     assert completeness instead of expanding.
//   - Required field sets: analyze.Tables.Structs (ordered fields per struct) and
//     analyze.Tables.Enums[…].FieldSet (field-name set per variant).
//   - Type at the construction site is usually named (`T{…}`, `Enum.Variant{…}`), so
//     no inference is needed for the common case.
//
// Defer-boundary: when the literal's type cannot be resolved lexically (a literal
// fed positionally to a func, a `:=` whose type isn't at the site), emit a located
// Warning naming what could not be resolved — do not assume the type.
func checkFields(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}
