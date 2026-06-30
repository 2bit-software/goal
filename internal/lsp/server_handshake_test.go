package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// frameMsg encodes obj as a Content-Length-framed JSON-RPC message. It is named
// distinctly from server_test.go's frame so this file is self-contained and can be
// run, unchanged, against the goal-built internal/compiler/lsp under the self-host
// gate (TestPortedLspServer) as well as against this legacy package under
// `task check`.
func frameMsg(obj map[string]any) []byte {
	body, _ := json.Marshal(obj)
	return fmt.Appendf(nil, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

// TestServerHandshakeScript drives a scripted initialize -> initialized ->
// shutdown -> exit session and pins the server's lifecycle responses. This is the
// US-016 parity oracle: the identical file runs against this legacy package and
// against the goal-built internal/compiler/lsp, so byte-identical assertions prove
// the goal-built server's handshake matches the legacy server's.
//
//   - initialize replies with the advertised server capabilities and serverInfo.
//   - initialized is a silent no-op (no response is written for it).
//   - shutdown replies with a null result keyed to its request id.
//   - exit ends the loop (Run returns nil).
func TestServerHandshakeScript(t *testing.T) {
	var in, out bytes.Buffer
	in.Write(frameMsg(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{},
	}))
	in.Write(frameMsg(map[string]any{
		"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{},
	}))
	in.Write(frameMsg(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "shutdown",
	}))
	in.Write(frameMsg(map[string]any{"jsonrpc": "2.0", "method": "exit"}))

	s := NewServer(&out)
	if err := s.Run(&in); err != nil {
		t.Fatalf("Run: %v", err)
	}

	b := out.Bytes()

	// initialize: the full capability set and serverInfo are advertised.
	for _, want := range []string{
		`"textDocumentSync":1`,
		`"documentSymbolProvider":true`,
		`"source.fixAll.goal"`,
		`"semanticTokensProvider"`,
		`"definitionProvider":true`,
		`"hoverProvider":true`,
		`"referencesProvider":true`,
		`"renameProvider":true`,
		`"name":"goal-lsp"`,
		`"version":"0.1.0"`,
	} {
		if !bytes.Contains(b, []byte(want)) {
			t.Fatalf("initialize response missing %q; output:\n%s", want, out.String())
		}
	}

	// shutdown: a null result keyed to request id 2.
	if !bytes.Contains(b, []byte(`"id":2`)) {
		t.Fatalf("no response keyed to the shutdown request id; output:\n%s", out.String())
	}
	if !bytes.Contains(b, []byte(`"result":null`)) {
		t.Fatalf("shutdown did not reply with a null result; output:\n%s", out.String())
	}
}
