package lsptest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goal/internal/goalfmt"
	"goal/internal/lsp"
)

// newFrameReader wraps the server's raw output for framed-message reads via readFrame.
func newFrameReader(raw []byte) *bufio.Reader {
	return bufio.NewReader(bytes.NewReader(raw))
}

// writeTempPackage writes one .goal file into a fresh temp dir and returns the dir and the
// file:// URI of the file, so the server resolves it to a real on-disk package and runs the
// typed depth stage over it.
func writeTempPackage(t *testing.T, name, src string) (dir, uri string) {
	t.Helper()
	dir = t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return dir, "file://" + path
}

// depthErrorSrc lexes, parses, and passes goal's AST-stage checks, but is a Go type error
// (a string assigned to an int) that only the typed depth stage catches — exactly the class
// of finding the LSP must now publish so its diagnostics match `goal check`.
const depthErrorSrc = `package demo

func bad() int {
	var x int = "nope"
	return x
}
`

// The LSP publishes typed depth-stage diagnostics, not just lexical ones: a Go type error
// invisible to the AST stage surfaces as a [go-type-error] diagnostic (US-031 AC1).
func TestLSPPublishesDepthDiagnostics(t *testing.T) {
	_, uri := writeTempPackage(t, "bad.goal", depthErrorSrc)

	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`1`), "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": depthErrorSrc}},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}
	// The debounced compile runs the depth stage (transpile + go/types) after Run returns;
	// give it time to publish.
	time.Sleep(800 * time.Millisecond)

	diags := lastPublishedDiagnostics(t, out.snapshot(), uri)
	if !hasCode(diags, "go-type-error") {
		t.Fatalf("expected a [go-type-error] depth diagnostic for %q, got %+v", uri, diags)
	}
}

// A package that `goal check` resolves cleanly publishes no diagnostics — the depth stage
// contributes no false positives (US-031 AC1, no-false-positive half).
func TestLSPCleanPackageHasNoDiagnostics(t *testing.T) {
	const cleanSrc = `package demo

func ok() int {
	x := 1
	return x
}
`
	_, uri := writeTempPackage(t, "ok.goal", cleanSrc)

	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`1`), "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": cleanSrc}},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}
	time.Sleep(800 * time.Millisecond)

	diags := lastPublishedDiagnostics(t, out.snapshot(), uri)
	if len(diags) != 0 {
		t.Fatalf("clean package published diagnostics: %+v", diags)
	}
}

// textDocument/formatting is advertised in the initialize capabilities and round-trips an
// unformatted buffer to its canonical goalfmt form (US-031 AC2).
func TestLSPFormattingRoundTrips(t *testing.T) {
	const unformatted = "package demo\nfunc f() {\nx := 1\n_ = x\n}\n"
	const uri = "file:///nonexistent-dir/fmt.goal"

	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`1`), "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": unformatted}},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`2`), "method": "textDocument/formatting",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}

	// Capabilities advertise formatting.
	caps := initializeCapabilities(t, out.snapshot(), "1")
	if !caps.DocumentFormattingProvider {
		t.Errorf("initialize did not advertise documentFormattingProvider: %+v", caps)
	}

	edits := formattingResult(t, out.snapshot(), "2")
	if len(edits) != 1 {
		t.Fatalf("expected exactly one whole-document edit, got %d: %+v", len(edits), edits)
	}
	want, err := goalfmt.Source(unformatted)
	if err != nil {
		t.Fatalf("goalfmt.Source: %v", err)
	}
	if edits[0].NewText != want {
		t.Errorf("formatting NewText = %q, want canonical %q", edits[0].NewText, want)
	}
	if edits[0].NewText == unformatted {
		t.Errorf("formatting returned the unformatted buffer unchanged")
	}
}

// Formatting a syntax-error buffer returns no edits, leaving the buffer untouched
// (US-031 AC3) — goalfmt never guesses at malformed input.
func TestLSPFormattingSyntaxErrorNoEdits(t *testing.T) {
	const broken = "package demo\nfunc f( {\n"
	const uri = "file:///nonexistent-dir/broken.goal"

	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`1`), "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": broken}},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`2`), "method": "textDocument/formatting",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}
	edits := formattingResult(t, out.snapshot(), "2")
	if len(edits) != 0 {
		t.Fatalf("syntax-error buffer produced edits: %+v", edits)
	}
}

