// Package lsptest black-box tests the goal language server through its public
// Run/framed-protocol seam. It lives outside internal/lsp because that package is
// generated from .goal sources (a hand-written _test.go there would trip the
// verify-generated drift gate), and because these tests only need the exported
// NewServer + Run surface.
package lsptest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"goal/internal/lsp"
)

// syncBuffer is an io.Writer safe for the concurrent write the server's debounced
// diagnostics timer may perform after Run returns.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) snapshot() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

// frame renders one Content-Length framed JSON-RPC message.
func frame(v any) string {
	body, _ := json.Marshal(v)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

// wire structs for decoding the codeAction response.
type wireResponse struct {
	ID     *json.RawMessage `json:"id"`
	Result json.RawMessage  `json:"result"`
}

type wireCodeAction struct {
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Edit  struct {
		DocumentChanges []struct {
			Edits []struct {
				NewText string `json:"newText"`
			} `json:"edits"`
		} `json:"documentChanges"`
	} `json:"edit"`
}

const nonExhaustiveSrc = `package demo

enum Color {
	Red
	Green
	Blue
}

func red()   {}
func green() {}

func handle(c Color) {
	match c {
		Color.Red => red()
		Color.Green => green()
	}
}
`

// The LSP offers a per-diagnostic quickfix code action carrying the same insertion the
// checker reports as a suggestedFix (US-030 AC2).
func TestLSPQuickfixForNonExhaustiveMatch(t *testing.T) {
	const uri = "file:///nonexistent-dir/color.goal"
	id1 := json.RawMessage(`1`)
	id2 := json.RawMessage(`2`)

	input := frame(map[string]any{
		"jsonrpc": "2.0", "id": &id1, "method": "initialize", "params": map[string]any{},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": nonExhaustiveSrc}},
	}) + frame(map[string]any{
		"jsonrpc": "2.0", "id": &id2, "method": "textDocument/codeAction",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 100, "character": 0},
			},
			"context": map[string]any{"only": []string{"quickfix"}},
		},
	})

	out := &syncBuffer{}
	srv := lsp.NewServer(out)
	if err := srv.Run(strings.NewReader(input)); err != nil {
		t.Fatalf("server Run: %v", err)
	}
	// Let any debounced diagnostics publish drain, so the snapshot is stable.
	time.Sleep(300 * time.Millisecond)

	actions := codeActionResult(t, out.snapshot(), "2")
	var quickfix *wireCodeAction
	for i := range actions {
		if actions[i].Kind == "quickfix" {
			quickfix = &actions[i]
		}
	}
	if quickfix == nil {
		t.Fatalf("no quickfix code action returned; actions=%+v", actions)
	}
	if len(quickfix.Edit.DocumentChanges) == 0 || len(quickfix.Edit.DocumentChanges[0].Edits) == 0 {
		t.Fatalf("quickfix carried no edit: %+v", quickfix)
	}
	newText := quickfix.Edit.DocumentChanges[0].Edits[0].NewText
	if !strings.Contains(newText, "Color.Blue") || !strings.Contains(newText, `panic("TODO")`) {
		t.Errorf("quickfix insertion is not the missing-variant repair: %q", newText)
	}
}

// codeActionResult scans framed responses for the one matching id and decodes its Result
// as a code-action array.
func codeActionResult(t *testing.T, raw []byte, wantID string) []wireCodeAction {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(raw))
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
		var actions []wireCodeAction
		if err := json.Unmarshal(resp.Result, &actions); err != nil {
			t.Fatalf("decode codeAction result: %v\nresult: %s", err, resp.Result)
		}
		return actions
	}
	t.Fatalf("no response with id %s found in server output:\n%s", wantID, raw)
	return nil
}

// readFrame reads one Content-Length framed message body.
func readFrame(r *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			length, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
		}
	}
	body := make([]byte, length)
	for read := 0; read < length; {
		n, err := r.Read(body[read:])
		if err != nil {
			return nil, err
		}
		read += n
	}
	return body, nil
}
