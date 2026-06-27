package corpus

import (
	"fmt"
	"os"
	"path/filepath"
)

// Parser parses goal source and reports whether it parsed without error. It is
// the parser counterpart of [Transpiler] / [Checker]: the seam that lets the
// corpus judge any front-end parser against the whole golden corpus. The new AST
// front-end's parser.ParseFile satisfies it via [ParserFunc].
type Parser interface {
	Parse(src string) error
}

// ParserFunc adapts a plain parse function to the [Parser] interface, so a free
// function can be passed where a Parser is expected. Wrap the AST parser as:
//
//	corpus.ParserFunc(func(src string) error { _, err := parser.ParseFile(src); return err })
type ParserFunc func(src string) error

// Parse calls the wrapped function.
func (f ParserFunc) Parse(src string) error { return f(src) }

// CaseInputs returns the repo-relative paths of the .goal source file(s) backing
// a case: the single [Case.Input] for a Mode=file case, or every [PackageSpec]
// file for a Mode=package case.
func CaseInputs(c Case) []string {
	if c.Mode == ModePackage && c.Package != nil {
		return append([]string(nil), c.Package.Files...)
	}
	return []string{c.Input}
}

// RunParse parses every .goal source file backing c through p. It returns a
// descriptive error naming the first input that fails to parse (or that cannot
// be read); nil means every backing input parsed with zero errors.
func RunParse(root string, c Case, p Parser) error {
	for _, in := range CaseInputs(c) {
		src, err := os.ReadFile(filepath.Join(root, in))
		if err != nil {
			return fmt.Errorf("read %s: %w", in, err)
		}
		if err := p.Parse(string(src)); err != nil {
			return fmt.Errorf("parse %s: %w", in, err)
		}
	}
	return nil
}
