package pass

import (
	"strings"
	"testing"

	"goal/internal/analyze"
)

// TestDefaultsRejectsUnsafe covers feature 08's unsafe / no-safe-zero classification:
// `...defaults` must refuse a field whose zero would panic, deadlock, or has no valid
// variant (with a located error), while safe zeros and explicitly-set fields pass.
func TestDefaultsRejectsUnsafe(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string // substring expected in the error; "" means transpile must succeed
	}{
		{
			name:    "pointer",
			src:     "package p\ntype S struct{ a *int }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "nil pointer",
		},
		{
			name:    "map",
			src:     "package p\ntype S struct{ m map[string]int }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "nil map",
		},
		{
			name:    "chan",
			src:     "package p\ntype Ch = chan int\ntype S struct{ c Ch }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "nil channel",
		},
		{
			name:    "func",
			src:     "package p\ntype S struct{ fn func() }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "nil func",
		},
		{
			name:    "enum sum type",
			src:     "package p\nenum Color { Red, Green }\ntype S struct{ c Color }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "sum type has no valid zero variant",
		},
		{
			name:    "sealed interface",
			src:     "package p\nsealed interface Shape {}\ntype S struct{ s Shape }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "sum type has no valid zero variant",
		},
		{
			name:    "named method interface",
			src:     "package p\ntype I interface{ Foo() }\ntype S struct{ x I }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "nil interface",
		},
		{
			name:    "located",
			src:     "package p\ntype S struct{ m map[string]int }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "`...defaults` at 3:23",
		},
		// --- allowed: safe zeros fill silently ---
		{name: "slice ok", src: "package p\ntype S struct{ a []int }\nfunc f() S { return S{...defaults} }\n"},
		{name: "error ok", src: "package p\ntype S struct{ e error }\nfunc f() S { return S{...defaults} }\n"},
		{name: "any ok", src: "package p\ntype S struct{ v any }\nfunc f() S { return S{...defaults} }\n"},
		{name: "struct ok", src: "package p\ntype A struct{ n int }\ntype S struct{ a A }\nfunc f() S { return S{...defaults} }\n"},
		{name: "int-backed enum ok", src: "package p\ntype Role int\ntype S struct{ r Role }\nfunc f() S { return S{...defaults} }\n"},
		{name: "primitives ok", src: "package p\ntype S struct{ a int; b string; c bool }\nfunc f() S { return S{...defaults} }\n"},
		// --- explicit value escapes the check even for an unsafe type ---
		{name: "explicit nil pointer ok", src: "package p\ntype S struct{ a *int }\nfunc f() S { return S{a: nil, ...defaults} }\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Defaults(tc.src, analyze.Build(tc.src))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got success", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
