// Package extpkg is a fixture foreign Go package: analyze's foreign-type tests read its
// exported struct field sets the way EnrichForeign reads an imported package. It is under
// testdata so the go tool never builds it as part of the project.
package extpkg

// Outer exercises every field shape the foreign extractor must render: a predeclared
// scalar, a package-local pointer and slice, a slice-of-pointer, a map, and an
// unexported field that must be skipped.
type Outer struct {
	ID     string
	Count  int
	Inner  *Inner
	Tags   []string
	Items  []*Inner
	ByName map[string]*Inner
	hidden string //nolint:unused // must be excluded from the extracted field set
}

// Inner is a package-local struct referenced by Outer, to verify local type references
// are qualified by the import alias.
type Inner struct {
	Label string
}

// unexported must never appear in the extracted set.
type unexported struct{ x int } //nolint:unused

// Mkdir is an error-only foreign function: its `?` callee arity must read as 1.
func Mkdir(p string) error { return nil }

// Open is a (value, error) foreign function: its `?` callee arity must read as 2.
func Open(p string) (int, error) { return 0, nil }

// Triple returns three values: its `?` callee arity must read as 3.
func Triple() (int, int, error) { return 0, 0, nil }

// hidden is unexported and must never contribute a foreign arity.
func hidden() error { return nil } //nolint:unused

// Close has a receiver and must be skipped — only package-level functions are recorded.
func (Inner) Close() error { return nil }
