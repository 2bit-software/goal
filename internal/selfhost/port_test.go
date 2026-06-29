package selfhost_test

import (
	"testing"

	"goal/internal/project"
	"goal/internal/selfhost"
)

// TestPortedTokenPackage validates US-005: the token package reimplemented as
// goal source under selfhost/token transpiles to compiling Go (the US-002 smoke
// gate) AND passes the existing internal/token tests against the transpiled
// output (behavioral equivalence — including the US-001 iota const-block ranges,
// which the round-trip tests in token_test.go pin). The test's working directory
// is internal/selfhost, so selfhost/token is at ../../selfhost/token and the
// existing token tests are at ../token.
func TestPortedTokenPackage(t *testing.T) {
	pkgs, err := project.Discover("../../selfhost/token")
	if err != nil {
		t.Fatalf("discovering selfhost/token: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("selfhost/token: got %d packages, want exactly 1", len(pkgs))
	}
	pkg := pkgs[0]
	if pkg.Name != "token" {
		t.Fatalf("selfhost/token: package name = %q, want \"token\"", pkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go compiles.
	if err := selfhost.BuildTranspiled(map[string]*project.Package{"internal/token": pkg}); err != nil {
		t.Fatalf("ported token failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing token tests pass against the transpiled package.
	// token is the leaf of the DAG, so it has no in-module dependencies (nil deps).
	if err := selfhost.BuildAndTest("internal/token", pkg, []string{"../token/token_test.go"}, nil); err != nil {
		t.Fatalf("existing token tests failed against the transpiled package: %v", err)
	}
}

// TestPortedLexerPackage validates US-006: the lexer package reimplemented as
// goal source under selfhost/lexer transpiles to compiling Go (the US-002 smoke
// gate, with unicode/unicode/utf8 passing through as foreign imports and the
// in-module token import resolving against the ported token package) AND passes
// the existing internal/lexer tests against the transpiled output (behavioral
// equivalence). The test's working directory is internal/selfhost, so the goal
// sources are at ../../selfhost/{token,lexer} and the existing lexer tests are
// at ../lexer.
func TestPortedLexerPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../selfhost/token")
	if err != nil {
		t.Fatalf("discovering selfhost/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("selfhost/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("selfhost/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	lexerPkgs, err := project.Discover("../../selfhost/lexer")
	if err != nil {
		t.Fatalf("discovering selfhost/lexer: %v", err)
	}
	if len(lexerPkgs) != 1 {
		t.Fatalf("selfhost/lexer: got %d packages, want exactly 1", len(lexerPkgs))
	}
	lexerPkg := lexerPkgs[0]
	if lexerPkg.Name != "lexer" {
		t.Fatalf("selfhost/lexer: package name = %q, want \"lexer\"", lexerPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries both lexer and its token dependency so the
	// in-module import resolves.
	layout := map[string]*project.Package{
		"internal/token": tokenPkg,
		"internal/lexer": lexerPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported lexer failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing lexer tests pass against the transpiled package,
	// with the ported token package transpiled in as its in-module dependency.
	deps := map[string]*project.Package{"internal/token": tokenPkg}
	if err := selfhost.BuildAndTest("internal/lexer", lexerPkg, []string{"../lexer/lexer_test.go"}, deps); err != nil {
		t.Fatalf("existing lexer tests failed against the transpiled package: %v", err)
	}
}

// TestPortedAstPackage validates US-007: the ast package reimplemented as goal
// source under selfhost/ast transpiles to compiling Go (the US-002 smoke gate,
// with the in-module token import resolving against the ported token package)
// AND passes the existing internal/ast tests against the transpiled output
// (behavioral equivalence — node definitions and Walk). The reflection-driven
// dump.go debug renderer is intentionally excluded from the self-hosted build
// (off the compile path; unreferenced by the tests). The test's working
// directory is internal/selfhost, so the goal sources are at
// ../../selfhost/{token,ast} and the existing ast tests are at ../ast.
func TestPortedAstPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../selfhost/token")
	if err != nil {
		t.Fatalf("discovering selfhost/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("selfhost/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("selfhost/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	astPkgs, err := project.Discover("../../selfhost/ast")
	if err != nil {
		t.Fatalf("discovering selfhost/ast: %v", err)
	}
	if len(astPkgs) != 1 {
		t.Fatalf("selfhost/ast: got %d packages, want exactly 1", len(astPkgs))
	}
	astPkg := astPkgs[0]
	if astPkg.Name != "ast" {
		t.Fatalf("selfhost/ast: package name = %q, want \"ast\"", astPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries both ast and its token dependency so the
	// in-module import resolves.
	layout := map[string]*project.Package{
		"internal/token": tokenPkg,
		"internal/ast":   astPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported ast failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing ast tests pass against the transpiled package,
	// with the ported token package transpiled in as its in-module dependency.
	deps := map[string]*project.Package{"internal/token": tokenPkg}
	if err := selfhost.BuildAndTest("internal/ast", astPkg, []string{"../ast/ast_test.go"}, deps); err != nil {
		t.Fatalf("existing ast tests failed against the transpiled package: %v", err)
	}
}

// TestPortedParserPackage validates US-008: the parser package reimplemented as
// goal source under selfhost/parser transpiles to compiling Go (the US-002 smoke
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
// internal/selfhost, so the goal sources are at ../../selfhost/{token,lexer,ast,parser}
// and the existing parser tests are at ../parser.
func TestPortedParserPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../selfhost/token")
	if err != nil {
		t.Fatalf("discovering selfhost/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("selfhost/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("selfhost/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	lexerPkgs, err := project.Discover("../../selfhost/lexer")
	if err != nil {
		t.Fatalf("discovering selfhost/lexer: %v", err)
	}
	if len(lexerPkgs) != 1 {
		t.Fatalf("selfhost/lexer: got %d packages, want exactly 1", len(lexerPkgs))
	}
	lexerPkg := lexerPkgs[0]
	if lexerPkg.Name != "lexer" {
		t.Fatalf("selfhost/lexer: package name = %q, want \"lexer\"", lexerPkg.Name)
	}

	astPkgs, err := project.Discover("../../selfhost/ast")
	if err != nil {
		t.Fatalf("discovering selfhost/ast: %v", err)
	}
	if len(astPkgs) != 1 {
		t.Fatalf("selfhost/ast: got %d packages, want exactly 1", len(astPkgs))
	}
	astPkg := astPkgs[0]
	if astPkg.Name != "ast" {
		t.Fatalf("selfhost/ast: package name = %q, want \"ast\"", astPkg.Name)
	}

	parserPkgs, err := project.Discover("../../selfhost/parser")
	if err != nil {
		t.Fatalf("discovering selfhost/parser: %v", err)
	}
	if len(parserPkgs) != 1 {
		t.Fatalf("selfhost/parser: got %d packages, want exactly 1", len(parserPkgs))
	}
	parserPkg := parserPkgs[0]
	if parserPkg.Name != "parser" {
		t.Fatalf("selfhost/parser: package name = %q, want \"parser\"", parserPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries parser plus its token, lexer, and ast
	// dependencies so the in-module imports resolve.
	layout := map[string]*project.Package{
		"internal/token":  tokenPkg,
		"internal/lexer":  lexerPkg,
		"internal/ast":    astPkg,
		"internal/parser": parserPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported parser failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing parser tests pass against the transpiled package,
	// with the ported token, lexer, and ast packages transpiled in as its
	// in-module dependencies.
	deps := map[string]*project.Package{
		"internal/token": tokenPkg,
		"internal/lexer": lexerPkg,
		"internal/ast":   astPkg,
	}
	if err := selfhost.BuildAndTest("internal/parser", parserPkg, []string{"../parser/parser_test.go"}, deps); err != nil {
		t.Fatalf("existing parser tests failed against the transpiled package: %v", err)
	}
}

// TestPortedSemaPackage validates US-009: the sema package reimplemented as goal
// source under selfhost/sema transpiles to compiling Go (the US-002 smoke gate,
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
// so the goal sources are at ../../selfhost/{token,lexer,ast,parser,sema} and the
// existing sema tests are at ../sema.
func TestPortedSemaPackage(t *testing.T) {
	tokenPkgs, err := project.Discover("../../selfhost/token")
	if err != nil {
		t.Fatalf("discovering selfhost/token: %v", err)
	}
	if len(tokenPkgs) != 1 {
		t.Fatalf("selfhost/token: got %d packages, want exactly 1", len(tokenPkgs))
	}
	tokenPkg := tokenPkgs[0]
	if tokenPkg.Name != "token" {
		t.Fatalf("selfhost/token: package name = %q, want \"token\"", tokenPkg.Name)
	}

	lexerPkgs, err := project.Discover("../../selfhost/lexer")
	if err != nil {
		t.Fatalf("discovering selfhost/lexer: %v", err)
	}
	if len(lexerPkgs) != 1 {
		t.Fatalf("selfhost/lexer: got %d packages, want exactly 1", len(lexerPkgs))
	}
	lexerPkg := lexerPkgs[0]
	if lexerPkg.Name != "lexer" {
		t.Fatalf("selfhost/lexer: package name = %q, want \"lexer\"", lexerPkg.Name)
	}

	astPkgs, err := project.Discover("../../selfhost/ast")
	if err != nil {
		t.Fatalf("discovering selfhost/ast: %v", err)
	}
	if len(astPkgs) != 1 {
		t.Fatalf("selfhost/ast: got %d packages, want exactly 1", len(astPkgs))
	}
	astPkg := astPkgs[0]
	if astPkg.Name != "ast" {
		t.Fatalf("selfhost/ast: package name = %q, want \"ast\"", astPkg.Name)
	}

	parserPkgs, err := project.Discover("../../selfhost/parser")
	if err != nil {
		t.Fatalf("discovering selfhost/parser: %v", err)
	}
	if len(parserPkgs) != 1 {
		t.Fatalf("selfhost/parser: got %d packages, want exactly 1", len(parserPkgs))
	}
	parserPkg := parserPkgs[0]
	if parserPkg.Name != "parser" {
		t.Fatalf("selfhost/parser: package name = %q, want \"parser\"", parserPkg.Name)
	}

	semaPkgs, err := project.Discover("../../selfhost/sema")
	if err != nil {
		t.Fatalf("discovering selfhost/sema: %v", err)
	}
	if len(semaPkgs) != 1 {
		t.Fatalf("selfhost/sema: got %d packages, want exactly 1", len(semaPkgs))
	}
	semaPkg := semaPkgs[0]
	if semaPkg.Name != "sema" {
		t.Fatalf("selfhost/sema: package name = %q, want \"sema\"", semaPkg.Name)
	}

	// Criterion 2: transpiles via the US-002 smoke gate and the generated Go
	// compiles. The layout carries sema plus its token, ast, and parser
	// dependencies (and lexer, pulled in by the transpiled parser) so the
	// in-module imports resolve; the go/* foreign imports pass through.
	layout := map[string]*project.Package{
		"internal/token":  tokenPkg,
		"internal/lexer":  lexerPkg,
		"internal/ast":    astPkg,
		"internal/parser": parserPkg,
		"internal/sema":   semaPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported sema failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing sema tests pass against the transpiled package,
	// with the ported token, lexer, ast, and parser packages transpiled in as its
	// in-module dependencies. The self-contained suites are included;
	// foreign_test.go and package_test.go are excluded (testdata/extpkg fixture).
	deps := map[string]*project.Package{
		"internal/token":  tokenPkg,
		"internal/lexer":  lexerPkg,
		"internal/ast":    astPkg,
		"internal/parser": parserPkg,
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
	if err := selfhost.BuildAndTest("internal/sema", semaPkg, testFiles, deps); err != nil {
		t.Fatalf("existing sema tests failed against the transpiled package: %v", err)
	}
}

// discoverPorted is a small helper for the project/pipeline ports: it discovers
// the single goal package under ../../selfhost/<name> and asserts its package
// name, failing the test on any deviation.
func discoverPorted(t *testing.T, name string) *project.Package {
	t.Helper()
	pkgs, err := project.Discover("../../selfhost/" + name)
	if err != nil {
		t.Fatalf("discovering selfhost/%s: %v", name, err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("selfhost/%s: got %d packages, want exactly 1", name, len(pkgs))
	}
	pkg := pkgs[0]
	if pkg.Name != name {
		t.Fatalf("selfhost/%s: package name = %q, want %q", name, pkg.Name, name)
	}
	return pkg
}

// TestPortedProjectPackage validates US-010 (project half): the project package
// reimplemented as goal source under selfhost/project transpiles to compiling Go
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
		"internal/token":   tokenPkg,
		"internal/lexer":   lexerPkg,
		"internal/ast":     astPkg,
		"internal/parser":  parserPkg,
		"internal/project": projectPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported project failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing project tests pass against the transpiled
	// package, with the ported token, lexer, ast, and parser packages transpiled
	// in as its in-module dependencies.
	deps := map[string]*project.Package{
		"internal/token":  tokenPkg,
		"internal/lexer":  lexerPkg,
		"internal/ast":    astPkg,
		"internal/parser": parserPkg,
	}
	if err := selfhost.BuildAndTest("internal/project", projectPkg, []string{"../project/project_test.go"}, deps); err != nil {
		t.Fatalf("existing project tests failed against the transpiled package: %v", err)
	}
}

// TestPortedPipelinePackage validates US-010 (pipeline half): the pipeline
// package reimplemented as goal source under selfhost/pipeline transpiles to
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
		"internal/token":    tokenPkg,
		"internal/lexer":    lexerPkg,
		"internal/ast":      astPkg,
		"internal/parser":   parserPkg,
		"internal/pipeline": pipelinePkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported pipeline failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing self-contained pipeline tests pass against the
	// transpiled package; pipeline_test.go is excluded (backend/corpus/manifest).
	deps := map[string]*project.Package{
		"internal/token":  tokenPkg,
		"internal/lexer":  lexerPkg,
		"internal/ast":    astPkg,
		"internal/parser": parserPkg,
	}
	if err := selfhost.BuildAndTest("internal/pipeline", pipelinePkg, []string{"../pipeline/sourcemap_test.go"}, deps); err != nil {
		t.Fatalf("existing pipeline tests failed against the transpiled package: %v", err)
	}
}

// TestPortedBackendPackage validates US-001: the backend package (the compiler's
// largest, ~5k LOC across arity/backend/doctest/emit/lower/package) reimplemented
// verbatim as goal source under selfhost/backend transpiles to compiling Go (the
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
// ../../selfhost/{token,lexer,ast,parser,sema,project,pipeline,backend} and the
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
		"internal/token":    tokenPkg,
		"internal/lexer":    lexerPkg,
		"internal/ast":      astPkg,
		"internal/parser":   parserPkg,
		"internal/sema":     semaPkg,
		"internal/project":  projectPkg,
		"internal/pipeline": pipelinePkg,
		"internal/backend":  backendPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported backend failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing self-contained backend tests pass against the
	// transpiled package, with the ported dependency closure transpiled in. The
	// behavioral test file (package backend_test) imports goal/internal/backend
	// and goal/internal/project, both present in the temp module.
	deps := map[string]*project.Package{
		"internal/token":    tokenPkg,
		"internal/lexer":    lexerPkg,
		"internal/ast":      astPkg,
		"internal/parser":   parserPkg,
		"internal/sema":     semaPkg,
		"internal/project":  projectPkg,
		"internal/pipeline": pipelinePkg,
	}
	if err := selfhost.BuildAndTest("internal/backend", backendPkg, []string{"../backend/backend_selfhost_test.go"}, deps); err != nil {
		t.Fatalf("existing backend tests failed against the transpiled package: %v", err)
	}
}

// TestPortedTypecheckPackage validates US-002: the typecheck package (the depth
// checker — mustuse/nozero/implements analyses) reimplemented verbatim as goal
// source under selfhost/typecheck transpiles to compiling Go (the smoke gate,
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
// sources are at ../../selfhost/{token,lexer,ast,parser,sema,project,pipeline,
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
		"internal/token":     tokenPkg,
		"internal/lexer":     lexerPkg,
		"internal/ast":       astPkg,
		"internal/parser":    parserPkg,
		"internal/sema":      semaPkg,
		"internal/project":   projectPkg,
		"internal/pipeline":  pipelinePkg,
		"internal/backend":   backendPkg,
		"internal/typecheck": typecheckPkg,
	}
	if err := selfhost.BuildTranspiled(layout); err != nil {
		t.Fatalf("ported typecheck failed the transpile-and-build gate: %v", err)
	}

	// Criterion 3: the existing typecheck depth tests pass against the transpiled
	// package, with the ported dependency closure transpiled in.
	deps := map[string]*project.Package{
		"internal/token":    tokenPkg,
		"internal/lexer":    lexerPkg,
		"internal/ast":      astPkg,
		"internal/parser":   parserPkg,
		"internal/sema":     semaPkg,
		"internal/project":  projectPkg,
		"internal/pipeline": pipelinePkg,
		"internal/backend":  backendPkg,
	}
	testFiles := []string{
		"../typecheck/checker_test.go",
		"../typecheck/implements_test.go",
		"../typecheck/mustuse_test.go",
		"../typecheck/nozero_test.go",
		"../typecheck/typecheck_test.go",
	}
	if err := selfhost.BuildAndTest("internal/typecheck", typecheckPkg, testFiles, deps); err != nil {
		t.Fatalf("existing typecheck tests failed against the transpiled package: %v", err)
	}
}
