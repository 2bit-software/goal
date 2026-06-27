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

	var transpile, check, other int
	for _, c := range m.Cases {
		switch c.Kind {
		case KindTranspile:
			transpile++
		case KindCheck:
			check++
		default:
			other++
		}
	}

	if transpile != 51 {
		t.Errorf("transpile pairs = %d, want 51", transpile)
	}
	if check != 50 {
		t.Errorf("check cases = %d, want 50", check)
	}
	if other != 0 {
		t.Errorf("unexpected non-transpile/non-check cases = %d, want 0", other)
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
