// Package sema holds the semantic information the AST back-ends consume: the
// name-keyed facts (enums, structs, signatures, the from-registry, methods)
// derived by walking the goal AST, plus the correctness checks layered on them.
//
// This is the Phase 2 replacement for the token-scanning internal/analyze: where
// analyze rebuilds structure from a flat token stream, sema reads it off the parsed
// tree. For now Info is a placeholder seam so the Backend interface
// (Emit(*ast.File, *sema.Info)) can be expressed; US-027 populates it by AST walk
// and the later sema stories add the checks.
package sema

// Info is the resolved semantic information for one goal file (or package): the
// name-keyed symbol facts a back-end needs to lower goal constructs.
//
// It is intentionally empty for now. US-027 ("Resolve symbols by AST walk") fills
// it with the enums/structs/signatures/from-registry/methods that internal/analyze
// produces today, derived structurally from the AST instead of by token scanning.
type Info struct {
	// Populated by US-027 onward. Kept as a struct (not a bare alias) so fields can
	// be added without changing the Backend.Emit signature that references *Info.
}

// New returns an empty *Info. Back-ends that do not yet need semantic facts (the
// plain-Go subset) call New to satisfy the Backend.Emit signature.
func New() *Info {
	return &Info{}
}
