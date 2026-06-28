package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goal/internal/project"
	"goal/internal/sema"
)

const serverVersion = "0.1.0"

// fullSync is the textDocumentSync mode where every change carries the whole
// document, so the server replaces its buffer rather than applying deltas.
const fullSync = 1

// doc is the latest known text of an open document and its edit revision.
type doc struct {
	text    string
	version int
}

// fileSrc is one on-disk .goal file: its cleaned path and contents.
type fileSrc struct {
	path string
	src  string
}

// dirReader lists and reads the .goal files directly in a directory. It is injected so the
// server can be driven in tests without a real filesystem; osDirReader is the production
// implementation.
type dirReader func(dir string) ([]fileSrc, error)

// Server is a diagnostics-only goal language server. Construct it with
// NewServer and drive it with Run.
type Server struct {
	out   io.Writer
	outMu sync.Mutex

	mu       sync.Mutex
	docs     map[string]*doc
	timers   map[string]*time.Timer
	debounce time.Duration

	// analysisMu serializes the package-analysis path so that, when several files'
	// debounced compiles overlap, the last one to run reads the freshest buffers and
	// publishes last — the final state is correct even though sibling diagnostics are
	// published from another file's run.
	analysisMu sync.Mutex
	files      dirReader
	resolve    sema.DirResolver
}

// NewServer returns a server that publishes diagnostics to out (the client's
// stdin) and resolves a file's package from the real filesystem. Only framed
// protocol messages are ever written to out.
func NewServer(out io.Writer) *Server {
	return NewServerWithIO(out, osDirReader, sema.DefaultResolver)
}

// NewServerWithIO is NewServer with the filesystem and import-resolution seams injected, so
// tests can drive the server across a synthetic package without touching real disk or the
// go toolchain.
func NewServerWithIO(out io.Writer, files dirReader, resolve sema.DirResolver) *Server {
	return &Server{
		out:      out,
		docs:     map[string]*doc{},
		timers:   map[string]*time.Timer{},
		debounce: 200 * time.Millisecond,
		files:    files,
		resolve:  resolve,
	}
}

// osDirReader returns the .goal files directly in dir (non-recursive), each read from disk.
func osDirReader(dir string) ([]fileSrc, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []fileSrc
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), project.Ext) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, fileSrc{path: filepath.Clean(p), src: string(b)})
	}
	return out, nil
}

// Run serves the protocol over in until the client disconnects or sends exit.
func (s *Server) Run(in io.Reader) error {
	r := bufio.NewReader(in)
	for {
		msg, err := readMessage(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if s.handle(msg) {
			return nil
		}
	}
}

// handle routes one message and reports whether the server should stop.
func (s *Server) handle(m *rpcMessage) (stop bool) {
	switch m.Method {
	case "initialize":
		s.reply(m.ID, InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync:       fullSync,
				CodeActionProvider:     &CodeActionOptions{CodeActionKinds: []string{"source.fixAll", "source.fixAll.goal"}},
				DocumentSymbolProvider: true,
				SemanticTokensProvider: &SemanticTokensOptions{Legend: defaultSemanticLegend(), Full: true},
				DefinitionProvider:     true,
				HoverProvider:          true,
				ReferencesProvider:     true,
				RenameProvider:         true,
			},
			ServerInfo: ServerInfo{Name: "goal-lsp", Version: serverVersion},
		})
	case "shutdown":
		s.replyNull(m.ID)
	case "exit":
		return true
	case "textDocument/didOpen":
		s.didOpen(m.Params)
	case "textDocument/didChange":
		s.didChange(m.Params)
	case "textDocument/didClose":
		s.didClose(m.Params)
	case "textDocument/codeAction":
		s.reply(m.ID, s.codeActions(m.Params))
	case "textDocument/documentSymbol":
		s.reply(m.ID, s.documentSymbols(m.Params))
	case "textDocument/semanticTokens/full":
		s.reply(m.ID, s.semanticTokens(m.Params))
	case "textDocument/definition":
		s.reply(m.ID, s.definition(m.Params))
	case "textDocument/hover":
		s.reply(m.ID, s.hover(m.Params))
	case "textDocument/references":
		s.reply(m.ID, s.references(m.Params))
	case "textDocument/rename":
		s.reply(m.ID, s.rename(m.Params))
	case "initialized", "$/setTrace", "textDocument/didSave":
		// no-op notifications
	default:
		if m.ID != nil {
			s.replyError(m.ID, codeMethodNotFound, "method not found: "+m.Method)
		}
	}
	return false
}

