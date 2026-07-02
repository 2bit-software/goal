package backendtest

import (
	"strings"
	"testing"

	"goal/internal/backend"
)

// commonDecls are the type + registry declarations shared by the parity inputs, so the
// inline and derive-func forms differ only in how the conversion is written.
const commonDecls = `package conv

type UUID struct {
	s string
}

func (u UUID) String() string {
	return u.s
}

from func uuidToString(u UUID) string {
	return u.String()
}

type Src struct {
	ID   UUID
	Name string
}

type Dst struct {
	ID   string
	Name string
}
`

// TestInlineDeriveParityWithDeriveFunc asserts an inline `...derive(src)` in a returned
// literal emits the same field-conversion statements as the equivalent bodyless
// `derive func` (FR-6). Both forms use the same parameter name, so the statements are
// byte-identical; no gensym normalization is needed for this infallible case.
func TestInlineDeriveParityWithDeriveFunc(t *testing.T) {
	inline := commonDecls + `
func convert(s Src) Dst {
	return Dst{...derive(s)}
}
`
	deriveFunc := commonDecls + `
derive func convert(s Src) Dst
`
	inlineStmts := conversionStmts(mustTranspile(t, inline))
	deriveStmts := conversionStmts(mustTranspile(t, deriveFunc))

	if len(inlineStmts) == 0 {
		t.Fatalf("no conversion statements found in inline output")
	}
	if strings.Join(inlineStmts, "\n") != strings.Join(deriveStmts, "\n") {
		t.Errorf("inline conversion statements differ from derive func:\ninline:\n%s\nderive func:\n%s",
			strings.Join(inlineStmts, "\n"), strings.Join(deriveStmts, "\n"))
	}
}

// TestInlineDeriveErrors covers the located rejections (FR-4, FR-5, FR-3). Each asserts
// backend.Transpile fails with a message that carries the spread position and names the
// field or the misuse. backend.Transpile does not run the checker, so these are the
// backend's own errors.
func TestInlineDeriveErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "non-return position",
			src: `package conv

type Src struct {
	Name string
}

type Dst struct {
	Name string
}

func sink(d Dst) {
}

func bad(s Src) {
	sink(Dst{...derive(s)})
}
`,
			want: "only supported in a return",
		},
		{
			name: "unsourceable field",
			src: `package conv

type Src struct {
	Name string
}

type Dst struct {
	Name  string
	Extra string
}

func f(s Src) Dst {
	return Dst{...derive(s)}
}
`,
			want: "cannot fill field `Extra`",
		},
		{
			name: "incompatible type, no conversion",
			src: `package conv

type Src struct {
	ID   int
	Name string
}

type Dst struct {
	ID   string
	Name string
}

func f(s Src) Dst {
	return Dst{...derive(s)}
}
`,
			want: "cannot fill field `ID`",
		},
		{
			name: "fallible field in non-fallible function",
			src: `package conv

type UUID struct {
	s string
}

from func parseUUID(s string) (UUID, error) {
	return UUID{s: s}, nil
}

type Src struct {
	ID string
}

type Dst struct {
	ID UUID
}

func f(s Src) Dst {
	return Dst{...derive(s)}
}
`,
			want: "fallible",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := backend.Transpile(c.src)
			if err == nil {
				t.Fatalf("expected a transpile error, got none")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), c.want)
			}
			if !strings.Contains(err.Error(), "...derive") {
				t.Errorf("error = %q, want it to name `...derive`", err.Error())
			}
		})
	}
}

// TestInlineDeriveRenamesOutputVarPastUserOut proves the derive output variable is
// gensym'd so it never collides with a user local named `out`: the user's `out` is
// preserved, and the emitted conversion uses a renamed output variable (`out1`)
// rather than relying on a block-scoped `var out` to shadow it (US-023). Shadowing
// alone was unsafe because an override expression reading the user `out` would bind
// to the inner declaration.
func TestInlineDeriveRenamesOutputVarPastUserOut(t *testing.T) {
	src := `package conv

type Src struct {
	Name string
}

type Dst struct {
	Name string
}

func convert(s Src) Dst {
	out := "unrelated"
	_ = out
	return Dst{...derive(s)}
}
`
	got := mustTranspile(t, src)
	if !strings.Contains(got, `out := "unrelated"`) {
		t.Errorf("expected the user `out` to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "var out1 Dst") {
		t.Errorf("expected the derive output var to be gensym-renamed to out1, got:\n%s", got)
	}
	if strings.Contains(got, "var out Dst") {
		t.Errorf("derive output var must not reuse the user identifier `out`, got:\n%s", got)
	}
}

// mustTranspile transpiles src and fails the test on error.
func mustTranspile(t *testing.T, src string) string {
	t.Helper()
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("transpile failed: %v", err)
	}
	return out.Go
}

// conversionStmts returns the trimmed `out.…` assignment lines of a generated function
// body, in order — the field conversions an inline `...derive` and a `derive func` must
// share.
func conversionStmts(goSrc string) []string {
	var stmts []string
	for _, line := range strings.Split(goSrc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "out.") {
			stmts = append(stmts, trimmed)
		}
	}
	return stmts
}
