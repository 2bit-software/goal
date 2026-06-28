package main

import (
	"strings"
	"testing"
)

// TestRejects covers the unsafe / no-safe-zero classification: `...defaults` must
// refuse a field whose zero value would panic, deadlock, or has no valid variant, and
// emit a located error. Safe zeros (and explicitly-set fields) still pass.
func TestRejects(t *testing.T) {
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
			name:    "named method interface",
			src:     "package p\ntype I interface{ Foo() }\ntype S struct{ x I }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "nil interface",
		},
		{
			name:    "located position",
			src:     "package p\ntype S struct{ m map[string]int }\nfunc f() S { return S{...defaults} }\n",
			wantErr: "at 3:23",
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
			_, err := transpile(tc.src)
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
