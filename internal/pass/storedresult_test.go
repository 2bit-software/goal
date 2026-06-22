package pass

import (
	"strings"
	"testing"

	"goal/internal/analyze"
)

// TestStoredResultGuard covers L4a: a Result stored as a value (slice/map/array element or
// struct/enum field) is refused with a located §8.7 error, while a Result consumed
// immediately (a function return), a Result parameter (left to the open §8.7 question), and
// a stored Option (which lowers to *T) all pass through untouched.
func TestStoredResultGuard(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string // substring expected in the error; "" means the guard must pass
	}{
		// --- rejected: unambiguously stored, provably broken today ---
		{
			name:    "slice element",
			src:     "package p\nfunc f() []Result[int, error] { return nil }\n",
			wantErr: "stored as a slice/array/map element",
		},
		{
			name:    "map value",
			src:     "package p\nfunc f() map[string]Result[int, error] { return nil }\n",
			wantErr: "stored as a slice/array/map element",
		},
		{
			name:    "array element",
			src:     "package p\nfunc f() [2]Result[int, error] { var a [2]Result[int, error]; return a }\n",
			wantErr: "stored as a slice/array/map element",
		},
		{
			name:    "struct field",
			src:     "package p\ntype Box struct { r Result[int, error] }\n",
			wantErr: "stored as a struct/enum field",
		},
		{
			name:    "located",
			src:     "package p\ntype Box struct { r Result[int, error] }\n",
			wantErr: "at 2:21",
		},
		// --- allowed: immediate use, ambiguous param, and stored Option pass through ---
		{name: "function return ok", src: "package p\nfunc f() Result[int, error] { return Result.Ok(1) }\n"},
		{name: "multi-return ok", src: "package p\nfunc f() (Result[int, error], int) { return Result.Ok(1), 0 }\n"},
		{name: "result param ok (deferred §8.7)", src: "package p\nfunc f(r Result[int, error]) int { return 0 }\n"},
		{name: "closed-E field ok (sum is first-class)", src: "package p\nenum E { Bad }\ntype Box struct { r Result[int, E] }\n"},
		{name: "closed-E slice ok", src: "package p\nenum E { Bad }\nfunc f() []Result[int, E] { return nil }\n"},
		{name: "option field ok", src: "package p\ntype Box struct { o Option[int] }\n"},
		{name: "slice of option ok", src: "package p\nfunc f() []Option[int] { return nil }\n"},
		{name: "no result", src: "package p\ntype Box struct { n int }\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := StoredResultGuard(tc.src, analyze.Build(tc.src))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected the guard to pass, got error: %v", err)
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
