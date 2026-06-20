package check

import "goal/internal/analyze"

// checkImplements enforces feature 07 (implements): for `type T struct implements I`,
// the type T must actually satisfy interface I — every method I declares must exist
// on T with a matching signature. A missing or mismatched method is an Error.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (07-implements) and prompt.md for the loop.
//
// Reuse, don't reinvent:
//   - The implements pass (internal/pass/implements.go) already locates the inline
//     `implements I, J` clause; analyze.Tables.Sealed tells a sealed interface (a
//     marker method, trivially satisfied) from an ordinary one (real method-set
//     obligation).
//   - The ordinary case needs the interface's method set and T's declared methods.
//     analyze.Tables carries struct/type decls but not a method index yet — extend
//     the Tables with one (a name-keyed method index built once) rather than
//     re-scanning per check. Record the Tables extension in DECISIONS.md.
//
// Defer-boundary: signature equality across type aliases / unexported embedding can
// be ambiguous lexically — when a method's match cannot be decided, emit a located
// Warning rather than a false Error.
func checkImplements(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}