// Existing definition support still resolves a symbol to its declaration after the depth /
// formatting additions (US-031 AC3 smoke test).
func TestLSPDefinitionSmoke(t *testing.T) {
	const src = "package demo\n\nfunc helper() int { return 1 }\n\nfunc use() int { return helper() }\n"
	const uri = "file:///nonexistent-dir/def.goal"
	// Cursor over the call to `helper` on the last line (0-based line 4). "func use() int { return "
	// is 24 chars, so `helper` starts at character 24.
	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`1`), "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": src}},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "id": json.RawMessage(`2`), "method": "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 4, "character": 26},
		},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}
	loc := definitionResult(t, out.snapshot(), "2")
	if loc == nil {
		t.Fatalf("definition returned null for the call to helper")
	}
	// helper is declared on 0-based line 2.
	if loc.Range.Start.Line != 2 {
		t.Errorf("definition resolved to line %d, want the declaration on line 2", loc.Range.Start.Line)
	}
}

// --- decoding helpers ---

type wireDiagnostic struct {
	Code     string `json:"code"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
}

type wirePublish struct {
	URI         string           `json:"uri"`
	Diagnostics []wireDiagnostic `json:"diagnostics"`
}

func hasCode(diags []wireDiagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// lastPublishedDiagnostics returns the diagnostics of the last publishDiagnostics
// notification for uri (the freshest revision wins).
func lastPublishedDiagnostics(t *testing.T, raw []byte, uri string) []wireDiagnostic {
	t.Helper()
	r := newFrameReader(raw)
	var last []wireDiagnostic
	found := false
	for {
		body, err := readFrame(r)
		if err != nil {
			break
		}
		var note struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(body, &note) != nil || note.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p wirePublish
		if json.Unmarshal(note.Params, &p) != nil || p.URI != uri {
			continue
		}
		last = p.Diagnostics
		found = true
	}
	if !found {
		t.Fatalf("no publishDiagnostics for %q in server output:\n%s", uri, raw)
	}
	return last
}

type wireCapabilities struct {
	DocumentFormattingProvider bool `json:"documentFormattingProvider"`
	DefinitionProvider         bool `json:"definitionProvider"`
	HoverProvider              bool `json:"hoverProvider"`
	RenameProvider             bool `json:"renameProvider"`
}

func initializeCapabilities(t *testing.T, raw []byte, wantID string) wireCapabilities {
	t.Helper()
	var result struct {
		Capabilities wireCapabilities `json:"capabilities"`
	}
	if err := json.Unmarshal(resultForID(t, raw, wantID), &result); err != nil {
		t.Fatalf("decode initialize result: %v", err)
	}
	return result.Capabilities
}

func formattingResult(t *testing.T, raw []byte, wantID string) []lsp.TextEdit {
	t.Helper()
	body := resultForID(t, raw, wantID)
	if strings.TrimSpace(string(body)) == "null" {
		return nil
	}
	var edits []lsp.TextEdit
	if err := json.Unmarshal(body, &edits); err != nil {
		t.Fatalf("decode formatting result: %v\nresult: %s", err, body)
	}
	return edits
}

func definitionResult(t *testing.T, raw []byte, wantID string) *lsp.Location {
	t.Helper()
	body := resultForID(t, raw, wantID)
	if strings.TrimSpace(string(body)) == "null" {
		return nil
	}
	var loc lsp.Location
	if err := json.Unmarshal(body, &loc); err != nil {
		t.Fatalf("decode definition result: %v\nresult: %s", err, body)
	}
	return &loc
}

// resultForID scans framed responses for the one matching id and returns its raw Result.
func resultForID(t *testing.T, raw []byte, wantID string) json.RawMessage {
	t.Helper()
	r := newFrameReader(raw)
	for {
		body, err := readFrame(r)
		if err != nil {
			break
		}
		var resp wireResponse
		if json.Unmarshal(body, &resp) != nil || resp.ID == nil {
			continue
		}
		if strings.TrimSpace(string(*resp.ID)) != wantID {
			continue
		}
		return resp.Result
	}
	t.Fatalf("no response with id %s in server output:\n%s", wantID, raw)
	return nil
}
