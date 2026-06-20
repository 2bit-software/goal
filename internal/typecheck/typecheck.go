// Package typecheck is the depth-checker harness (Phase B of DEPTH-TODO): it loads a
// goal package's *lowered* Go into stdlib go/types so the per-guarantee depth checks can
// ask real type questions — identity, assignability, interface satisfaction, and the
// Defs/Uses flow primitive — that the lexical checker (internal/check, which runs on the
// original source) must defer.
//
// It rests on Phase A: pipeline.TranspilePackage produces a compilable Go package whose
// //line directives (U5) make go/parser/go/types report positions in the .goal source
// (SPIKE-B1). So a depth diagnostic is goal-located for free — see GoalPos.
//
// Zero-dependency, like the rest of the project: only stdlib go/parser, go/types,
// go/importer, go/token, go/ast. Flow facts come from types.Info, not an SSA library.
package typecheck

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	"goal/internal/analyze"
	"goal/internal/check"
	"goal/internal/pipeline"
	"goal/internal/project"
)

// Package is the type-checked view of one goal package's lowered Go: the go/types
// outputs (Fset, Types, Info, Files), the merged name-keyed goal tables (what goal said
// about each symbol, so a check knows which question to ask), and any type errors
// collected error-tolerantly. Errors being non-empty does not make Load fail — a check
// may still read partial type info, and genuine Go type errors are themselves
// goal-mappable diagnostics worth surfacing.
type Package struct {
	Fset   *token.FileSet
	Types  *types.Package
	Info   *types.Info
	Files  []*ast.File
	Tables *analyze.Tables
	Errors []error
	// Src is the original goal package (pre-lowering): some depth checks locate a
	// construct in the goal source (an `implements` clause, a `derive func`) and then
	// verify it against the type information, rather than reconstruct it from the AST.
	Src *project.Package
}

// Load transpiles a goal package, parses the lowered Go, and type-checks it with
// stdlib go/types under an error-collecting configuration. It returns an error only for
// a transpile or parse failure (a goal-compiler bug — the lowered Go must be valid Go);
// Go type errors in the user's program are collected into Package.Errors instead.
func Load(pkg *project.Package) (*Package, error) {
	out, err := pipeline.TranspilePackage(pkg)
	if err != nil {
		return nil, fmt.Errorf("transpile: %w", err)
	}

	srcs := make([]string, len(pkg.Files))
	for i, f := range pkg.Files {
		srcs[i] = f.Src
	}
	tables := analyze.BuildPackage(srcs)

	fset := token.NewFileSet()
	var files []*ast.File
	for _, gf := range out.Files {
		f, err := parser.ParseFile(fset, gf.Name, gf.Go, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse generated %s: %w", gf.Name, err)
		}
		files = append(files, f)
	}

	info := &types.Info{
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Types:      map[ast.Expr]types.TypeAndValue{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	p := &Package{Fset: fset, Info: info, Files: files, Tables: tables, Src: pkg}
	conf := types.Config{
		Importer: importer.Default(),
		Error:    func(e error) { p.Errors = append(p.Errors, e) },
	}
	// Check returns a usable (possibly incomplete) package even when Error fires.
	p.Types, _ = conf.Check(pkg.Name, fset, files, info)
	return p, nil
}

// Diagnostic is one depth-checker finding. Unlike the lexical checker's byte-offset
// Diagnostic, it carries a resolved token.Position (already in .goal coordinates), since
// a depth check works from go/types positions and source scans rather than one source
// string. Severity reuses the lexical checker's type so both stages render uniformly.
type Diagnostic struct {
	Pos      token.Position
	Severity check.Severity
	Feature  string
	Code     string
	Message  string
}

// String renders the diagnostic as `file:line:col: severity: [code] message`.
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s: %s: [%s] %s", d.Pos, d.Severity, d.Code, d.Message)
}

// goalPosition turns a byte offset into a goal source file into a token.Position. Used
// by checks that locate a construct in the source (e.g. an `implements` clause) and
// report there rather than at a go/types node.
func goalPosition(f project.File, off int) token.Position {
	p := check.OffsetToPosition(f.Src, off)
	return token.Position{Filename: f.Path, Line: p.Line, Column: p.Col}
}

// GoalPos returns the .goal source position of an AST node, resolved through the //line
// directives the lowered Go carries. The Filename is the goal file and the line is its
// source line (per-declaration accurate; see DEPTH-TODO / U5).
func (p *Package) GoalPos(n ast.Node) token.Position {
	return p.Fset.Position(n.Pos())
}

// Lookup returns the package-scope object named name, or nil. A user declaration keeps
// its goal name through lowering, so checks look symbols up by the name goal used.
func (p *Package) Lookup(name string) types.Object {
	if p.Types == nil {
		return nil
	}
	return p.Types.Scope().Lookup(name)
}
