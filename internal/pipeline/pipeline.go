// Package pipeline is the unified front-end driver: it builds the name-keyed tables
// once, threads the source string through an ordered list of lowering passes, and
// formats the result exactly once at the end.
//
// Pass order is a dependency graph. Signature/type lowering (Result, Option) must
// precede the control-flow lowering that depends on it (`?`), so that the `?` pass
// sees the lowered (T, error) / *T shapes and recovers each function's original mode
// from the tables by name. No pass formats its own output; an intermediate source is
// only required to be lexable, not gofmt-parseable.
package pipeline

import (
	"fmt"
	"go/format"

	"goal/internal/analyze"
	"goal/internal/pass"
)

// Pass is one named lowering step over the source.
type Pass struct {
	Name string
	Run  func(src string, t *analyze.Tables) (string, error)
}

// Passes is the ordered front-end pipeline.
//
//  1. Pure       — strip the `pure func` modifier.
//  2. Implements  — strip a struct's inline `implements` clause, emitting a compile-time
//     assertion per ordinary interface and a marker method per sealed interface.
//  3. Defaults    — `...defaults` -> explicit per-field zero values.
//  4. Result      — Result[T, error] signatures, Ok/Err returns, statement match.
//  5. Option      — Option[T] -> *T, Some/None returns, statement match.
//  6. Question     — open-E `?` / Option `?` (closed-E `?` is skipped for pass 7).
//  7. ResultClosed — closed-E Result: sum constructors, `match`, `?`, From-conversion.
//  8. Derive       — `from func` strip + `derive func` field-by-field expansion.
//  9. Assert       — `assert` -> runtime `if !(cond) { panic(...) }`.
//
// 10. Match        — enum `match` -> type-switch over the §8.1 encoding.
// 11. Enums        — enum/sealed declarations -> encoding, variant constructions.
//
// The independent declaration/statement transforms (1-3, 7) touch disjoint
// constructs and could run anywhere; they are grouped to mirror the spec's pass
// order. Match precedes Enums: a variant construction and an enum match pattern share
// the surface `Enum.Variant(...)`, so the match pass must consume the patterns before
// the enums pass rewrites the remaining (genuine) constructions. Implements emits any
// sealed-interface marker method as plain Go that later passes leave untouched; the
// sealed interface declaration itself is still emitted by Enums, so the two passes'
// order relative to each other does not matter.
var Passes = []Pass{
	{Name: "pure", Run: pass.Pure},
	{Name: "implements", Run: pass.Implements},
	{Name: "defaults", Run: pass.Defaults},
	{Name: "result", Run: pass.Result},
	{Name: "option", Run: pass.Option},
	{Name: "question", Run: pass.Question},
	{Name: "closed", Run: pass.ResultClosed},
	{Name: "derive", Run: pass.Derive},
	{Name: "assert", Run: pass.Assert},
	{Name: "match", Run: pass.Match},
	{Name: "enums", Run: pass.Enums},
}

// Output is the set of files the front-end produces from one source: the lowered Go,
// and an optional sibling `_test.go` extracted from doctests (empty when the source
// has none). Doctests are a side output — the driver supports N outputs, not one.
type Output struct {
	Go   string
	Test string
}

// Transpile lowers goal source to formatted Go by running every pass in order and
// formatting once, and separately extracts any doctests from the ORIGINAL source into
// a sibling test file. The returned error names the failing pass, or reports the
// generated Go alongside the gofmt error when the final source does not parse.
func Transpile(src string) (Output, error) {
	tables := analyze.Build(src)
	cur := src
	for _, p := range Passes {
		next, err := p.Run(cur, tables)
		if err != nil {
			return Output{}, fmt.Errorf("pass %s: %w", p.Name, err)
		}
		cur = next
	}
	formatted, err := format.Source([]byte(cur))
	if err != nil {
		return Output{}, fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, cur)
	}
	// Doctests read the untouched source: `///` comments are erased by the lexer the
	// passes use, so extraction must run on the original text, not the lowered Go.
	test, err := pass.Doctests(src)
	if err != nil {
		return Output{}, fmt.Errorf("doctests: %w", err)
	}
	return Output{Go: string(formatted), Test: test}, nil
}
