package lsptest

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"goal/internal/lsp"
)

// The LSP protocol measures Position.Character in UTF-16 code units, so a non-BMP rune on
// the line must widen every later column by one. Here 🚀 (U+1F680, two UTF-16 units) sits
// before an invalid binary literal `0b2`; the published diagnostic must land on the UTF-16
// column, one past the rune column, proving the conversion is explicit rather than a raw
// rune (or byte) count.
func TestLSPDiagnosticCharacterIsUTF16(t *testing.T) {
	const src = "package demo\n\nvar s, n = \"🚀\", 0b2\n"
	_, uri := writeTempPackage(t, "emoji.goal", src)

	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`1`), "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": src}},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}
	time.Sleep(800 * time.Millisecond)

	diags := lastPublishedDiagnostics(t, out.snapshot(), uri)
	var found bool
	for _, d := range diags {
		if d.Code != "invalid-number-literal" {
			continue
		}
		found = true
		// Rune column of `0` on `var s, n = "🚀", 0b2` is 16 (0-based); the astral 🚀
		// adds one UTF-16 unit, so the protocol column is 17.
		if got := d.Range.Start.Character; got != 17 {
			t.Errorf("diagnostic Character = %d, want 17 (UTF-16 units, not rune column 16)", got)
		}
	}
	if !found {
		t.Fatalf("no invalid-number-literal diagnostic published; got %+v", diags)
	}
}
