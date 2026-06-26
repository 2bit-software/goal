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
	"strings"

	"goal/internal/analyze"
	"goal/internal/pass"
	"goal/internal/project"
)

// Pass is one named lowering step over the source.
type Pass struct {
	Name string
	Run  func(src string, t *analyze.Tables) (string, error)
}

// Passes is the ordered front-end pipeline.
//
//  0. StoredResult — guard: reject a Result[T,E] stored as a value (slice/map/array
//     element, struct/enum field) with a located §8.7 error, before any Result lowering.
//  1. Implements  — strip a struct's inline `implements` clause, emitting a compile-time
//     assertion per ordinary interface and a marker method per sealed interface.
//  2. Defaults    — `...defaults` -> explicit per-field zero values.
//  3. Result      — Result[T, error] signatures, Ok/Err returns, statement match.
//  4. Option      — Option[T] -> *T, Some/None returns, statement match.
//  5. Question     — open-E `?` / Option `?` (closed-E `?` is skipped for pass 6).
//  6. ResultClosed — closed-E Result: sum constructors, `match`, `?`, From-conversion.
//  7. Derive       — `from func` strip + `derive func` field-by-field expansion.
//  8. Assert       — `assert` -> runtime `if !(cond) { panic(...) }`.
//  9. Match        — enum `match` -> type-switch over the §8.1 encoding.
// 10. Enums        — enum/sealed declarations -> encoding, variant constructions.
//
// The independent declaration/statement transforms (1-2, 6) touch disjoint
// constructs and could run anywhere; they are grouped to mirror the spec's pass
// order. Match precedes Enums: a variant construction and an enum match pattern share
// the surface `Enum.Variant(...)`, so the match pass must consume the patterns before
// the enums pass rewrites the remaining (genuine) constructions. Implements emits any
// sealed-interface marker method as plain Go that later passes leave untouched; the
// sealed interface declaration itself is still emitted by Enums, so the two passes'
// order relative to each other does not matter.
var Passes = []Pass{
	{Name: "storedresult", Run: pass.StoredResultGuard},
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
	return transpileWith(src, analyze.Build(src))
}

// transpileWith is Transpile's core with the tables supplied by the caller: the
// single-file path passes analyze.Build(src); the package path passes the merged tables
// (analyze.BuildPackage) with SuppressResultPrelude set, so cross-file references
// resolve and the prelude is emitted once by the package driver rather than per file.
func transpileWith(src string, tables *analyze.Tables) (Output, error) {
	lowered, err := runPasses(src, tables)
	if err != nil {
		return Output{}, err
	}
	formatted, err := format.Source([]byte(lowered))
	if err != nil {
		return Output{}, fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, lowered)
	}
	test, err := doctestFile(src, tables)
	if err != nil {
		return Output{}, fmt.Errorf("doctests: %w", err)
	}
	return Output{Go: string(formatted), Test: test}, nil
}

// runPasses threads src through every front-end pass in order, naming the failing pass on
// error. It does not format — an intermediate source need only be lexable.
func runPasses(src string, tables *analyze.Tables) (string, error) {
	cur := src
	for _, p := range Passes {
		next, err := p.Run(cur, tables)
		if err != nil {
			return "", fmt.Errorf("pass %s: %w", p.Name, err)
		}
		cur = next
	}
	return cur, nil
}

// doctestFile extracts doctests from the untouched source (`///` comments are erased by
// the lexer the passes use, so extraction must run on the original text), renders them as
// a goal-shaped `_test.go`, and lowers that through the SAME passes and tables as the
// source — so a doctest over enum variants, keyed struct literals, or Result/Option
// constructors lowers to the same Go the function body would. It returns "" when the
// source has no doctests.
func doctestFile(src string, tables *analyze.Tables) (string, error) {
	goalTest := pass.RenderDoctests(src, pass.ExtractDoctests(src))
	if goalTest == "" {
		return "", nil
	}
	lowered, err := runPasses(goalTest, tables)
	if err != nil {
		return "", err
	}
	formatted, err := format.Source([]byte(lowered))
	if err != nil {
		return "", fmt.Errorf("generated test file did not parse: %w\n--- generated ---\n%s", err, lowered)
	}
	return string(formatted), nil
}

// GoFile is one generated Go source: the base file name to write and its formatted
// content. Names are derived from the originating `.goal` file (foo.goal -> foo.go,
// foo_test.go for its doctest sidecar); the synthesized prelude is goal_prelude.go.
type GoFile struct {
	Name string
	Go   string
}

// PackageOutput is the full Go output for one goal package, held in memory (no disk
// I/O): one Go file per source, the optional shared goal_prelude.go, and any doctest
// sidecars. The build driver (U6) decides whether to compile this from a temp dir or
// persist it via --emit.
type PackageOutput struct {
	Files []GoFile // transpiled sources, plus goal_prelude.go when the package uses closed-E Result
	Tests []GoFile // doctest sidecars (`_test.go`), one per source file that has doctests
}

// TranspilePackage lowers every file in a package against one set of merged, name-keyed
// tables, so a file resolves enums/structs/from-funcs/signatures declared in a sibling
// (U2), and emits the closed-E Result prelude exactly once for the package (U3). It does
// no disk I/O — it returns the Go in memory.
func TranspilePackage(pkg *project.Package) (PackageOutput, error) {
	srcs := make([]string, len(pkg.Files))
	for i, f := range pkg.Files {
		srcs[i] = f.Src
	}
	tables := analyze.BuildPackage(srcs)
	tables.SuppressResultPrelude = true // the package emits one prelude below, not one per file

	var out PackageOutput
	for _, f := range pkg.Files {
		res, err := transpileWith(f.Src, tables)
		if err != nil {
			return PackageOutput{}, fmt.Errorf("%s: %w", f.Name, err)
		}
		gen := goName(f.Name)
		// Map generated decls back to the .goal file so toolchain errors land on source
		// positions (U5); the synthesized prelude keeps no directives (errors there are
		// compiler bugs, honestly reported against goal_prelude.go).
		mapped := addLineDirectives(f.Src, res.Go, f.Name, gen)
		out.Files = append(out.Files, GoFile{Name: gen, Go: mapped})
		if res.Test != "" {
			out.Tests = append(out.Tests, GoFile{Name: testName(f.Name), Go: res.Test})
		}
	}
	if pass.NeedsResultPrelude(tables) {
		preludeGo, err := preludeFile(pkg.Name)
		if err != nil {
			return PackageOutput{}, fmt.Errorf("prelude: %w", err)
		}
		out.Files = append(out.Files, GoFile{Name: "goal_prelude.go", Go: preludeGo})
	}
	return out, nil
}

// goName maps a source file name to its generated Go name: foo.goal -> foo.go.
func goName(goalName string) string {
	return strings.TrimSuffix(goalName, project.Ext) + ".go"
}

// testName maps a source file name to its doctest sidecar: foo.goal -> foo_test.go.
func testName(goalName string) string {
	return strings.TrimSuffix(goalName, project.Ext) + "_test.go"
}

// preludeFile is the standalone goal_prelude.go for a package: the package clause plus
// the generic closed-E Result sum encoding, formatted.
func preludeFile(pkgName string) (string, error) {
	src := "package " + pkgName + "\n\n" + pass.ResultPreamble + "\n"
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
