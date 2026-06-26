package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

// A framed message survives a write then read with method and params intact.
func TestFramingRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex

	msg := rpcNotification{JSONRPC: "2.0", Method: "test/ping", Params: map[string]int{"n": 7}}
	if err := writeMessage(&buf, &mu, msg); err != nil {
		t.Fatalf("writeMessage: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("Content-Length: ")) {
		t.Fatalf("missing Content-Length header: %q", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("\r\n\r\n")) {
		t.Fatalf("missing CRLF header separator: %q", buf.String())
	}

	got, err := readMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if got.Method != "test/ping" {
		t.Fatalf("method = %q, want test/ping", got.Method)
	}
	var p struct {
		N int `json:"n"`
	}
	if err := json.Unmarshal(got.Params, &p); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if p.N != 7 {
		t.Fatalf("n = %d, want 7", p.N)
	}
}

// A message body without a Content-Length header is rejected.
func TestReadMessageMissingLength(t *testing.T) {
	r := bufio.NewReader(bytes.NewBufferString("\r\n{}"))
	if _, err := readMessage(r); err == nil {
		t.Fatal("expected error for missing Content-Length")
	}
}
