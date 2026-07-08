package corpus

import (
	"encoding/json"
	"testing"
)

// repoRoot is the repository root relative to this package directory
// (internal/corpus).
const repoRoot = "../.."

func TestGenerateCounts(t *testing.T) {
	m, err := Generate(repoRoot)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var transpile, pkg, check, doctest, other int
	for _, c := range m.Cases {
		switch c.Kind {
		case KindTranspile:
			if c.Mode == ModePackage {
				pkg++
			} else {
				transpile++
			}
		case KindCheck:
			check++
		case KindDoctest:
			doctest++
		default:
			other++
		}
	}

	if transpile != 70 {
		t.Errorf("file-mode transpile pairs = %d, want 70", transpile)
	}
	// Package-mode cases reify the formerly-inline cross-file and foreign-derive
	// package tests as on-disk fixtures.
	if pkg != 2 {
		t.Errorf("package-mode cases = %d, want 2", pkg)
	}
	if check != 67 {
		t.Errorf("check cases = %d, want 67", check)
	}
	// Doctest cases are additive: each transpile pair whose golden is an emitted
	// _test.go sidecar (the four feature-11 examples, the testdata qprop_assign_eq
	// `?`-assignment case, and the testdata option_some_copy value-capture case)
	// also yields a doctest case.
	if doctest != 8 {
		t.Errorf("doctest cases = %d, want 8", doctest)
	}
	if other != 0 {
		t.Errorf("unexpected non-transpile/non-check/non-doctest cases = %d, want 0", other)
	}
}

// TestGenerateDeterministic asserts repeated generation over an unchanged
// corpus produces byte-identical JSON, so the committed manifest is diffable.
func TestGenerateDeterministic(t *testing.T) {
	first, err := Generate(repoRoot)
	if err != nil {
		t.Fatalf("Generate (first): %v", err)
	}
	second, err := Generate(repoRoot)
	if err != nil {
		t.Fatalf("Generate (second): %v", err)
	}

	a, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	b, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(a) != string(b) {
		t.Error("Generate is not deterministic: two runs differ")
	}
}

// TestGenerateNonDestructiveShape sanity-checks the emitted case fields so a
// future runner can rely on them.
func TestGenerateNonDestructiveShape(t *testing.T) {
	m, err := Generate(repoRoot)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, c := range m.Cases {
		if c.ID == "" {
			t.Errorf("case with empty ID: %+v", c)
		}
		if c.Input == "" {
			t.Errorf("case %q has empty Input", c.ID)
		}
		switch c.Kind {
		case KindTranspile:
			if c.Mode == ModePackage {
				// Package cases carry their sources and import map in Package,
				// not in the single Input/Expected pair.
				if c.Package == nil {
					t.Errorf("package case %q has nil Package spec", c.ID)
				} else if len(c.Package.Files) == 0 {
					t.Errorf("package case %q has no files", c.ID)
				}
				break
			}
			if c.Expected == "" {
				t.Errorf("transpile case %q has empty Expected", c.ID)
			}
			if c.Normalize != NormalizeGofmt {
				t.Errorf("transpile case %q Normalize = %q, want gofmt", c.ID, c.Normalize)
			}
		case KindCheck:
			if c.Normalize != NormalizeNone {
				t.Errorf("check case %q Normalize = %q, want none", c.ID, c.Normalize)
			}
		}
	}
}
