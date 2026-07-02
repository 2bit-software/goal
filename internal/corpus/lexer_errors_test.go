package corpus

import (
	"strings"
	"testing"

	"goal/internal/lexer"
	"goal/internal/token"
)

// TestLexerReportsLocatedErrors is US-006: the lexer records unterminated
// string/raw-string/rune literals and malformed numeric literals as located
// lexer.Error values (surfaced downstream as `00-lex` diagnostics), instead of
// silently emitting a malformed token. internal/lexer is an emitted package, so
// this Go-only test lives in internal/corpus (per the repo's test-placement rule).
func TestLexerReportsLocatedErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{"unterminated string", `x := "abc`, "unterminated-string"},
		{"unterminated raw string", "x := `abc", "unterminated-raw-string"},
		{"unterminated rune", `x := 'a`, "unterminated-rune"},
		{"octal out of radix", `x := 0o889`, "invalid-number-literal"},
		{"binary out of radix", `x := 0b77`, "invalid-number-literal"},
		{"empty hex prefix", `x := 0x`, "invalid-number-literal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, errs := lexer.Scan(tc.src)
			if len(errs) != 1 {
				t.Fatalf("Scan(%q) errors = %d, want 1: %+v", tc.src, len(errs), errs)
			}
			e := errs[0]
			if e.Code != tc.code {
				t.Errorf("code = %q, want %q", e.Code, tc.code)
			}
			if e.Pos.Line != 1 || e.Pos.Col < 1 {
				t.Errorf("pos = %d:%d, want a located 1-based position on line 1", e.Pos.Line, e.Pos.Col)
			}
			if strings.TrimSpace(e.Msg) == "" {
				t.Errorf("message is empty")
			}
		})
	}
}

// TestValidLiteralsLexClean guards against false positives: well-formed literals
// (including the octal file-mode bits the compiler's own source uses) produce no
// lexer errors.
func TestValidLiteralsLexClean(t *testing.T) {
	for _, src := range []string{
		`x := "abc"`,
		"x := `abc`",
		`x := 'a'`,
		`x := 0o755`,
		`x := 0b1010`,
		`x := 0xff`,
		`x := 42`,
		`x := 3.14`,
	} {
		if _, errs := lexer.Scan(src); len(errs) != 0 {
			t.Errorf("Scan(%q) unexpectedly reported errors: %+v", src, errs)
		}
	}
}

// TestUnterminatedStringDoesNotSwallowNewline is the AC that a `\` at end-of-line
// inside an unterminated string no longer consumes the following newline: the
// next line must lex as ordinary code (an IDENT on line 2), not as continued
// string content.
func TestUnterminatedStringDoesNotSwallowNewline(t *testing.T) {
	src := "x := \"abc\\\n" + "y := 1\n"
	toks, errs := lexer.Scan(src)

	if len(errs) != 1 || errs[0].Code != "unterminated-string" || errs[0].Pos.Line != 1 {
		t.Fatalf("errors = %+v, want one unterminated-string on line 1", errs)
	}

	foundY := false
	for _, tk := range toks {
		if tk.Kind == token.IDENT && tk.Lit == "y" {
			foundY = true
			if tk.Pos.Line != 2 {
				t.Errorf("IDENT `y` at line %d, want line 2", tk.Pos.Line)
			}
		}
	}
	if !foundY {
		t.Errorf("line 2 was not lexed as code — the unterminated string swallowed the newline; tokens = %+v", toks)
	}
}