// didOpen records a freshly opened document and queues its first analysis.
func (s *Server) didOpen(raw json.RawMessage) {
	var p didOpenParams
	if !s.decode(raw, &p, "didOpen") {
		return
	}
	s.upsert(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
	s.schedule(p.TextDocument.URI)
}

// didChange replaces a document's text with the edited buffer and re-queues analysis.
func (s *Server) didChange(raw json.RawMessage) {
	var p didChangeParams
	if !s.decode(raw, &p, "didChange") {
		return
	}
	if len(p.ContentChanges) == 0 {
		return
	}
	// Full sync: the last change holds the entire new document.
	text := p.ContentChanges[len(p.ContentChanges)-1].Text
	s.upsert(p.TextDocument.URI, text, p.TextDocument.Version)
	s.schedule(p.TextDocument.URI)
}

// didClose forgets a document and clears its diagnostics from the editor.
func (s *Server) didClose(raw json.RawMessage) {
	var p didCloseParams
	if !s.decode(raw, &p, "didClose") {
		return
	}
	uri := p.TextDocument.URI
	s.mu.Lock()
	delete(s.docs, uri)
	if t := s.timers[uri]; t != nil {
		t.Stop()
		delete(s.timers, uri)
	}
	s.mu.Unlock()
	s.publish(uri, 0, []Diagnostic{})
	// The closed file reverts to its on-disk contents in the package view, which can change
	// a still-open sibling's diagnostics; re-analyze the siblings so they don't go stale.
	if path, ok := uriToPath(uri); ok {
		for sibURI := range s.openFilesInDir(filepath.Dir(path)) {
			s.schedule(sibURI)
		}
	}
}

// buffer returns the current text and revision of an open document, or ok=false when the
// document is not open.
func (s *Server) buffer(uri string) (text string, version int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.docs[uri]
	if d == nil {
		return "", 0, false
	}
	return d.text, d.version, true
}

func (s *Server) upsert(uri, text string, version int) {
	s.mu.Lock()
	s.docs[uri] = &doc{text: text, version: version}
	s.mu.Unlock()
}

// schedule debounces analysis so rapid keystrokes coalesce into one compile. A
// non-positive debounce runs the analysis synchronously, which keeps tests
// deterministic.
func (s *Server) schedule(uri string) {
	if s.debounce <= 0 {
		s.compileLatest(uri)
		return
	}
	s.mu.Lock()
	if t := s.timers[uri]; t != nil {
		t.Stop()
	}
	s.timers[uri] = time.AfterFunc(s.debounce, func() { s.compileLatest(uri) })
	s.mu.Unlock()
}

// compileLatest analyzes the most recent buffer for uri.
func (s *Server) compileLatest(uri string) {
	s.mu.Lock()
	d := s.docs[uri]
	s.mu.Unlock()
	if d == nil {
		return
	}
	s.compile(uri, d.text, d.version)
}

func (s *Server) decode(raw json.RawMessage, v any, what string) bool {
	if err := json.Unmarshal(raw, v); err != nil {
		fmt.Fprintf(os.Stderr, "goal-lsp: %s params: %v\n", what, err)
		return false
	}
	return true
}

func (s *Server) reply(id *json.RawMessage, result any) {
	body, err := json.Marshal(result)
	if err != nil {
		s.logf("marshal result: %v", err)
		return
	}
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: body})
}

func (s *Server) replyNull(id *json.RawMessage) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: json.RawMessage("null")})
}

func (s *Server) replyError(id *json.RawMessage, code int, msg string) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *Server) notify(method string, params any) {
	s.write(rpcNotification{JSONRPC: "2.0", Method: method, Params: params})
}

func (s *Server) write(v any) {
	if err := writeMessage(s.out, &s.outMu, v); err != nil {
		s.logf("write: %v", err)
	}
}

func (s *Server) logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "goal-lsp: "+format+"\n", args...)
}
