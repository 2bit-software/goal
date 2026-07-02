package tokentest

import (
	"strings"
	"testing"

	"goal/internal/token"
)

// OffsetToPosition reports a rune column, not a byte column, so a diagnostic points at the
// offending character on a multibyte line. "héllo" holds a 2-byte é, so the ':' after
// "héllo " is rune column 7 even though it is byte column 8.
func TestOffsetToPositionRuneColumn(t *testing.T) {
	src := "héllo := 1\n"
	off := strings.IndexByte(src, ':')
	p := token.OffsetToPosition(src, off)
	if p.Line != 1 {
		t.Fatalf("line = %d, want 1", p.Line)
	}
	if p.Col != 7 {
		t.Errorf("col = %d, want 7 (rune-aware: the 2-byte é is one column)", p.Col)
	}
}

// ColFor is the single column authority that OffsetToPosition and the lexer both route
// through. A tab is one rune (one column) and a 4-byte astral rune is one column too.
func TestColForCountsRunesNotBytes(t *testing.T) {
	if got := token.ColFor("\tx", 0, 2); got != 3 {
		t.Errorf("ColFor after tab+x = %d, want 3", got)
	}
	if got := token.ColFor("🚀y", 0, len("🚀")); got != 2 {
		t.Errorf("ColFor after a 4-byte emoji = %d, want 2", got)
	}
}
