package backend

// package.go is the AST engine's package-mode driver: the counterpart of the
// single-file Transpile for a whole goal package. It is the sole package-mode
// front-end (the splice engine it superseded was removed in US-043):
//
//   - Cross-file sema: every file is resolved and the facts merged into ONE
//     *sema.Info (sema.ResolvePackage), so a file lowers a `match`/derive/`?` over
//     an enum, struct, or signature declared in a sibling file. This mirrors
//     analyze.BuildPackage / analyze.Merge.
//   - A single shared prelude: the closed-E Result sum encoding is emitted once for
//     the package (goal_prelude.go), not per file, by suppressing the inline
//     prelude in each file's emit (emitFileWith) and appending the shared one.
//   - Foreign-import resolution: a `derive func` / `from func` whose source or
//     target is an out-of-package Go type needs that type's field set. The foreign
//     loading (read the imported package's exported structs) is reused from
//     internal/analyze.EnrichForeign — the one IO seam — and folded into the merged
//     sema.Info so the AST derive lowering resolves the foreign fields by name.

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/ast"
	"goal/internal/parser"
	"goal/internal/pipeline"
	"goal/internal/project"
	"goal/internal/sema"
)

// TranspilePackage lowers every file of a goal package through the AST engine
// against one set of merged, name-keyed facts, and emits the closed-E Result
// prelude exactly once for the package. It satisfies corpus.PackageTranspiler, so
// the package-mode behavioral tier judges the AST engine by the same yardstick as
// the splice engine. It does no disk I/O for output (the Go is returned in memory);
// the only IO is foreign-import resolution, which reads the imported Go packages.
func TranspilePackage(pkg *project.Package) (pipeline.PackageOutput, error) {
	files := make([]*ast.File, len(pkg.Files))
	srcs := make([]string, len(pkg.Files))
	for i, f := range pkg.Files {
		parsed, err := parser.ParseFile(f.Src)
		if err != nil {
			return pipeline.PackageOutput{}, fmt.Errorf("%s: parse: %w", f.Name, err)
		}
		files[i] = parsed
		srcs[i] = f.Src
	}

	info := sema.ResolvePackage(files)
	enrichForeign(info, srcs, pkg.Dir)

	var out pipeline.PackageOutput
	for i, f := range pkg.Files {
		goSrc, err := emitFileWith(files[i], info, true)
		if err != nil {
			return pipeline.PackageOutput{}, fmt.Errorf("%s: %w", f.Name, err)
		}
		formatted, err := GoFormatter{}.Format([]byte(goSrc))
		if err != nil {
			return pipeline.PackageOutput{}, fmt.Errorf("%s: generated Go did not parse: %w\n--- generated ---\n%s", f.Name, err, goSrc)
		}
		// Anchor generated decls back to the .goal file (by name) so toolchain build
		// errors land on source positions, matching the splice engine. The shared
		// prelude below carries no directives (errors there are compiler bugs).
		gen := goName(f.Name)
		mapped := pipeline.AddLineDirectives(f.Src, string(formatted), f.Name, gen)
		out.Files = append(out.Files, pipeline.GoFile{Name: gen, Go: mapped})

		testSrc, err := emitDoctests(files[i], info)
		if err != nil {
			return pipeline.PackageOutput{}, fmt.Errorf("%s: doctests: %w", f.Name, err)
		}
		if testSrc != "" {
			ft, err := GoFormatter{}.Format([]byte(testSrc))
			if err != nil {
				return pipeline.PackageOutput{}, fmt.Errorf("%s: generated test did not parse: %w\n--- generated ---\n%s", f.Name, err, testSrc)
			}
			out.Tests = append(out.Tests, pipeline.GoFile{Name: testName(f.Name), Go: string(ft)})
		}
	}

	if needsResultPrelude(info) {
		preludeGo, err := GoFormatter{}.Format([]byte("package " + pkg.Name + "\n\n" + resultPrelude + "\n"))
		if err != nil {
			return pipeline.PackageOutput{}, fmt.Errorf("prelude: %w", err)
		}
		out.Files = append(out.Files, pipeline.GoFile{Name: "goal_prelude.go", Go: string(preludeGo)})
	}
	return out, nil
}

// enrichForeign folds the struct field sets of imported Go packages that a
// `derive func` / `from func` references by qualifier into the merged sema.Info,
// so the AST derive lowering can resolve an out-of-package source/target type. The
// foreign loading (resolve import path -> directory -> parse exported structs) is
// reused from internal/analyze, the one explicitly-IO seam; resolution failures are
// non-fatal (an unresolved import simply leaves derive to defer as before). Foreign
// facts never overwrite an in-package fact (the in-file resolution wins).
func enrichForeign(info *sema.Info, srcs []string, dir string) {
	t := analyze.BuildPackage(srcs)
	analyze.EnrichForeign(t, srcs, dir, nil)
	for name, fields := range t.Structs {
		if _, ours := info.Structs[name]; ours {
			continue
		}
		conv := make([]sema.Field, len(fields))
		for i, f := range fields {
			conv[i] = sema.Field{Name: f.Name, Type: f.Type}
		}
		info.Structs[name] = conv
	}
	for key, entry := range t.FromRegistry {
		if _, ours := info.FromRegistry[key]; ours {
			continue
		}
		info.FromRegistry[key] = sema.ConvEntry{Name: entry.Name, Fallible: entry.Fallible}
	}
}

// goName maps a source file name to its generated Go name: foo.goal -> foo.go.
func goName(goalName string) string {
	return strings.TrimSuffix(goalName, project.Ext) + ".go"
}

// testName maps a source file name to its doctest sidecar: foo.goal -> foo_test.go.
func testName(goalName string) string {
	return strings.TrimSuffix(goalName, project.Ext) + "_test.go"
}
