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
//  1. Result  — Result[T, error] signatures, Ok/Err returns, statement-position match.
//  2. Option  — Option[T] -> *T, Some/None returns.
//  3. Question — `?` propagation, mode recovered by function name from the tables.
var Passes = []Pass{
	{Name: "result", Run: pass.Result},
	{Name: "option", Run: pass.Option},
	{Name: "question", Run: pass.Question},
}

// Transpile lowers goal source to formatted Go by running every pass in order and
// formatting once. The returned error names the failing pass, or reports the
// generated Go alongside the gofmt error when the final source does not parse.
func Transpile(src string) (string, error) {
	tables := analyze.Build(src)
	cur := src
	for _, p := range Passes {
		next, err := p.Run(cur, tables)
		if err != nil {
			return "", fmt.Errorf("pass %s: %w", p.Name, err)
		}
		cur = next
	}
	formatted, err := format.Source([]byte(cur))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, cur)
	}
	return string(formatted), nil
}
