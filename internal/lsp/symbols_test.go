package lsp

import (
	"bytes"
	"encoding/json"
	"testing"
)

// symbolSrc exercises every top-level declaration form the outline reports, and deliberately
// places a `type X = …` alias and bodyless `from`/`derive func`s next to other declarations
// to catch a range that over-runs into its successor.
const symbolSrc = `package p

type ID = string

enum Color {
	Red
	Green
}

type Point struct {
	X int
	Y int
}

sealed interface Shape {}

type Drawer interface {
	Draw() string
}

func area(p Point) int {
	return p.X
}

func (p Point) Sum() int {
	return p.X + p.Y
}

from func toColor(s string) Color

derive func mkPoint(s Point) Point
`

func symbolByName(syms []DocumentSymbol, name string) (DocumentSymbol, bool) {
	for _, s := range syms {
		if s.Name == name {
			return s, true
		}
	}
	return DocumentSymbol{}, false
}

// Every top-level declaration appears once with the expected symbol kind.
func TestCollectSymbolsKinds(t *testing.T) {
	syms := collectSymbols(symbolSrc)
	want := map[string]int{
		"ID":      symClass,
		"Color":   symEnum,
		"Point":   symStruct,
		"Shape":   symInterface,
		"Drawer":  symInterface,
		"area":    symFunction,
		"Sum":     symMethod,
		"toColor": symFunction,
		"mkPoint": symFunction,
	}
	if len(syms) != len(want) {
		t.Fatalf("got %d symbols, want %d: %+v", len(syms), len(want), syms)
	}
	for name, kind := range want {
		s, ok := symbolByName(syms, name)
		if !ok {
			t.Errorf("missing symbol %q", name)
			continue
		}
		if s.Kind != kind {
			t.Errorf("%q kind = %d, want %d", name, s.Kind, kind)
		}
		if s.SelectionRange.Start.Line < s.Range.Start.Line {
			t.Errorf("%q selectionRange starts before range", name)
		}
	}
}

// A bodyless alias or `from`/`derive func` must not extend its range over the declaration
// that follows it.
func TestCollectSymbolsBodylessDoesNotSwallow(t *testing.T) {
	syms := collectSymbols(symbolSrc)
	pairs := [][2]string{{"ID", "Color"}, {"toColor", "mkPoint"}}
	for _, p := range pairs {
		a, okA := symbolByName(syms, p[0])
		b, okB := symbolByName(syms, p[1])
		if !okA || !okB {
			t.Fatalf("expected both %q and %q present", p[0], p[1])
		}
		if a.Range.End.Line >= b.Range.Start.Line {
			t.Errorf("%q range (end line %d) overruns into %q (start line %d)",
				p[0], a.Range.End.Line, p[1], b.Range.Start.Line)
		}
	}
}

// An empty/partial document yields a non-nil empty result and never panics.
func TestCollectSymbolsEmptyAndPartial(t *testing.T) {
	if got := collectSymbols(""); got == nil || len(got) != 0 {
		t.Errorf("empty source should yield an empty (non-nil) slice, got %+v", got)
	}
	_ = collectSymbols("package p\n\ntype Broken struct {\n") // must not panic
}

// The documentSymbol request handler returns the outline for an open document and an empty
// result for an unknown one.
func TestDocumentSymbolHandler(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, symbolSrc, 1)

	raw, _ := json.Marshal(DocumentSymbolParams{TextDocument: textDocumentIdentifier{URI: uri}})
	if syms := s.documentSymbols(raw); len(syms) != 9 {
		t.Errorf("handler returned %d symbols, want 9", len(syms))
	}
	unknown, _ := json.Marshal(DocumentSymbolParams{TextDocument: textDocumentIdentifier{URI: "file:///pkg/missing.goal"}})
	if syms := s.documentSymbols(unknown); len(syms) != 0 {
		t.Errorf("unknown URI should yield no symbols, got %d", len(syms))
	}
}
