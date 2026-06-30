package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// This file is the US-017 query-feature parity oracle. It exercises hover,
// go-to-definition, find-references, rename, semantic tokens, and the
// diagnostics publish path entirely through the package's public/white-box
// surface, deliberately avoiding the two drifted surfaces that block the legacy
// feature *_test.go files from running against the goal-built package:
//   - it never names a sema severity constant (the goal-built sema models
//     Severity as a sealed interface, not the legacy comparable consts), and
//   - it never reads the unexported symKey.enum field (renamed enumName in the
//     port because `enum` is a goal reserved word).
//
// Every assertion is DERIVED from the fixture source (offsets resolved at
// runtime), so the identical file holds on BOTH this legacy package (under
// `task check`) and the transpiled internal/compiler/lsp package (fed to
// internal/selfhost TestPortedLspFeatures) — that two-package agreement is the
// parity proof for the AC-2 "matches the legacy responses" requirement. Helper
// names carry the fp* prefix so this file is self-contained and does not clash
// with the shared helpers in the other lsp test files.

// fpSrc declares an enum with two variants, a function called from two sites,
// and a type-position use of the enum — enough to exercise every query feature.
const fpSrc = `package p

enum Color {
	Red
	Green
}

func pick() Color {
	return Color.Red
}

func describe(c Color) string {
	_ = pick()
	_ = pick()
	return "ok"
}
`

// fpBadSrc is a non-exhaustive match: the invalid program used to drive the
// diagnostics publish path.
const fpBadSrc = `package p

enum Light {
	Red
	Green
}

func f(l Light) string {
	x := match l {
		Light.Red => "r"
	}
	return x
}
`

// fpOffsetOfNth returns the byte offset of the n-th (0-based) occurrence of sub.
func fpOffsetOfNth(t *testing.T, src, sub string, n int) int {
	t.Helper()
	idx, from := -1, 0
	for k := 0; k <= n; k++ {
		j := strings.Index(src[from:], sub)
		if j < 0 {
			t.Fatalf("occurrence %d of %q not found", n, sub)
		}
		idx = from + j
		from = idx + 1
	}
	return idx
}

// fpCursorAt converts a byte offset into the 0-based line/char the handlers key on.
func fpCursorAt(src string, off int) (line, char int) {
	for i := 0; i < off && i < len(src); i++ {
		if src[i] == '\n' {
			line++
			char = 0
		} else {
			char++
		}
	}
	return line, char
}

// fpFrame encodes obj as a Content-Length-framed JSON-RPC message.
func fpFrame(obj map[string]any) []byte {
	body, _ := json.Marshal(obj)
	return append([]byte("Content-Length: "), append([]byte(itoa(len(body))), append([]byte("\r\n\r\n"), body...)...)...)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// fpOpenedServer feeds initialize + didOpen(uri, src) and lets EOF end Run, so
// the document is resident in the returned server for direct handler queries.
func fpOpenedServer(t *testing.T, uri, src string) (*Server, *bytes.Buffer) {
	t.Helper()
	var in, out bytes.Buffer
	in.Write(fpFrame(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}))
	in.Write(fpFrame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": src}},
	}))
	s := NewServer(&out)
	s.debounce = 0 // analyze synchronously
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return s, &out
}

func fpRaw(t *testing.T, v map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	return b
}

// Hover over a call reference describes the called function's signature.
func TestParityHover(t *testing.T) {
	off := fpOffsetOfNth(t, fpSrc, "pick", 1) // first call site
	line, char := fpCursorAt(fpSrc, off)
	info, ok := resolveHover(fpSrc, line, char)
	if !ok {
		t.Fatal("hover did not resolve over the pick() call")
	}
	if !strings.Contains(info.signature, "func pick") {
		t.Fatalf("hover signature = %q, want it to mention `func pick`", info.signature)
	}
}

// Go-to-definition over an enum-variant reference resolves to the variant decl.
func TestParityDefinition(t *testing.T) {
	declOff := fpOffsetOfNth(t, fpSrc, "Red", 0) // the `Red` variant declaration
	refOff := fpOffsetOfNth(t, fpSrc, "Red", 1)  // the `Color.Red` reference in pick()
	line, char := fpCursorAt(fpSrc, refOff)
	got, ok := resolveDefinition(fpSrc, line, char)
	if !ok {
		t.Fatal("definition did not resolve over the Color.Red reference")
	}
	wl, wc := fpCursorAt(fpSrc, declOff)
	if got.Start.Line != wl || got.Start.Character != wc {
		t.Fatalf("definition start = {%d,%d}, want the variant decl at {%d,%d}", got.Start.Line, got.Start.Character, wl, wc)
	}
}

