// Package extpkg is a fixture foreign Go package imported by the pipeline's end-to-end
// foreign-derive test. It lives under testdata so the go tool never builds it.
package extpkg

// Outer is the foreign source struct a `derive func` converts from.
type Outer struct {
	ID    string
	Count int
}
