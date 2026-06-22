package pass

import (
	"strings"
	"testing"

	"goal/internal/analyze"
)

// TestMatchValuePositionInference covers L3: a value-position `name := match` lowers to a
// typed `var name T` switch when the result type is inferable from the arm bodies (one
// enum, string, or bool), and otherwise keeps the honest located deferral.
func TestMatchValuePositionInference(t *testing.T) {
	cases := []struct {
		name         string
		src          string
		wantContains string // substring expected in the lowered output (when inferable)
		wantErr      string // substring expected in the error (when deferred)
	}{
		{
			name:         "enum inferred",
			src:          "package p\nenum C { A, B }\nenum L { X, Y }\nfunc f(c C) L {\n\tx := match c { C.A => L.X\n\tC.B => L.Y }\n\treturn x\n}\n",
			wantContains: "var x L",
		},
		{
			name:         "string inferred",
			src:          "package p\nenum C { A, B }\nfunc f(c C) string {\n\tx := match c { C.A => \"a\"\n\tC.B => \"b\" }\n\treturn x\n}\n",
			wantContains: "var x string",
		},
		{
			name:         "bool inferred",
			src:          "package p\nenum C { A, B }\nfunc f(c C) bool {\n\tx := match c { C.A => true\n\tC.B => false }\n\treturn x\n}\n",
			wantContains: "var x bool",
		},
		{
			name:    "numeric defers",
			src:     "package p\nenum C { A, B }\nfunc f(c C) int {\n\tx := match c { C.A => 1\n\tC.B => 2 }\n\treturn x\n}\n",
			wantErr: "inferable result type",
		},
		{
			name:    "mixed kinds defer",
			src:     "package p\nenum C { A, B }\nfunc f(c C) any {\n\tx := match c { C.A => \"a\"\n\tC.B => true }\n\treturn x\n}\n",
			wantErr: "inferable result type",
		},
		{
			name:    "two enums defer",
			src:     "package p\nenum C { A, B }\nenum L { X }\nenum M { P }\nfunc f(c C) any {\n\tx := match c { C.A => L.X\n\tC.B => M.P }\n\treturn x\n}\n",
			wantErr: "inferable result type",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Match(tc.src, analyze.Build(tc.src))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected deferral error containing %q, got success:\n%s", tc.wantErr, out)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, tc.wantContains) {
				t.Fatalf("output missing %q:\n%s", tc.wantContains, out)
			}
		})
	}
}
