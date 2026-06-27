package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// hoverSrc declares a Result-returning function (with a /// doc comment), an enum, and a
// reference to the function, so hover is exercised on a function signature and its doc.
const hoverSrc = `package p

enum Color {
	Red
	Green
}

/// Convert text into a number.
/// It fails on bad input.
func parse(s string) Result[int, error] {
	return Result.Ok(0)
}

func main() {
	parse("1")?
	pick()
}

func pick() Color {
	return Color.Red
}
`

// Hover over a Result-returning function's call reports its signature including the Result
// result type (AC-2).
func TestHoverResultFunctionSignature(t *testing.T) {
	// The 2nd "parse" (index 1) is the call inside main(); index 0 is the declaration.
	callOff := offsetOfNth(t, hoverSrc, "parse", 1)
	line, char := cursorAt(hoverSrc, callOff)

	info, ok := resolveHover(hoverSrc, line, char)
	if !ok {
		t.Fatal("function call did not resolve to a hover")
	}
	if !strings.Contains(info.signature, "func parse(s string) Result[int, error]") {
		t.Errorf("hover signature = %q, want it to contain the full func signature", info.signature)
	}
}

// Hover over a documented symbol returns its doc-comment text (AC-1).
func TestHoverReportsDocComment(t *testing.T) {
	declOff := offsetOfNth(t, hoverSrc, "parse", 0) // the declaration name
	line, char := cursorAt(hoverSrc, declOff)

	info, ok := resolveHover(hoverSrc, line, char)
	if !ok {
		t.Fatal("function declaration did not resolve to a hover")
	}
	if !strings.Contains(info.doc, "Convert text into a number.") ||
		!strings.Contains(info.doc, "It fails on bad input.") {
		t.Errorf("hover doc = %q, want both doc lines", info.doc)
	}
	// The rendered markdown carries both the signature fence and the doc.
	rendered := info.render()
	if !strings.Contains(rendered, "```goal") || !strings.Contains(rendered, "Result[int, error]") {
		t.Errorf("rendered hover = %q, want a goal code fence with the signature", rendered)
	}
}

// Hover over an enum-variant reference describes the variant.
func TestHoverEnumVariant(t *testing.T) {
	refOff := offsetOfNth(t, hoverSrc, "Red", 1) // the Color.Red in pick()
	line, char := cursorAt(hoverSrc, refOff)

	info, ok := resolveHover(hoverSrc, line, char)
	if !ok {
		t.Fatal("enum variant did not resolve to a hover")
	}
	if !strings.Contains(info.signature, "Color.Red") {
		t.Errorf("variant hover = %q, want it to mention Color.Red", info.signature)
	}
}

// A position over whitespace or out of range resolves to nothing.
func TestHoverNoSymbol(t *testing.T) {
	if _, ok := resolveHover(hoverSrc, 1, 0); ok { // blank line after `package p`
		t.Error("blank line should not resolve to a hover")
	}
	if _, ok := resolveHover(hoverSrc, 9999, 0); ok {
		t.Error("a line past EOF should not resolve")
	}
}

// Unparseable source yields no hover and does not panic.
func TestHoverUnparseable(t *testing.T) {
	if _, ok := resolveHover("package p\n\nenum Broken {\n", 2, 5); ok {
		t.Error("unparseable source should not resolve to a hover")
	}
}

// The hover handler returns a Hover for an open document and null for an unknown URI.
func TestHoverHandler(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, hoverSrc, 1)

	callOff := offsetOfNth(t, hoverSrc, "parse", 1)
	line, char := cursorAt(hoverSrc, callOff)
	raw, _ := json.Marshal(HoverParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: char},
	})
	h := s.hover(raw)
	if h == nil || !strings.Contains(h.Contents.Value, "func parse") {
		t.Fatalf("handler returned %+v, want a Hover describing parse", h)
	}

	unknown, _ := json.Marshal(HoverParams{
		TextDocument: textDocumentIdentifier{URI: "file:///pkg/missing.goal"},
		Position:     Position{Line: line, Character: char},
	})
	if h := s.hover(unknown); h != nil {
		t.Errorf("unknown URI should yield a null hover, got %+v", h)
	}
}

// Initialize advertises the hover capability.
func TestServerAdvertisesHover(t *testing.T) {
	var in, out bytes.Buffer
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}))
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "method": "exit"}))

	s := NewServer(&out)
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"hoverProvider":true`)) {
		t.Fatalf("initialize did not advertise hover; output:\n%s", out.String())
	}
}
