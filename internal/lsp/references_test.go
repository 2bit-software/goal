package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// refSrc declares an enum with two variants, a function called from two sites, and a struct
// whose field type references the enum — so references/rename are exercised across function
// calls, enum/variant references, and a type-position reference.
const refSrc = `package p

enum Color {
	Red
	Green
}

type Box struct {
	c Color
}

func pick() Color {
	return Color.Red
}

func describe(c Color) string {
	match c {
		Color.Red => "r"
		Color.Green => "g"
	}
	return label()
}

func label() string {
	return "x"
}

func run() string {
	return label()
}
`

// countAll returns the number of (possibly overlapping) occurrences of sub in src.
func countAll(src, sub string) int {
	n, off := 0, 0
	for {
		i := strings.Index(src[off:], sub)
		if i < 0 {
			return n
		}
		n++
		off += i + len(sub)
	}
}

// TestReferencesFunction lists every use of a function plus its declaration when requested.
func TestReferencesFunction(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, refSrc, 1)

	// Cursor on the `label()` call inside describe (the 1st reference, after the decl).
	callOff := offsetOfNth(t, refSrc, "label", 1)
	line, char := cursorAt(refSrc, callOff)

	raw, _ := json.Marshal(ReferenceParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: char},
		Context:      ReferenceContext{IncludeDeclaration: true},
	})
	locs := s.references(raw)
	// `label` appears 3 times: the declaration and two call sites.
	if got, want := len(locs), countAll(refSrc, "label"); got != want {
		t.Fatalf("references(label) = %d locations, want %d", got, want)
	}
}

// TestReferencesIncludeDeclarationToggle proves the declaration is included only when asked.
func TestReferencesIncludeDeclarationToggle(t *testing.T) {
	callOff := offsetOfNth(t, refSrc, "label", 1)
	line, char := cursorAt(refSrc, callOff)
	const uri = "file:///pkg/a.goal"

	mk := func(include bool) []Location {
		var out bytes.Buffer
		s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
		s.upsert(uri, refSrc, 1)
		raw, _ := json.Marshal(ReferenceParams{
			TextDocument: textDocumentIdentifier{URI: uri},
			Position:     Position{Line: line, Character: char},
			Context:      ReferenceContext{IncludeDeclaration: include},
		})
		return s.references(raw)
	}

	with := mk(true)
	without := mk(false)
	if len(with) != len(without)+1 {
		t.Fatalf("includeDeclaration toggle: with=%d without=%d, want with == without+1",
			len(with), len(without))
	}
}

// TestReferencesEnumVariant lists the variant tag at its declaration and every reference, and
// does NOT include the enum type or the sibling variant.
func TestReferencesEnumVariant(t *testing.T) {
	key, occ, ok := resolveOccurrences(lineCharOf(t, refSrc, "Red", 1))
	if !ok {
		t.Fatal("Red reference did not resolve")
	}
	if key.kind != symKindVariant || key.enum != "Color" || key.name != "Red" {
		t.Fatalf("Red key = %+v, want variant Color.Red", key)
	}
	// Red appears as: the variant declaration, the Color.Red in pick(), the Color.Red match arm.
	if got, want := len(occ), countAll(refSrc, "Red"); got != want {
		t.Fatalf("Red occurrences = %d, want %d", got, want)
	}
	for _, o := range occ {
		if o.key.name == "Green" || o.key.kind == symKindType {
			t.Fatalf("variant references leaked into a sibling/type: %+v", o.key)
		}
	}
}

// TestReferencesTypeName lists the enum type at its declaration and every type-position use.
func TestReferencesTypeName(t *testing.T) {
	_, occ, ok := resolveOccurrences(lineCharOf(t, refSrc, "Color", 0))
	if !ok {
		t.Fatal("Color did not resolve")
	}
	// Color: enum decl, `c Color` field in Box, pick() result, Color.Red in pick(), describe
	// param, two match-arm Color.* selectors.
	if got, want := len(occ), countAll(refSrc, "Color"); got != want {
		t.Fatalf("Color occurrences = %d, want %d", got, want)
	}
}

