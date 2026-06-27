package typecheck

import (
	"testing"

	"goal/internal/project"
)

// directDepthChecks reproduces the concrete depth-check path (Load then every Check*) so a
// test can assert the TypeChecker interface produces exactly the same diagnostics.
func directDepthChecks(t *testing.T, pkg *project.Package) []Diagnostic {
	t.Helper()
	p, err := Load(pkg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var diags []Diagnostic
	diags = append(diags, CheckImplements(p)...)
	diags = append(diags, CheckMustUse(p)...)
	diags = append(diags, CheckNoZeroValue(p)...)
	return diags
}

func diagStrings(diags []Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = d.String()
	}
	return out
}

// GoTypesChecker must satisfy the TypeChecker interface — the seam's central promise.
var _ TypeChecker = GoTypesChecker{}

// TestTypeCheckerInterfaceParity exercises the depth checks THROUGH the TypeChecker
// interface and asserts they yield exactly the diagnostics of the concrete path, for both
// a clean package and one that trips a depth check (the elided no-zero-value literal). This
// is the US-028 acceptance test: the existing typecheck cases still pass through the seam.
func TestTypeCheckerInterfaceParity(t *testing.T) {
	// A literal that omits a field, resolved only via go/types — exercises CheckNoZeroValue
	// so the parity assertion covers a non-empty diagnostic set, not just the clean case.
	const elided = `package demo

type Inner struct {
    a int
    b int
}

func f() []Inner {
    return []Inner{{a: 1}}
}
`
	cases := map[string]string{
		"clean":  harnessSrc, // implements + import + local, no violations
		"elided": elided,     // one elided-missing-field diagnostic
	}

	var tc TypeChecker = GoTypesChecker{}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			pkg := pkgOf(map[string]string{"x.goal": src})

			got, err := tc.Check(pkg)
			if err != nil {
				t.Fatalf("TypeChecker.Check: %v", err)
			}
			want := directDepthChecks(t, pkgOf(map[string]string{"x.goal": src}))

			gs, ws := diagStrings(got), diagStrings(want)
			if len(gs) != len(ws) {
				t.Fatalf("diagnostic count through interface=%d, direct=%d\n interface=%v\n direct=%v",
					len(gs), len(ws), gs, ws)
			}
			for i := range gs {
				if gs[i] != ws[i] {
					t.Errorf("diagnostic[%d] through interface=%q, direct=%q", i, gs[i], ws[i])
				}
			}
		})
	}

	// The elided case must actually have produced a diagnostic, or the parity check above is
	// vacuous for the non-clean path.
	if d, err := tc.Check(pkgOf(map[string]string{"x.goal": cases["elided"]})); err != nil {
		t.Fatalf("Check(elided): %v", err)
	} else if len(d) == 0 {
		t.Fatal("elided case produced no diagnostics through the interface")
	}
}

// TestTypeCheckerErrorsOnBadTranspile confirms the seam surfaces a compiler-level
// transpile/parse failure as an error (not a panic, not a silent empty result), matching
// Load's contract.
func TestTypeCheckerErrorsOnBadTranspile(t *testing.T) {
	var tc TypeChecker = GoTypesChecker{}
	// Missing the package clause: the lowered Go fails to parse, so Load (and thus Check)
	// returns an error.
	bad := pkgOf(map[string]string{"x.goal": "func f() {}\n"})
	if _, err := tc.Check(bad); err == nil {
		t.Fatal("expected an error for an untranspilable package, got nil")
	}
}
