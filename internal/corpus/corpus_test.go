package corpus

import (
	"os"
	"path/filepath"
	"testing"
)

// caseByID returns the case with the given ID, failing the test if absent.
func caseByID(t *testing.T, m Manifest, id string) Case {
	t.Helper()
	for _, c := range m.Cases {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("case %q not found in manifest", id)
	return Case{}
}

// TestLoadFixture loads the fixture manifest and asserts every field of one
// case of each Kind.
func TestLoadFixture(t *testing.T) {
	m, err := Load(filepath.Join("testdata", "manifest.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(m.Cases) != 3 {
		t.Fatalf("got %d cases, want 3", len(m.Cases))
	}

	transpile := caseByID(t, m, "transpile-enums")
	if transpile.Kind != KindTranspile {
		t.Errorf("transpile Kind = %q, want %q", transpile.Kind, KindTranspile)
	}
	if transpile.Input != "features/01-enums/examples/basic.goal" {
		t.Errorf("transpile Input = %q", transpile.Input)
	}
	if transpile.Expected != "features/01-enums/examples/basic.go.expected" {
		t.Errorf("transpile Expected = %q", transpile.Expected)
	}
	if transpile.Mode != ModeFile {
		t.Errorf("transpile Mode = %q, want %q", transpile.Mode, ModeFile)
	}
	if transpile.Normalize != NormalizeGofmt {
		t.Errorf("transpile Normalize = %q, want %q", transpile.Normalize, NormalizeGofmt)
	}

	check := caseByID(t, m, "check-exhaustive")
	if check.Kind != KindCheck {
		t.Errorf("check Kind = %q, want %q", check.Kind, KindCheck)
	}
	if check.Input != "testdata/check/exhaustive_missing.goal" {
		t.Errorf("check Input = %q", check.Input)
	}
	if check.Expected != "match is not exhaustive" {
		t.Errorf("check Expected = %q", check.Expected)
	}
	if check.Mode != ModeFile {
		t.Errorf("check Mode = %q, want %q", check.Mode, ModeFile)
	}
	if check.Normalize != NormalizeNone {
		t.Errorf("check Normalize = %q, want %q", check.Normalize, NormalizeNone)
	}

	doctest := caseByID(t, m, "doctest-basic")
	if doctest.Kind != KindDoctest {
		t.Errorf("doctest Kind = %q, want %q", doctest.Kind, KindDoctest)
	}
	if doctest.Input != "features/11-doctests/examples/basic.goal" {
		t.Errorf("doctest Input = %q", doctest.Input)
	}
	if doctest.Expected != "features/11-doctests/examples/basic.go.expected" {
		t.Errorf("doctest Expected = %q", doctest.Expected)
	}
	if doctest.Mode != ModeFile {
		t.Errorf("doctest Mode = %q, want %q", doctest.Mode, ModeFile)
	}
	if doctest.Normalize != NormalizeGofmt {
		t.Errorf("doctest Normalize = %q, want %q", doctest.Normalize, NormalizeGofmt)
	}
}

// TestLoadMissingFile asserts a missing manifest yields an error, not a panic.
func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join("testdata", "does-not-exist.json")); err == nil {
		t.Fatal("Load of missing file returned nil error, want error")
	}
}

// TestLoadMalformed asserts invalid JSON yields an error, not a panic.
func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load of malformed file returned nil error, want error")
	}
}
