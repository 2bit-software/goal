// Package extpkg is a fixture foreign Go package for the LSP's foreign-derive test.
// It lives under testdata (ignored by the go tool) so the foreign resolver can read
// its exported struct the way sema.EnrichForeign reads an imported package.
package extpkg

// Outer is the foreign source struct the checker reads to prove a derive total.
type Outer struct {
	ID    string
	Count int
}
