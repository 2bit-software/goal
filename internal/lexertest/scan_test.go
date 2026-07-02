package lexertest

import (
	"testing"

	"goal/internal/lexer"
)

// A lexical error on a line prefixed by a multibyte rune reports its column at the
// offending rune, not shifted right by the rune's extra UTF-8 bytes. "héllo" holds a
// 2-byte é, so the unterminated string opening at rune column 10 is reported as 10.
func TestScanErrorColumnIsRuneAware(t *testing.T) {
	src := `héllo := "oops`
	_, errs := lexer.Scan(src)
	if len(errs) == 0 {
		t.Fatalf("expected an unterminated-string error, got none")
	}
	e := errs[0]
	if e.Code != "unterminated-string" {
		t.Fatalf("first error code = %q, want unterminated-string", e.Code)
	}
	if e.Pos.Col != 10 {
		t.Errorf("error column = %d, want 10 (rune-aware, not byte column 11)", e.Pos.Col)
	}
}
