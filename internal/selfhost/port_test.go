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
	if err := selfhost.BuildAndTest("internal/token", pkg, []string{"../token/token_test.go"}); err != nil {
		t.Fatalf("existing token tests failed against the transpiled package: %v", err)
	}
}
