package corpus

import (
	"strings"
	"testing"

	"goal/internal/backend"
)

// TestASTEngineWholeCorpusBehavioralGate is US-041: the whole-corpus behavioral
// parity gate for the new (AST) engine. It runs EVERY case in the committed
// manifest — all 108, no filtered subset — through the AST front-end at the
// behavioral conformance tier appropriate to its kind/mode, and reports zero
// failures:
//
//   - transpile (file)    -> RunCompile via backend.Transpile: the lowered Go is
//     written to a temp module and must go build + go vet cleanly.
//   - transpile (package) -> RunPackage via backend.TranspilePackage: the package
//     is lowered cross-file with one shared prelude and foreign imports wired in,
//     then built.
//   - check               -> RunCheck via the AST checker (sema.Check, behind
//     SemaCheck): every inline // want marker must match a diagnostic and no
//     unclaimed Error may fire.
//   - doctest             -> RunDoctestExec via backend.Transpile: the emitted
//     _test.go sidecar is executed with go test in a temp module.
//
// This is the single gate that proves the flip to the AST engine is safe: judged
// by behavior, not Go spelling, the AST engine is conformant across the entire
// corpus. It spawns the go toolchain heavily, so it is skipped under -short; the
// project verifyCommands run it without -short.
func TestASTEngineWholeCorpusBehavioralGate(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain per case; skipped under -short")
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}
	if len(m.Cases) == 0 {
		t.Fatalf("manifest %q is empty — a filtered or mis-generated manifest cannot pass as green", manifestPath)
	}

	tp := TranspilerFunc(backend.Transpile)
	pt := PackageTranspilerFunc(backend.TranspilePackage)
	ck := CheckerFunc(SemaCheck)

	var ran, transpile, pkg, check, doctest int
	for _, c := range m.Cases {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			switch {
			case c.Kind == KindCheck:
				if err := RunCheck(repoRoot, c, ck); err != nil {
					t.Error(err)
				}
			case c.Kind == KindDoctest:
				if err := RunDoctestExec(repoRoot, c, tp); err != nil {
					t.Error(err)
				}
			case c.Kind == KindTranspile && c.Mode == ModePackage:
				if err := RunPackage(repoRoot, c, pt); err != nil {
					t.Error(err)
				}
			case c.Kind == KindTranspile:
				if err := RunCompile(repoRoot, c, tp); err != nil {
					t.Error(err)
				}
			default:
				t.Fatalf("case %q has unhandled kind/mode %q/%q", c.ID, c.Kind, c.Mode)
			}
		})
		ran++
		switch {
		case c.Kind == KindCheck:
			check++
		case c.Kind == KindDoctest:
			doctest++
		case c.Mode == ModePackage:
			pkg++
		default:
			transpile++
		}
	}

	// Guard the gate's own scope: it must span the whole corpus across every tier,
	// so a future change that silently drops a tier (e.g. stops indexing package or
	// doctest cases) fails here rather than passing a narrowed gate.
	if ran != len(m.Cases) {
		t.Fatalf("gate ran %d of %d manifest cases", ran, len(m.Cases))
	}
	if transpile == 0 || pkg == 0 || check == 0 || doctest == 0 {
		t.Fatalf("gate must exercise every tier; got transpile=%d package=%d check=%d doctest=%d", transpile, pkg, check, doctest)
	}
	t.Logf("AST engine: %d cases green (transpile=%d package=%d check=%d doctest=%d)", ran, transpile, pkg, check, doctest)
}

// TestSemaAssertRunner drives every feature-10 (assert) check case through the
// AST-based SemaCheck via the Checker interface, proving the ported static-fold
// diagnostics (always-false Error, always-true Warning) match the same inline
// // want markers the lexical checker is judged against.
func TestSemaAssertRunner(t *testing.T) {
	runSemaCheckDir(t, "testdata/check/10-assert/")
}

// TestSemaConvertRunner drives every feature-12 (derive-convert) check case
// through the AST-based SemaCheck, proving the ported totality diagnostics
// (unsourced/unbridged/fallible-in-total) match the inline // want markers.
func TestSemaConvertRunner(t *testing.T) {
	runSemaCheckDir(t, "testdata/check/12-derive-convert/")
}

// runSemaCheckDir runs every check case under dir through SemaCheck, failing
// loudly if the manifest yields none (so a mis-generated manifest cannot pass).
func runSemaCheckDir(t *testing.T, dir string) {
	t.Helper()
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}
	ck := CheckerFunc(SemaCheck)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindCheck || !strings.HasPrefix(c.Input, dir) {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunCheck(repoRoot, c, ck); err != nil {
				t.Error(err)
			}
		})
	}
	if ran == 0 {
		t.Fatalf("manifest %q contains no %s check cases", manifestPath, dir)
	}
}
