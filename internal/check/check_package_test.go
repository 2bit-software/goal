package check

import "testing"

// The enum lives in one file; a non-exhaustive match over it lives in another. The
// exhaustiveness guarantee can only fire when the enum resolves — which, across files,
// only the package-level merged tables provide.
const enumFile = `package demo

enum Shape {
    Circle { r: float64 }
    Square { s: float64 }
}
`

const matchFile = `package demo

func describe(sh Shape) string {
    return match sh {
        Shape.Circle(c) => "circle"
    }
}
`

func hasError(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Severity == Error && d.Code == code {
			return true
		}
	}
	return false
}

func TestAnalyzePackageResolvesCrossFileExhaustiveness(t *testing.T) {
	// Control: checked alone, the match file cannot see Shape, so exhaustiveness is
	// deferred — no non-exhaustive Error is raised.
	solo, err := Analyze(matchFile)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if hasError(solo, "non-exhaustive-match") {
		t.Fatal("single-file check unexpectedly resolved the cross-file enum")
	}

	// Package: with merged tables Shape resolves, and the missing Square variant is
	// caught as a real error in the match file (index 1).
	perFile, err := AnalyzePackage([]string{enumFile, matchFile})
	if err != nil {
		t.Fatalf("AnalyzePackage: %v", err)
	}
	if len(perFile) != 2 {
		t.Fatalf("want diagnostics for 2 files, got %d", len(perFile))
	}
	if hasError(perFile[0], "non-exhaustive-match") {
		t.Error("the enum-only file should have no match error")
	}
	if !hasError(perFile[1], "non-exhaustive-match") {
		t.Errorf("cross-file exhaustiveness not caught; diags: %+v", perFile[1])
	}
}
