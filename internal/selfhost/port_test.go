package selfhost_test

import (
	"testing"

	"goal/internal/project"
	"goal/internal/selfhost"
)

// TestPortedTokenPackage validates US-005: the token package reimplemented as
// goal source under internal/compiler/token transpiles to compiling Go (the US-002 smoke
// gate) AND passes the existing internal/token tests against the transpiled
// output (behavioral equivalence — including the US-001 iota const-block ranges,
// which the round-trip tests in token_test.go pin). The test's working directory
// is internal/selfhost, so internal/compiler/token is at ../../internal/compiler/token and the
// existing token tests are at ../token.
func TestPortedTokenPackage(t *testing.T) {
	pkgs, err := project.Discover("../../internal/compiler/token")
	if err != nil {
		t.Fatalf("discovering internal/compiler/token: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("internal/compiler/token: got %d packages, want exactly 1", len(pkgs))
	}
	pkg := pkgs[0]
	if pkg.Name != "token" {
		t.Fatalf("internal/compiler/token: package name = %q, want \"token\"", pkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go compiles.
	if err := selfhost.BuildTranspiled(map[string]*project.Package{"internal/compiler/token": pkg}); err != nil {
		t.Fatalf("ported token failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing token tests pass against the transpiled package.
	// token is the leaf of the DAG, so it has no in-module dependencies (nil deps).
	if err := selfhost.BuildAndTest("internal/compiler/token", pkg, []string{"../token/token_test.go"}, nil); err != nil {
		t.Fatalf("existing token tests failed against the transpiled package: %v", err)
	}
}

// TestPortedLexerPackage validates US-006: the lexer package reimplemented as
// goal source under internal/compiler/lexer transpiles to compiling Go (the US-002 smoke
// gate, with unicode/unicode/utf8 passing through as foreign imports and the
// in-module token import resolving against the ported token package) AND passes
// the existing internal/lexer tests against the transpiled output (behavioral
// equivalence). The test's working directory is internal/selfhost, so the goal
// sources are at ../../internal/compiler/{token,lexer} and the existing lexer tests are
// at ../lexer.
func TestPortedLexerPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../internal/compiler/token")
	if err != nil {
		t.Fatalf("discovering internal/compiler/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("internal/compiler/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("internal/compiler/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	lexerPkgs, err := project.Discover("../../internal/compiler/lexer")
	if err != nil {
		t.Fatalf("discovering internal/compiler/lexer: %v", err)
	}
	if len(lexerPkgs) != 1 {
		t.Fatalf("internal/compiler/lexer: got %d packages, want exactly 1", len(lexerPkgs))
	}
	lexerPkg := lexerPkgs[0]
	if lexerPkg.Name != "lexer" {
		t.Fatalf("internal/compiler/lexer: package name = %q, want \"lexer\"", lexerPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries both lexer and its token dependency so the
	// in-module import resolves.
	layout := map[string]*project.Package{
		"internal/compiler/token": tokenPkg,
		"internal/compiler/lexer": lexerPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported lexer failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing lexer tests pass against the transpiled package,
	// with the ported token package transpiled in as its in-module dependency.
	deps := map[string]*project.Package{"internal/compiler/token": tokenPkg}
	if err := selfhost.BuildAndTest("internal/compiler/lexer", lexerPkg, []string{"../lexer/lexer_test.go"}, deps); err != nil {
		t.Fatalf("existing lexer tests failed against the transpiled package: %v", err)
	}
}

// TestPortedAstPackage validates US-007: the ast package reimplemented as goal
// source under internal/compiler/ast transpiles to compiling Go (the US-002 smoke gate,
// with the in-module token import resolving against the ported token package)
// AND passes the existing internal/ast tests against the transpiled output
// (behavioral equivalence — node definitions and Walk). The reflection-driven
// dump.go debug renderer is intentionally excluded from the self-hosted build
// (off the compile path; unreferenced by the tests). The test's working
// directory is internal/selfhost, so the goal sources are at
// ../../internal/compiler/{token,ast} and the existing ast tests are at ../ast.
func TestPortedAstPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../internal/compiler/token")
	if err != nil {
		t.Fatalf("discovering internal/compiler/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("internal/compiler/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("internal/compiler/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	astPkgs, err := project.Discover("../../internal/compiler/ast")
	if err != nil {
		t.Fatalf("discovering internal/compiler/ast: %v", err)
	}
	if len(astPkgs) != 1 {
		t.Fatalf("internal/compiler/ast: got %d packages, want exactly 1", len(astPkgs))
	}
	astPkg := astPkgs[0]
	if astPkg.Name != "ast" {
		t.Fatalf("internal/compiler/ast: package name = %q, want \"ast\"", astPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries both ast and its token dependency so the
	// in-module import resolves.
	layout := map[string]*project.Package{
		"internal/compiler/token": tokenPkg,
		"internal/compiler/ast":   astPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported ast failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing ast tests pass against the transpiled package,
	// with the ported token package transpiled in as its in-module dependency.
	deps := map[string]*project.Package{"internal/compiler/token": tokenPkg}
	if err := selfhost.BuildAndTest("internal/compiler/ast", astPkg, []string{"../ast/ast_test.go"}, deps); err != nil {
		t.Fatalf("existing ast tests failed against the transpiled package: %v", err)
	}
}

// TestPortedParserPackage validates US-008: the parser package reimplemented as
// goal source under internal/compiler/parser transpiles to compiling Go (the US-002 smoke
// gate, with the in-module token, lexer, and ast imports resolving against the
// ported packages) AND passes the existing internal/parser tests against the
// transpiled output (behavioral equivalence — recursive-descent parsing,
// precedence climbing, and the goal-specific constructs).
//
// The behavioral gate runs the self-contained parser_test.go suite. The other
// parser suites (goal_construct/decl/match/stmt_test.go) read repo-relative
// ../../features example fixtures via shared helpers, and snapshot_test.go depends
// on ast.Sexpr (intentionally dropped from the ported ast in US-007) plus
// repo-relative fixtures — none are self-contained in the harness's throwaway temp
// module, so they are excluded from this gate. The test's working directory is
// internal/selfhost, so the goal sources are at ../../internal/compiler/{token,lexer,ast,parser}
// and the existing parser tests are at ../parser.
func TestPortedParserPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../internal/compiler/token")
	if err != nil {
		t.Fatalf("discovering internal/compiler/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("internal/compiler/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("internal/compiler/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	lexerPkgs, err := project.Discover("../../internal/compiler/lexer")
	if err != nil {
		t.Fatalf("discovering internal/compiler/lexer: %v", err)
	}
	if len(lexerPkgs) != 1 {
		t.Fatalf("internal/compiler/lexer: got %d packages, want exactly 1", len(lexerPkgs))
	}
	lexerPkg := lexerPkgs[0]
	if lexerPkg.Name != "lexer" {
		t.Fatalf("internal/compiler/lexer: package name = %q, want \"lexer\"", lexerPkg.Name)
	}

	astPkgs, err := project.Discover("../../internal/compiler/ast")
	if err != nil {
		t.Fatalf("discovering internal/compiler/ast: %v", err)
	}
	if len(astPkgs) != 1 {
		t.Fatalf("internal/compiler/ast: got %d packages, want exactly 1", len(astPkgs))
	}
	astPkg := astPkgs[0]
	if astPkg.Name != "ast" {
		t.Fatalf("internal/compiler/ast: package name = %q, want \"ast\"", astPkg.Name)
	}

	parserPkgs, err := project.Discover("../../internal/compiler/parser")
	if err != nil {
		t.Fatalf("discovering internal/compiler/parser: %v", err)
	}
	if len(parserPkgs) != 1 {
		t.Fatalf("internal/compiler/parser: got %d packages, want exactly 1", len(parserPkgs))
	}
	parserPkg := parserPkgs[0]
	if parserPkg.Name != "parser" {
		t.Fatalf("internal/compiler/parser: package name = %q, want \"parser\"", parserPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries parser plus its token, lexer, and ast
	// dependencies so the in-module imports resolve.
	layout := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported parser failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing parser tests pass against the transpiled package,
	// with the ported token, lexer, and ast packages transpiled in as its
	// in-module dependencies.
	deps := map[string]*project.Package{
		"internal/compiler/token": tokenPkg,
		"internal/compiler/lexer": lexerPkg,
		"internal/compiler/ast":   astPkg,
	}
	if err := selfhost.BuildAndTest("internal/compiler/parser", parserPkg, []string{"../parser/parser_test.go"}, deps); err != nil {
		t.Fatalf("existing parser tests failed against the transpiled package: %v", err)
	}
}

// TestPortedSemaPackage validates US-009: the sema package reimplemented as goal
// source under internal/compiler/sema transpiles to compiling Go (the US-002 smoke gate,
// with the in-module token, ast, and parser imports resolving against the ported
// packages and the foreign go/parser, go/format, go/types imports passing
// through) AND passes the existing internal/sema tests against the transpiled
// output (behavioral equivalence — resolution and checking).
//
// sema imports token, ast, parser directly; the transpiled parser in turn
// imports goal/internal/lexer, so the layout carries lexer too even though sema
// does not import it directly. The behavioral gate runs the self-contained sema
// suites; foreign_test.go and package_test.go are excluded because they read the
// repo-relative internal/sema/testdata/extpkg fixture, which is absent from the
// harness's throwaway temp module (same spirit as US-007/US-008 excluding
// fixture-dependent suites). The test's working directory is internal/selfhost,
// so the goal sources are at ../../internal/compiler/{token,lexer,ast,parser,sema} and the
// existing sema tests are at ../sema.
func TestPortedSemaPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../internal/compiler/token")
	if err != nil {
		t.Fatalf("discovering internal/compiler/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("internal/compiler/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("internal/compiler/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	lexerPkgs, err := project.Discover("../../internal/compiler/lexer")
	if err != nil {
		t.Fatalf("discovering internal/compiler/lexer: %v", err)
	}
	if len(lexerPkgs) != 1 {
		t.Fatalf("internal/compiler/lexer: got %d packages, want exactly 1", len(lexerPkgs))
	}
	lexerPkg := lexerPkgs[0]
	if lexerPkg.Name != "lexer" {
		t.Fatalf("internal/compiler/lexer: package name = %q, want \"lexer\"", lexerPkg.Name)
	}

	astPkgs, err := project.Discover("../../internal/compiler/ast")
	if err != nil {
		t.Fatalf("discovering internal/compiler/ast: %v", err)
	}
	if len(astPkgs) != 1 {
		t.Fatalf("internal/compiler/ast: got %d packages, want exactly 1", len(astPkgs))
	}
	astPkg := astPkgs[0]
	if astPkg.Name != "ast" {
		t.Fatalf("internal/compiler/ast: package name = %q, want \"ast\"", astPkg.Name)
	}

	parserPkgs, err := project.Discover("../../internal/compiler/parser")
	if err != nil {
		t.Fatalf("discovering internal/compiler/parser: %v", err)
	}
	if len(parserPkgs) != 1 {
		t.Fatalf("internal/compiler/parser: got %d packages, want exactly 1", len(parserPkgs))
	}
	parserPkg := parserPkgs[0]
	if parserPkg.Name != "parser" {
		t.Fatalf("internal/compiler/parser: package name = %q, want \"parser\"", parserPkg.Name)
	}

	semaPkgs, err := project.Discover("../../internal/compiler/sema")
	if err != nil {
		t.Fatalf("discovering internal/compiler/sema: %v", err)
	}
	if len(semaPkgs) != 1 {
		t.Fatalf("internal/compiler/sema: got %d packages, want exactly 1", len(semaPkgs))
	}
	semaPkg := semaPkgs[0]
	if semaPkg.Name != "sema" {
		t.Fatalf("internal/compiler/sema: package name = %q, want \"sema\"", semaPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries sema plus its token, ast, and parser
	// dependencies (and lexer, pulled in by the transpiled parser) so the
	// in-module imports resolve; the go/* foreign imports pass through.
	layout := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
		"internal/compiler/sema":   semaPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported sema failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing sema tests pass against the transpiled package,
	// with the ported token, lexer, ast, and parser packages transpiled in as its
	// in-module dependencies. The self-contained suites are included;
	// foreign_test.go and package_test.go are excluded (testdata/extpkg fixture).
	deps := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
	}
	testFiles := []string{
		"../sema/sema_test.go",
		"../sema/assert_test.go",
		"../sema/check_test.go",
		"../sema/convert_test.go",
		"../sema/implements_test.go",
		"../sema/mustuse_test.go",
		"../sema/question_test.go",
		"../sema/resolve_test.go",
	}
	if err := selfhost.BuildAndTest("internal/compiler/sema", semaPkg, testFiles, deps); err != nil {
		t.Fatalf("existing sema tests failed against the transpiled package: %v", err)
	}
}

// discoverPorted is a small helper for the project/pipeline ports: it discovers
// the single goal package under ../../internal/compiler/<name> and asserts its package
// name, failing the test on any deviation.
func discoverPorted(t *testing.T, name string) *project.Package {
	t.Helper()
	pkgs, err := project.Discover("../../internal/compiler/" + name)
	if err != nil {
		t.Fatalf("discovering internal/compiler/%s: %v", name, err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("internal/compiler/%s: got %d packages, want exactly 1", name, len(pkgs))
	}
	pkg := pkgs[0]
	if pkg.Name != name {
		t.Fatalf("internal/compiler/%s: package name = %q, want %q", name, pkg.Name, name)
	}
	return pkg
}

// TestPortedProjectPackage validates US-010 (project half): the project package
// reimplemented as goal source under internal/compiler/project transpiles to compiling Go
// (the US-002 smoke gate, with os, io/fs, path/filepath, fmt, sort, strings
// passing through as foreign imports and the in-module parser import resolving
// against the ported parser) AND passes the existing internal/project tests
// against the transpiled output (behavioral equivalence — directory-grouped
// package discovery).
//
// project imports goal/internal/parser directly; the transpiled parser in turn
// imports goal/internal/lexer and goal/internal/ast (which import token), so the
// layout and deps carry token, lexer, ast, parser even though project names only
// parser. The behavioral gate runs project_test.go, which is stdlib-only
// (os/path/filepath/testing) and builds its fixtures in temp dirs, so it is
// self-contained in the harness's throwaway temp module.
func TestPortedProjectPackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	projectPkg := discoverPorted(t, "project")

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries project plus its parser dependency (and
	// parser's transitive lexer/ast/token) so the in-module imports resolve.
	layout := map[string]*project.Package{
		"internal/compiler/token":   tokenPkg,
		"internal/compiler/lexer":   lexerPkg,
		"internal/compiler/ast":     astPkg,
		"internal/compiler/parser":  parserPkg,
		"internal/compiler/project": projectPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported project failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing project tests pass against the transpiled
	// package, with the ported token, lexer, ast, and parser packages transpiled
	// in as its in-module dependencies.
	deps := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
	}
	if err := selfhost.BuildAndTest("internal/compiler/project", projectPkg, []string{"../project/project_test.go"}, deps); err != nil {
		t.Fatalf("existing project tests failed against the transpiled package: %v", err)
	}
}

// TestPortedPipelinePackage validates US-010 (pipeline half): the pipeline
// package reimplemented as goal source under internal/compiler/pipeline transpiles to
// compiling Go (the US-002 smoke gate, with the in-module ast, parser, token
// imports resolving against the ported packages) AND passes the existing
// internal/pipeline tests against the transpiled output (behavioral equivalence
// — the //line source-position map).
//
// sourcemap.goal imports ast, parser, token directly; the transpiled parser in
// turn imports lexer, so the layout and deps carry lexer too. pipeline.goal is
// pure output types with no imports. The behavioral gate runs sourcemap_test.go
// (white-box, strings/testing only — self-contained); pipeline_test.go is
// excluded because it imports goal/internal/backend and goal/internal/corpus and
// reads the repo-relative corpus manifest fixture, which is absent from the
// harness's throwaway temp module (same exclusion spirit as the prior ports'
// fixture-dependent suites).
func TestPortedPipelinePackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	pipelinePkg := discoverPorted(t, "pipeline")

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries pipeline plus its ast, parser, token
	// dependencies (and lexer, pulled in by the transpiled parser).
	layout := map[string]*project.Package{
		"internal/compiler/token":    tokenPkg,
		"internal/compiler/lexer":    lexerPkg,
		"internal/compiler/ast":      astPkg,
		"internal/compiler/parser":   parserPkg,
		"internal/compiler/pipeline": pipelinePkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported pipeline failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing self-contained pipeline tests pass against the
	// transpiled package; pipeline_test.go is excluded (backend/corpus/manifest).
	deps := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
	}
	if err := selfhost.BuildAndTest("internal/compiler/pipeline", pipelinePkg, []string{"../pipeline/sourcemap_test.go"}, deps); err != nil {
		t.Fatalf("existing pipeline tests failed against the transpiled package: %v", err)
	}
}

// TestPortedBackendPackage validates US-001: the backend package (the compiler's
// largest, ~5k LOC across arity/backend/doctest/emit/lower/package) reimplemented
// verbatim as goal source under internal/compiler/backend transpiles to compiling Go (the
// smoke gate, with the in-module token, ast, parser, sema, project, pipeline
// imports resolving against the ported packages and the foreign go/format,
// go/importer, go/token, go/types imports passing through) AND passes the
// existing internal/backend tests against the transpiled output (behavioral
// equivalence — the emitter, lowering, arity resolution, and doctest harness).
//
// backend imports token, ast, parser, sema, project, pipeline directly; the
// transpiled parser/sema/project/pipeline in turn import goal/internal/lexer, so
// the layout carries lexer too. The behavioral gate runs the self-contained
// backend_selfhost_test.go suite (the subset split out of backend_test.go that
// depends on neither the corpus harness nor repo-relative ../../features/testdata
// fixtures, both absent from the harness's throwaway temp module — same exclusion
// spirit as the prior ports' fixture-dependent suites). The test's working
// directory is internal/selfhost, so the goal sources are at
// ../../internal/compiler/{token,lexer,ast,parser,sema,project,pipeline,backend} and the
// existing backend tests are at ../backend.
func TestPortedBackendPackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	semaPkg := discoverPorted(t, "sema")
	projectPkg := discoverPorted(t, "project")
	pipelinePkg := discoverPorted(t, "pipeline")
	backendPkg := discoverPorted(t, "backend")

	// Criterion 2: transpiles via the smoke gate and the generated Go compiles.
	// The layout carries backend plus its full in-module dependency closure so
	// the in-module imports resolve; the go/* foreign imports pass through.
	layout := map[string]*project.Package{
		"internal/compiler/token":    tokenPkg,
		"internal/compiler/lexer":    lexerPkg,
		"internal/compiler/ast":      astPkg,
		"internal/compiler/parser":   parserPkg,
		"internal/compiler/sema":     semaPkg,
		"internal/compiler/project":  projectPkg,
		"internal/compiler/pipeline": pipelinePkg,
		"internal/compiler/backend":  backendPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported backend failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing self-contained backend tests pass against the
	// transpiled package, with the ported dependency closure transpiled in. The
	// behavioral test file (package backend_test) imports goal/internal/backend
	// and goal/internal/project, both present in the temp module.
	deps := map[string]*project.Package{
		"internal/compiler/token":    tokenPkg,
		"internal/compiler/lexer":    lexerPkg,
		"internal/compiler/ast":      astPkg,
		"internal/compiler/parser":   parserPkg,
		"internal/compiler/sema":     semaPkg,
		"internal/compiler/project":  projectPkg,
		"internal/compiler/pipeline": pipelinePkg,
	}
	if err := selfhost.BuildAndTest("internal/compiler/backend", backendPkg, []string{"../backend/backend_selfhost_test.go"}, deps); err != nil {
		t.Fatalf("existing backend tests failed against the transpiled package: %v", err)
	}
}

// TestPortedTypecheckPackage validates US-002: the typecheck package (the depth
// checker — mustuse/nozero/implements analyses) reimplemented verbatim as goal
// source under internal/compiler/typecheck transpiles to compiling Go (the smoke gate,
// with the in-module token, ast, parser, sema, project, backend imports
// resolving against the ported packages and the foreign go/ast, go/importer,
// go/parser, go/token, go/types imports passing through) AND passes the existing
// internal/typecheck tests against the transpiled output (behavioral
// equivalence — the depth checks).
//
// typecheck imports token, ast, parser, sema, project, backend directly; the
// transpiled backend (and parser/sema/project) in turn import
// goal/internal/lexer and goal/internal/pipeline, so the layout carries lexer
// and pipeline too even though typecheck names neither. All five typecheck test
// files are white-box (package typecheck), stdlib + project/sema only, and read
// no repo-relative fixtures (the .goal suffix checks run on synthetic
// filenames), so the full suite is self-contained in the harness's throwaway
// temp module. The test's working directory is internal/selfhost, so the goal
// sources are at ../../internal/compiler/{token,lexer,ast,parser,sema,project,pipeline,
// backend,typecheck} and the existing typecheck tests are at ../typecheck.
func TestPortedTypecheckPackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	semaPkg := discoverPorted(t, "sema")
	projectPkg := discoverPorted(t, "project")
	pipelinePkg := discoverPorted(t, "pipeline")
	backendPkg := discoverPorted(t, "backend")
	typecheckPkg := discoverPorted(t, "typecheck")

	// Criterion 2: transpiles via the smoke gate and the generated Go compiles.
	// The layout carries typecheck plus its full in-module dependency closure so
	// the in-module imports resolve; the go/* foreign imports pass through.
	layout := map[string]*project.Package{
		"internal/compiler/token":     tokenPkg,
		"internal/compiler/lexer":     lexerPkg,
		"internal/compiler/ast":       astPkg,
		"internal/compiler/parser":    parserPkg,
		"internal/compiler/sema":      semaPkg,
		"internal/compiler/project":   projectPkg,
		"internal/compiler/pipeline":  pipelinePkg,
		"internal/compiler/backend":   backendPkg,
		"internal/compiler/typecheck": typecheckPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported typecheck failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing typecheck depth tests pass against the transpiled
	// package, with the ported dependency closure transpiled in.
	deps := map[string]*project.Package{
		"internal/compiler/token":    tokenPkg,
		"internal/compiler/lexer":    lexerPkg,
		"internal/compiler/ast":      astPkg,
		"internal/compiler/parser":   parserPkg,
		"internal/compiler/sema":     semaPkg,
		"internal/compiler/project":  projectPkg,
		"internal/compiler/pipeline": pipelinePkg,
		"internal/compiler/backend":  backendPkg,
	}
	testFiles := []string{
		"../typecheck/checker_test.go",
		"../typecheck/implements_test.go",
		"../typecheck/mustuse_test.go",
		"../typecheck/nozero_test.go",
		"../typecheck/typecheck_test.go",
	}
	if err := selfhost.BuildAndTest("internal/compiler/typecheck", typecheckPkg, testFiles, deps); err != nil {
		t.Fatalf("existing typecheck tests failed against the transpiled package: %v", err)
	}
}

// TestPortedGoalfmtPackage validates US-006: the goalfmt formatter reimplemented as
// goal source under internal/compiler/goalfmt transpiles to compiling Go (the smoke
// gate, with strings passing through as a foreign import and the in-module lexer,
// parser, token imports resolving against the ported packages) AND passes the
// existing internal/goalfmt behavior tests against the transpiled output (behavioral
// equivalence — byte-identical formatting and idempotence).
//
// goalfmt imports lexer, parser, token directly; the transpiled parser in turn
// imports goal/internal/compiler/ast (and lexer/token), so the layout and deps carry
// ast too even though goalfmt names only lexer/parser/token. The behavioral gate runs
// the self-contained format_selfhost_test.go suite (TestPreservesComments,
// TestReindentsAndIsStable — which pins byte-identical output, AC-2 — and
// TestRejectsUnparseable, all asserting idempotence, AC-3). format_test.go's
// TestIdempotentOverCorpus is excluded because it imports goal/internal/corpus and
// reads the repo-relative corpus manifest, absent from the harness's throwaway temp
// module (same exclusion spirit as the prior ports' fixture-dependent suites). The
// test's working directory is internal/selfhost, so the goal sources are at
// ../../internal/compiler/{token,lexer,ast,parser,goalfmt} and the existing goalfmt
// tests are at ../goalfmt.
func TestPortedGoalfmtPackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	goalfmtPkg := discoverPorted(t, "goalfmt")

	// Criterion 2: transpiles via the smoke gate and the generated Go compiles.
	// The layout carries goalfmt plus its lexer, parser, token dependencies (and
	// ast, pulled in by the transpiled parser); the strings foreign import passes
	// through.
	layout := map[string]*project.Package{
		"internal/compiler/token":   tokenPkg,
		"internal/compiler/lexer":   lexerPkg,
		"internal/compiler/ast":     astPkg,
		"internal/compiler/parser":  parserPkg,
		"internal/compiler/goalfmt": goalfmtPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported goalfmt failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing self-contained goalfmt tests pass against the
	// transpiled package, with the ported dependency closure transpiled in. The
	// behavioral test file (package goalfmt, strings + testing only) pins
	// byte-identical formatting output and idempotence.
	deps := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
	}
	if err := selfhost.BuildAndTest("internal/compiler/goalfmt", goalfmtPkg, []string{"../goalfmt/format_selfhost_test.go"}, deps); err != nil {
		t.Fatalf("existing goalfmt tests failed against the transpiled package: %v", err)
	}
}

// TestPortedTexteditPackage validates US-007: the edit-primitives package
// reimplemented as goal source under internal/compiler/textedit transpiles to
// compiling Go (the smoke gate — textedit is a LEAF that imports only stdlib
// sort/strings/unicode, so the layout is the single package and there are no
// in-module deps to transpile alongside) AND produces the same resulting text as
// the legacy package for the same edits. The behavioral gate runs the
// self-contained textedit_selfhost_test.go suite, whose pinned byte-identical
// outputs (Splice spans, IsLineStart/NextNewline scanning, LeadIdent/IsIdent,
// SplitAssign, IsStmtKeyword, BaseType, ZeroLit) are the AC-2 parity oracle —
// the identical file runs against the legacy package under `task check`. The
// test's working directory is internal/selfhost, so the goal source is at
// ../../internal/compiler/textedit and the existing tests are at ../textedit.
func TestPortedTexteditPackage(t *testing.T) {
	texteditPkg := discoverPorted(t, "textedit")

	// Criterion 1: transpiles via the smoke gate and the generated Go compiles.
	// A leaf package, so the layout is textedit alone; sort, strings and unicode
	// pass through as foreign stdlib imports.
	layout := map[string]*project.Package{
		"internal/compiler/textedit": texteditPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported textedit failed the transpile-and-build gate: %v", err)
	}

	// Criterion 2: the self-contained textedit tests pass against the transpiled
	// package. Leaf package => no dependency closure to transpile (nil deps). The
	// behavioral test file pins byte-identical edit results, proving same-input
	// same-output parity with the legacy package.
	if err := selfhost.BuildAndTest("internal/compiler/textedit", texteditPkg, []string{"../textedit/textedit_selfhost_test.go"}, nil); err != nil {
		t.Fatalf("existing textedit tests failed against the transpiled package: %v", err)
	}
}

// TestPortedCapPackage validates US-008: the capability/authority model
// reimplemented as goal source under internal/compiler/cap transpiles to
// compiling Go (the smoke gate — cap is a LEAF that imports nothing at all, so
// the layout is the single package and there are no in-module deps to transpile
// alongside) AND makes the same allow/deny decisions as the legacy package for
// the same capability checks. The behavioral gate runs the legacy package's own
// cap_test.go, which is self-contained (stdlib testing only) and exercises
// GrantAll/DenyAll/Grant membership plus the String enumeration over every
// defined capability — the identical file runs against the legacy package under
// `task check`, so it is the AC-2 parity oracle. The test's working directory
// is internal/selfhost, so the goal source is at ../../internal/compiler/cap and
// the existing tests are at ../cap.
func TestPortedCapPackage(t *testing.T) {
	capPkg := discoverPorted(t, "cap")

	// Criterion 1: transpiles via the smoke gate and the generated Go compiles.
	// A leaf package with no imports at all, so the layout is cap alone and there
	// is no dependency closure to transpile.
	layout := map[string]*project.Package{
		"internal/compiler/cap": capPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported cap failed the transpile-and-build gate: %v", err)
	}

	// Criterion 2: the self-contained cap tests pass against the transpiled
	// package. Leaf package => no dependency closure to transpile (nil deps). The
	// behavioral test file pins grant/deny/membership/String results, proving
	// same-input same-decision parity with the legacy package.
	if err := selfhost.BuildAndTest("internal/compiler/cap", capPkg, []string{"../cap/cap_test.go"}, nil); err != nil {
		t.Fatalf("existing cap tests failed against the transpiled package: %v", err)
	}
}

// TestPortedGuidePackage validates US-009's port SHAPE: the AI-bootstrap guide
// surface reimplemented as goal source under internal/compiler/guide discovers as
// exactly one package named "guide".
//
// Unlike the other ports, guide does NOT use the BuildTranspiled/BuildAndTest
// throwaway-module gate. guide imports the root `goal` package (its embedded docs
// FS) and goal/internal/byexample, both Go-only dev infra (US-001) that are absent
// from the harness's isolated `module goal` temp dir, so the transpiled guide
// cannot build there. Instead the two US-009 acceptance criteria are met in the
// REAL module, where every dependency exists:
//   - AC-1 (generates colocated .go that builds): `task generate` emits
//     internal/compiler/guide/{guide,catalog}.go and `go build ./internal/compiler/
//     guide/...` compiles them; the `task verify-generated` drift gate (in `task
//     check`) holds them to the .goal source.
//   - AC-2 (output matches the legacy guide on a fixture set): the in-repo parity
//     oracle internal/guide/guide_parity_test.go renders every section and the full
//     document from BOTH the legacy goal/internal/guide and the goal-sourced
//     goal/internal/compiler/guide and asserts byte-identical output under `task
//     check`.
//
// The test's working directory is internal/selfhost, so the goal source is at
// ../../internal/compiler/guide.
func TestPortedGuidePackage(t *testing.T) {
	// discoverPorted asserts exactly one package discovered and the package name.
	// The feedback demo program the legacy package embeds as feedback_sample.goal
	// is inlined as a const in the port (the goal backend drops //go:embed), so the
	// guide directory holds only its own `package guide` source — no stray second
	// package clause to trip project.Discover.
	_ = discoverPorted(t, "guide")
}

// TestPortedFixPackage validates US-010: the autofixer (internal/fix —
// callsite/fix/match/propagate/resultsig) reimplemented as goal source under
// internal/compiler/fix transpiles to compiling Go (the smoke gate, with the
// in-module ast, parser, sema, textedit, token imports resolving against the
// ported packages and the foreign `strings` import passing through) AND produces
// byte-identical output to the legacy Go fix across the existing autofix golden
// fixtures (fix_test.go, package fix, strings + testing only, with every input
// and expected rewrite/report pinned inline — it drives fix.File(src) and so
// computes its sema info internally; the identical file runs against the legacy
// package under `task check`, making it the AC-2 parity oracle).
//
// fix imports ast, parser, sema, textedit, token directly; the transpiled
// parser/sema in turn import goal/internal/compiler/lexer, so the layout and
// deps carry lexer too. The test's working directory is internal/selfhost, so
// the goal sources are at ../../internal/compiler/{token,lexer,ast,parser,sema,
// textedit,fix} and the existing fix tests are at ../fix.
func TestPortedFixPackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	semaPkg := discoverPorted(t, "sema")
	texteditPkg := discoverPorted(t, "textedit")
	fixPkg := discoverPorted(t, "fix")

	// Criterion 1: transpiles via the smoke gate and the generated Go compiles.
	// The layout carries fix plus its full in-module dependency closure so the
	// in-module imports resolve; the strings foreign import passes through.
	layout := map[string]*project.Package{
		"internal/compiler/token":    tokenPkg,
		"internal/compiler/lexer":    lexerPkg,
		"internal/compiler/ast":      astPkg,
		"internal/compiler/parser":   parserPkg,
		"internal/compiler/sema":     semaPkg,
		"internal/compiler/textedit": texteditPkg,
		"internal/compiler/fix":      fixPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported fix failed the transpile-and-build gate: %v", err)
	}

	// Criterion 2: the existing fix tests pass against the transpiled package,
	// with the ported dependency closure transpiled in. fix_test.go (package fix)
	// pins byte-identical rewritten source and reports, proving same-input
	// same-output parity with the legacy autofixer.
	deps := map[string]*project.Package{
		"internal/compiler/token":    tokenPkg,
		"internal/compiler/lexer":    lexerPkg,
		"internal/compiler/ast":      astPkg,
		"internal/compiler/parser":   parserPkg,
		"internal/compiler/sema":     semaPkg,
		"internal/compiler/textedit": texteditPkg,
	}
	if err := selfhost.BuildAndTest("internal/compiler/fix", fixPkg, []string{"../fix/fix_test.go"}, deps); err != nil {
		t.Fatalf("existing fix tests failed against the transpiled package: %v", err)
	}
}

// TestPortedInterpPackage validates US-011 and US-012: the interpreter's runtime
// foundation — the value model (value.goal), lexical environment (env.goal),
// host-function bridge (host.goal), assert evaluation (assert.goal) — AND the
// expression EVALUATOR (eval.goal, US-012), reimplemented as goal source under
// internal/compiler/interp, transpiles to compiling Go (the smoke gate) AND
// passes the existing internal/interp value/env tests plus the US-012
// eval-subset tests against the transpiled output (behavioral equivalence —
// Value construction/equality/Kind, Env scope/lookup/assign, and the
// arithmetic/comparison/logical/unary/error evaluation matrix).
//
// eval.goal (US-012) is the real evaluator but calls driver symbols ported in
// US-013 (interp.go -> interp.goal, derive.go): callFunc, callMethod, sigFor,
// curSig, evalDerive, and match dispatch. So the goal-sourced package builds at
// the US-012 checkpoint, internal/compiler/interp/interp.goal is a transitional
// skeleton supplying the full Interp struct, panicSignal/returnSignal/
// CapabilityError/emitStdout, plus loud-refusal placeholders for those driver
// symbols (US-013 deletes them). The behavioral gate runs the AC-2 oracle —
// value_test.go and env_test.go (US-011) and eval_subset_test.go (US-012,
// driver-free: it builds a bare *Interp and drives parsed expressions straight
// through evalExpr) — all white-box and self-contained; the identical files run
// against the legacy package under `task check`. host_test.go, assert_test.go,
// and the legacy eval_test.go are excluded because they drive whole programs
// through the US-013 driver (New/findMain/Run), not ported until US-013.
//
// Because eval.goal imports sema, the interp package now pulls sema (-> ast,
// parser, token) and parser (-> lexer) in addition to the US-011 ast/token/cap;
// the eval-subset test itself imports parser/ast. So the layout/deps carry
// token, lexer, ast, parser, sema, cap. The test's working directory is
// internal/selfhost, so the goal sources are at ../../internal/compiler/<pkg>
// and the existing interp tests are at ../interp.
func TestPortedInterpPackage(t *testing.T) {
	tokenPkg := discoverPorted(t, "token")
	lexerPkg := discoverPorted(t, "lexer")
	astPkg := discoverPorted(t, "ast")
	parserPkg := discoverPorted(t, "parser")
	semaPkg := discoverPorted(t, "sema")
	capPkg := discoverPorted(t, "cap")
	interpPkg := discoverPorted(t, "interp")

	// Criterion 1: transpiles via the smoke gate and the generated Go compiles.
	// The layout carries interp plus its ast, token, cap, sema (-> parser ->
	// lexer) dependencies so the in-module imports resolve; the stdlib foreign
	// imports pass through.
	layout := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
		"internal/compiler/sema":   semaPkg,
		"internal/compiler/cap":    capPkg,
		"internal/compiler/interp": interpPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported interp failed the transpile-and-build gate: %v", err)
	}

	// Criterion 2: the existing interp value/env tests and the US-012 eval-subset
	// tests pass against the transpiled package, with the ported token, lexer,
	// ast, parser, sema, cap packages transpiled in as in-module dependencies.
	// value_test.go pins per-kind construction and equality; env_test.go pins
	// scope/lookup/assign and NotFoundError; eval_subset_test.go pins the
	// evaluator's arithmetic/comparison/logical/unary/short-circuit/error matrix —
	// the same-input same-output parity oracle with the legacy package.
	deps := map[string]*project.Package{
		"internal/compiler/token":  tokenPkg,
		"internal/compiler/lexer":  lexerPkg,
		"internal/compiler/ast":    astPkg,
		"internal/compiler/parser": parserPkg,
		"internal/compiler/sema":   semaPkg,
		"internal/compiler/cap":    capPkg,
	}
	testFiles := []string{
		"../interp/value_test.go",
		"../interp/env_test.go",
		"../interp/eval_subset_test.go",
	}
	if err := selfhost.BuildAndTest("internal/compiler/interp", interpPkg, testFiles, deps); err != nil {
		t.Fatalf("existing interp value/env and US-012 eval-subset tests failed against the transpiled package: %v", err)
	}
}