// TestRenameProducesEditsAtEveryReference is the acceptance test: renaming a symbol yields a
// text edit at every one of its occurrences (declaration + all references).
func TestRenameProducesEditsAtEveryReference(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, refSrc, 7)

	callOff := offsetOfNth(t, refSrc, "label", 1)
	line, char := cursorAt(refSrc, callOff)
	raw, _ := json.Marshal(RenameParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: char},
		NewName:      "emit",
	})
	we := s.rename(raw)
	if we == nil {
		t.Fatal("rename returned null")
	}
	if len(we.DocumentChanges) != 1 {
		t.Fatalf("rename produced %d document changes, want 1", len(we.DocumentChanges))
	}
	dc := we.DocumentChanges[0]
	if dc.TextDocument.URI != uri || dc.TextDocument.Version != 7 {
		t.Fatalf("rename document = %+v, want %s @ v7", dc.TextDocument, uri)
	}
	if got, want := len(dc.Edits), countAll(refSrc, "label"); got != want {
		t.Fatalf("rename(label) produced %d edits, want %d (decl + every reference)", got, want)
	}
	for _, e := range dc.Edits {
		if e.NewText != "emit" {
			t.Fatalf("edit new text = %q, want %q", e.NewText, "emit")
		}
	}

	// Applying the edits replaces every `label` occurrence with `emit`.
	if applied := applyEdits(refSrc, dc.Edits); strings.Contains(applied, "label") ||
		countAll(applied, "emit") != countAll(refSrc, "label") {
		t.Fatalf("applying rename edits did not replace every occurrence:\n%s", applied)
	}
}

// TestRenameInvalidName refuses a new name that is not a legal identifier.
func TestRenameInvalidName(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, refSrc, 1)

	callOff := offsetOfNth(t, refSrc, "label", 1)
	line, char := cursorAt(refSrc, callOff)
	for _, bad := range []string{"", "1abc", "has space", "a-b"} {
		raw, _ := json.Marshal(RenameParams{
			TextDocument: textDocumentIdentifier{URI: uri},
			Position:     Position{Line: line, Character: char},
			NewName:      bad,
		})
		if we := s.rename(raw); we != nil {
			t.Errorf("rename to %q should be refused (null), got %+v", bad, we)
		}
	}
}

// TestReferencesNoSymbolAndUnparseable yields null/empty for a blank position and bad source.
func TestReferencesNoSymbolAndUnparseable(t *testing.T) {
	if _, occ, ok := resolveOccurrences(refSrc, 1, 0); ok || occ != nil {
		t.Error("blank line should not resolve to occurrences")
	}
	if _, _, ok := resolveOccurrences("package p\n\nenum Broken {\n", 2, 5); ok {
		t.Error("unparseable source should not resolve")
	}
}

// TestReferenceRenameHandlersNullForUnknownURI proves both handlers null out an unknown URI.
func TestReferenceRenameHandlersNullForUnknownURI(t *testing.T) {
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))

	refRaw, _ := json.Marshal(ReferenceParams{
		TextDocument: textDocumentIdentifier{URI: "file:///pkg/missing.goal"},
		Position:     Position{Line: 13, Character: 0},
	})
	if locs := s.references(refRaw); locs != nil {
		t.Errorf("references on unknown URI = %+v, want null", locs)
	}
	renRaw, _ := json.Marshal(RenameParams{
		TextDocument: textDocumentIdentifier{URI: "file:///pkg/missing.goal"},
		Position:     Position{Line: 13, Character: 0},
		NewName:      "ok",
	})
	if we := s.rename(renRaw); we != nil {
		t.Errorf("rename on unknown URI = %+v, want null", we)
	}
}

// TestServerAdvertisesReferencesAndRename asserts initialize advertises both capabilities.
func TestServerAdvertisesReferencesAndRename(t *testing.T) {
	var in, out bytes.Buffer
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}))
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "method": "exit"}))

	s := NewServer(&out)
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}
	body := out.Bytes()
	if !bytes.Contains(body, []byte(`"referencesProvider":true`)) {
		t.Errorf("initialize did not advertise references; output:\n%s", out.String())
	}
	if !bytes.Contains(body, []byte(`"renameProvider":true`)) {
		t.Errorf("initialize did not advertise rename; output:\n%s", out.String())
	}
}

// lineCharOf returns the 0-based cursor for the n-th occurrence of sub in src.
func lineCharOf(t *testing.T, src, sub string, n int) (string, int, int) {
	t.Helper()
	off := offsetOfNth(t, src, sub, n)
	line, char := cursorAt(src, off)
	return src, line, char
}

// applyEdits applies non-overlapping text edits to src by byte offset, right-to-left so earlier
// edits' offsets stay valid. Edits are keyed by their start position, recovered from the range.
func applyEdits(src string, edits []TextEdit) string {
	type span struct {
		start, end int
		text       string
	}
	spans := make([]span, 0, len(edits))
	for _, e := range edits {
		start, _ := offsetForPosition(src, e.Range.Start.Line, e.Range.Start.Character)
		end, _ := offsetForPosition(src, e.Range.End.Line, e.Range.End.Character)
		spans = append(spans, span{start, end, e.NewText})
	}
	// Sort descending by start (simple insertion sort; edit sets are tiny).
	for i := 1; i < len(spans); i++ {
		for j := i; j > 0 && spans[j-1].start < spans[j].start; j-- {
			spans[j-1], spans[j] = spans[j], spans[j-1]
		}
	}
	out := src
	for _, sp := range spans {
		out = out[:sp.start] + sp.text + out[sp.end:]
	}
	return out
}