// Find-references lists every occurrence of the symbol; the includeDeclaration
// toggle adds or drops the declaration name.
func TestParityReferences(t *testing.T) {
	const uri = "file:///fp.goal"
	s, _ := fpOpenedServer(t, uri, fpSrc)
	off := fpOffsetOfNth(t, fpSrc, "pick", 1)
	line, char := fpCursorAt(fpSrc, off)

	withDecl := s.references(fpRaw(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": char},
		"context":      map[string]any{"includeDeclaration": true},
	}))
	if len(withDecl) != 3 { // decl + two call sites
		t.Fatalf("references (incl decl) = %d, want 3", len(withDecl))
	}
	noDecl := s.references(fpRaw(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": char},
		"context":      map[string]any{"includeDeclaration": false},
	}))
	if len(noDecl) != 2 { // two call sites only
		t.Fatalf("references (excl decl) = %d, want 2", len(noDecl))
	}
}

// Rename returns a single version-pinned document edit touching every occurrence.
func TestParityRename(t *testing.T) {
	const uri = "file:///fp.goal"
	s, _ := fpOpenedServer(t, uri, fpSrc)
	off := fpOffsetOfNth(t, fpSrc, "pick", 1)
	line, char := fpCursorAt(fpSrc, off)

	we := s.rename(fpRaw(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": char},
		"newName":      "chosen",
	}))
	if we == nil {
		t.Fatal("rename returned nil for a valid symbol")
	}
	if len(we.DocumentChanges) != 1 {
		t.Fatalf("rename document changes = %d, want 1", len(we.DocumentChanges))
	}
	edits := we.DocumentChanges[0].Edits
	if len(edits) != 3 { // decl + two call sites
		t.Fatalf("rename edits = %d, want 3", len(edits))
	}
	for _, e := range edits {
		if e.NewText != "chosen" {
			t.Fatalf("rename edit NewText = %q, want \"chosen\"", e.NewText)
		}
	}
}

// Rename refuses an illegal identifier with a null result.
func TestParityRenameInvalid(t *testing.T) {
	const uri = "file:///fp.goal"
	s, _ := fpOpenedServer(t, uri, fpSrc)
	off := fpOffsetOfNth(t, fpSrc, "pick", 1)
	line, char := fpCursorAt(fpSrc, off)
	we := s.rename(fpRaw(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": char},
		"newName":      "1bad",
	}))
	if we != nil {
		t.Fatalf("rename to an illegal name returned %+v, want nil", we)
	}
}

// Semantic tokens classify the enum/variant/function/keyword roles from the AST.
func TestParitySemanticTokens(t *testing.T) {
	data := computeSemanticTokens(fpSrc)
	if len(data) == 0 || len(data)%5 != 0 {
		t.Fatalf("semantic token stream length = %d, want a non-empty multiple of 5", len(data))
	}
	seen := map[uint]bool{}
	for i := 3; i < len(data); i += 5 {
		seen[data[i]] = true // the tokenType column of each 5-tuple
	}
	for _, want := range []struct {
		role int
		name string
	}{
		{semEnum, "enum"},
		{semEnumMember, "enumMember"},
		{semFunction, "function"},
		{semKeyword, "keyword"},
	} {
		if !seen[uint(want.role)] {
			t.Fatalf("semantic tokens never classified a %s (role %d)", want.name, want.role)
		}
	}
}

// Opening an invalid document publishes diagnostics carrying the finding, with
// the goal-severity mapped to the protocol scale (Error -> 1). A non-file URI
// forces the single-file analysis fallback, so the test touches no filesystem.
func TestParityDiagnosticsPublish(t *testing.T) {
	const uri = "untitled:Untitled-1"
	_, out := fpOpenedServer(t, uri, fpBadSrc)

	found := false
	r := bufio.NewReader(out)
	for {
		m, err := readMessage(r)
		if err != nil {
			break
		}
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p PublishDiagnosticsParams
		if err := json.Unmarshal(m.Params, &p); err != nil {
			t.Fatalf("unmarshal diagnostics: %v", err)
		}
		if p.URI != uri || len(p.Diagnostics) == 0 {
			continue
		}
		d := p.Diagnostics[0]
		if d.Severity != 1 {
			t.Fatalf("diagnostic severity = %d, want 1 (Error)", d.Severity)
		}
		if d.Source != "goal" || d.Code == "" {
			t.Fatalf("diagnostic metadata = %+v, want source goal and a non-empty code", d)
		}
		found = true
	}
	if !found {
		t.Fatalf("no diagnostics published for %s; output:\n%s", uri, out.String())
	}
}
