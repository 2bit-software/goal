package check

import (
	"strings"
	"testing"
)

// TestConvertForeignResolved proves the feature-12 deferral closes once the foreign
// source type is loaded: with the import resolvable, the checker reads ext.Outer's field
// set and proves the derive total — no unresolved-derive-type warning, no error.
func TestConvertForeignResolved(t *testing.T) {
	src := `package conv

import ext "goal/internal/check/testdata/extpkg"

type Local struct {
	ID    string
	Count int
}

derive func make(o *ext.Outer) Local
`
	diags := analyzeInDir(t, src)
	for _, d := range diags {
		if d.Code == "unresolved-derive-type" {
			t.Errorf("expected no deferral once foreign type resolves, got: %s", d.Message)
		}
		if d.Severity == Error {
			t.Errorf("unexpected error: [%s] %s", d.Code, d.Message)
		}
	}
}

// TestConvertForeignUnsourcedField proves completeness is genuinely enforced across the
// boundary: a target field with no same-named field on the foreign source is a located
// Error, not a silent zero.
func TestConvertForeignUnsourcedField(t *testing.T) {
	src := `package conv

import ext "goal/internal/check/testdata/extpkg"

type Local struct {
	ID    string
	Count int
	Extra string
}

derive func make(o *ext.Outer) Local
`
	diags := analyzeInDir(t, src)
	var found bool
	for _, d := range diags {
		if d.Code == "unsourced-field" && strings.Contains(d.Message, "Extra") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an unsourced-field error for Local.Extra, got %+v", diags)
	}
}

func analyzeInDir(t *testing.T, src string) []Diagnostic {
	t.Helper()
	perFile, err := AnalyzePackageInDir([]string{src}, "testdata")
	if err != nil {
		t.Fatalf("AnalyzePackageInDir: %v", err)
	}
	return perFile[0]
}
