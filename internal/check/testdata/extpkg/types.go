// Package extpkg is a fixture foreign Go package for the checker's foreign-derive test.
// It lives under testdata (ignored by the go tool) and outside testdata/check (so the
// golden-case walk in check_test.go never treats it as a case).
package extpkg

// Outer is the foreign source struct the checker reads to prove a derive total.
type Outer struct {
	ID    string
	Count int
}
