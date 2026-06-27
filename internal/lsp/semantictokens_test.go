package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"goal/internal/check"
)

// semTokSrc exercises the three goal constructs the acceptance criterion calls out — an enum
// declaration, a match, and a postfix `?` — alongside ordinary functions so the classifier is
// proven on the AST-driven roles, not just lexer-level keywords.
const semTokSrc = `package p

enum Color {
	Red
	Green
}

func describe(c Color) string {
	match c {
		Color.Red => "r"
		Color.Green => "g"
	}
	return "x"
}

func load() (int, error) {
	x := parse()?
	return x, nil
}

func parse() (int, error) {
	return 0, nil
}
`

// absTok is one semantic token decoded back from the wire's delta encoding to absolute
// 0-based line/char, its length, and its token-type index.
type absTok struct {
	line, char, length, typ int
}

// decodeSemTokens reverses the LSP delta encoding into absolute tokens, in document order.
func decodeSemTokens(t *testing.T, data []uint) []absTok {
	t.Helper()
	if len(data)%5 != 0 {
		t.Fatalf("semantic token data length %d is not a multiple of 5", len(data))
	}
	var out []absTok
	line, char := 0, 0
	for i := 0; i < len(data); i += 5 {
		dl, dc := int(data[i]), int(data[i+1])
		if dl == 0 {
			char += dc
		} else {
			line += dl
			char = dc
		}
		out = append(out, absTok{line: line, char: char, length: int(data[i+2]), typ: int(data[i+3])})
	}
	return out
}

// typeOfFirst returns the semantic token-type at the first occurrence of sub in src, mapping
// the byte offset to the 0-based line/char the token stream is keyed by.
func typeOfFirst(t *testing.T, src string, toks []absTok, sub string) int {
	t.Helper()
	off := strings.Index(src, sub)
	if off < 0 {
		t.Fatalf("substring %q not found in source", sub)
	}
	p := check.OffsetToPosition(src, off)
	l, c := p.Line-1, p.Col-1
	for _, tk := range toks {
		if tk.line == l && tk.char == c {
			return tk.typ
		}
	}
	t.Fatalf("no semantic token at %q (line %d, char %d)", sub, l, c)
	return -1
}

// The enum name, its variants, the match keyword, and the postfix `?` are each classified
// with their expected semantic role — the acceptance-criteria sample.
func TestComputeSemanticTokensEnumMatchQuestion(t *testing.T) {
	toks := decodeSemTokens(t, computeSemanticTokens(semTokSrc))
	if len(toks) == 0 {
		t.Fatal("expected semantic tokens, got none")
	}
	checks := []struct {
		sub  string
		want int
		name string
	}{
		{"Color", semEnum, "enum name"},
		{"Red", semEnumMember, "enum variant Red"},
		{"Green", semEnumMember, "enum variant Green"},
		{"match", semKeyword, "match keyword"},
		{"?", semOperator, "postfix ? operator"},
		{"describe", semFunction, "function name"},
	}
	for _, c := range checks {
		if got := typeOfFirst(t, semTokSrc, toks, c.sub); got != c.want {
			t.Errorf("%s (%q): type = %s, want %s",
				c.name, c.sub, semanticTokenTypes[got], semanticTokenTypes[c.want])
		}
	}
}

// The token stream is well-formed: a multiple of 5, every token within the legend, and
// non-overlapping in strictly non-decreasing document order.
func TestSemanticTokensWellFormed(t *testing.T) {
	data := computeSemanticTokens(semTokSrc)
	if len(data)%5 != 0 {
		t.Fatalf("data length %d not a multiple of 5", len(data))
	}
	toks := decodeSemTokens(t, data)
	prevLine, prevEnd := 0, 0
	for i, tk := range toks {
		if tk.typ < 0 || tk.typ >= len(semanticTokenTypes) {
			t.Fatalf("token %d type %d out of legend range", i, tk.typ)
		}
		if tk.length <= 0 {
			t.Fatalf("token %d has non-positive length %d", i, tk.length)
		}
		if tk.line < prevLine {
			t.Fatalf("token %d line %d precedes previous line %d", i, tk.line, prevLine)
		}
		if tk.line == prevLine && tk.char < prevEnd {
			t.Fatalf("token %d at char %d overlaps previous token ending at %d", i, tk.char, prevEnd)
		}
		if tk.line != prevLine {
			prevEnd = 0
		}
		prevLine, prevEnd = tk.line, tk.char+tk.length
	}
}

// Empty and unparseable documents yield a non-nil empty token set and never panic.
func TestSemanticTokensEmptyAndUnparseable(t *testing.T) {
	if got := computeSemanticTokens(""); got == nil {
		t.Error("empty source should yield a non-nil (empty) data slice")
	}
	// A broken document still tokenizes at the lexer level (so keywords classify) but its AST
	// roles are empty; it must not panic.
	_ = computeSemanticTokens("package p\n\nenum Broken {\n")
}

// The semanticTokens handler returns tokens for an open document and an empty (non-nil) set
// for an unknown URI.
func TestSemanticTokensHandler(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, semTokSrc, 1)

	raw, _ := json.Marshal(SemanticTokensParams{TextDocument: textDocumentIdentifier{URI: uri}})
	got := s.semanticTokens(raw)
	if len(got.Data) == 0 {
		t.Error("handler returned no tokens for an open document")
	}

	unknown, _ := json.Marshal(SemanticTokensParams{TextDocument: textDocumentIdentifier{URI: "file:///pkg/missing.goal"}})
	if miss := s.semanticTokens(unknown); miss.Data == nil || len(miss.Data) != 0 {
		t.Errorf("unknown URI should yield an empty (non-nil) token set, got %v", miss.Data)
	}
}
