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
