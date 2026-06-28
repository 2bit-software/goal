package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"goal/internal/token"
)

// defSrc declares an enum, a function called before its own declaration, and an Enum.Variant
// reference, so go-to-definition is exercised on a forward function call and an enum variant.
const defSrc = `package p

enum Color {
	Red
	Green
}

func pick() Color {
	return Color.Red
}

func describe(c Color) string {
	match c {
		Color.Red => "r"
		Color.Green => "g"
	}
	return run()
}

func run() string {
	return "x"
}
`

// offsetOfNth returns the byte offset of the n-th (0-based) occurrence of sub in src.
func offsetOfNth(t *testing.T, src, sub string, n int) int {
	t.Helper()
	off := 0
	for i := 0; i <= n; i++ {
		idx := strings.Index(src[off:], sub)
		if idx < 0 {
			t.Fatalf("occurrence %d of %q not found", n, sub)
		}
		off += idx
		if i < n {
			off += len(sub)
		}
	}
	return off
}

// cursorAt converts a byte offset into the 0-based line/char the definition handler is keyed by.
func cursorAt(src string, off int) (line, char int) {
	p := token.OffsetToPosition(src, off)
	return p.Line - 1, p.Col - 1
}

// wantRangeAt asserts that got begins exactly at the 0-based position of src offset off — i.e.
// the resolved declaration's name starts where the declaration is.
func wantRangeAt(t *testing.T, src string, got Range, off int, what string) {
	t.Helper()
	wl, wc := cursorAt(src, off)
	if got.Start.Line != wl || got.Start.Character != wc {
		t.Errorf("%s: definition at (%d,%d), want (%d,%d)",
			what, got.Start.Line, got.Start.Character, wl, wc)
	}
}

// A function call resolves to its function declaration's name position, even when the call
// precedes the declaration in the file (forward reference).
func TestDefinitionFunctionCall(t *testing.T) {
	callOff := offsetOfNth(t, defSrc, "run", 0) // the call `run()` inside describe
	declOff := strings.Index(defSrc, "func run") + len("func ")
	line, char := cursorAt(defSrc, callOff)

	got, ok := resolveDefinition(defSrc, line, char)
	if !ok {
		t.Fatal("function call did not resolve to a definition")
	}
	wantRangeAt(t, defSrc, got, declOff, "function call")
}

// An enum variant reference resolves to the variant's declaration inside its enum.
func TestDefinitionEnumVariant(t *testing.T) {
	// The 2nd "Red" is the `Color.Red` reference in pick(); the 1st is the enum declaration.
	refOff := offsetOfNth(t, defSrc, "Red", 1)
	declOff := offsetOfNth(t, defSrc, "Red", 0)
	line, char := cursorAt(defSrc, refOff)

	got, ok := resolveDefinition(defSrc, line, char)
	if !ok {
		t.Fatal("enum variant did not resolve to a definition")
	}
	wantRangeAt(t, defSrc, got, declOff, "enum variant")
}

// A reference to an enum/type name (the `Color` in `Color.Red`) resolves to its declaration.
func TestDefinitionTypeName(t *testing.T) {
	// The `Color` enum name reference inside pick()'s `return Color.Red`.
	refOff := offsetOfNth(t, defSrc, "Color.Red", 0)
	declOff := strings.Index(defSrc, "enum Color") + len("enum ")
	line, char := cursorAt(defSrc, refOff)

	got, ok := resolveDefinition(defSrc, line, char)
	if !ok {
		t.Fatal("type name did not resolve to a definition")
	}
	wantRangeAt(t, defSrc, got, declOff, "type name")
}

// A position over whitespace or an out-of-range coordinate resolves to nothing.
func TestDefinitionNoSymbol(t *testing.T) {
	// A blank line (the line after `package p`) holds no identifier.
	if _, ok := resolveDefinition(defSrc, 1, 0); ok {
		t.Error("blank line should not resolve to a definition")
	}
	if _, ok := resolveDefinition(defSrc, 9999, 0); ok {
		t.Error("a line past EOF should not resolve")
	}
}

// Unparseable source yields no definition and does not panic.
func TestDefinitionUnparseable(t *testing.T) {
	if _, ok := resolveDefinition("package p\n\nenum Broken {\n", 2, 5); ok {
		t.Error("unparseable source should not resolve to a definition")
	}
}

// The definition handler returns a Location for an open document and null for an unknown URI.
func TestDefinitionHandler(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, defSrc, 1)

	callOff := offsetOfNth(t, defSrc, "run", 0)
	line, char := cursorAt(defSrc, callOff)
	raw, _ := json.Marshal(DefinitionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: char},
	})
	if loc := s.definition(raw); loc == nil || loc.URI != uri {
		t.Fatalf("handler returned %+v, want a Location for %s", loc, uri)
	}

	unknown, _ := json.Marshal(DefinitionParams{
		TextDocument: textDocumentIdentifier{URI: "file:///pkg/missing.goal"},
		Position:     Position{Line: line, Character: char},
	})
	if loc := s.definition(unknown); loc != nil {
		t.Errorf("unknown URI should yield a null definition, got %+v", loc)
	}
}

// Initialize advertises the definition capability.
func TestServerAdvertisesDefinition(t *testing.T) {
	var in, out bytes.Buffer
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}))
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "method": "exit"}))

	s := NewServer(&out)
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"definitionProvider":true`)) {
		t.Fatalf("initialize did not advertise definitions; output:\n%s", out.String())
	}
}
