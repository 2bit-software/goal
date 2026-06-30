// Package extpkg is a fixture foreign Go package imported by the corpus's
// end-to-end foreign-derive case. It lives under testdata so the go tool never
// builds it. Relocated here from the deleted legacy internal/pipeline/testdata
// when the hand-written Go transpiler was removed (US-019).
package extpkg

// Outer is the foreign source struct a `derive func` converts from.
type Outer struct {
	ID    string
	Count int
}

// Mkdir is an error-only foreign function used to prove arity-aware `?` lowering compiles.
func Mkdir(p string) error { return nil }
