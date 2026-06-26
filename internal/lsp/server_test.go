package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func frame(obj map[string]any) []byte {
	body, _ := json.Marshal(obj)
	return fmt.Appendf(nil, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

// Driving the server through initialize and an opened invalid document yields a
// publishDiagnostics notification carrying the finding for that document.
func TestServerPublishesDiagnosticsOnOpen(t *testing.T) {
	const uri = "file:///x.goal"
	var in, out bytes.Buffer

	in.Write(frame(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{},
	}))
	in.Write(frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": uri, "version": 1, "text": nonExhaustiveSrc,
			},
		},
	}))
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "method": "exit"}))

	s := NewServer(&out)
	s.debounce = 0 // analyze synchronously so the result is in `out` before Run returns
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	r := bufio.NewReader(&out)
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
		if p.URI == uri && len(p.Diagnostics) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("no diagnostics published for %s; output:\n%s", uri, out.String())
	}
}

// Initialize advertises full-document sync.
func TestServerInitializeCapabilities(t *testing.T) {
	var in, out bytes.Buffer
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}))
	in.Write(frame(map[string]any{"jsonrpc": "2.0", "method": "exit"}))

	s := NewServer(&out)
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"textDocumentSync":1`)) {
		t.Fatalf("initialize did not advertise full sync; output:\n%s", out.String())
	}
}
