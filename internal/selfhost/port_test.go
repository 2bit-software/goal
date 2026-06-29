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
