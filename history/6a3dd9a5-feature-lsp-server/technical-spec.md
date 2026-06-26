# Technical Spec — Goal LSP M1

## Server package `internal/lsp`

### Wire structs (`protocol.go`)

```go
type Position struct { Line int `json:"line"`; Character int `json:"character"` } // 0-based
type Range struct { Start, End Position `json:"start" / "end"` }
type Diagnostic struct {
    Range    Range  `json:"range"`
    Severity int    `json:"severity"`        // 1=Error 2=Warning 3=Info 4=Hint
    Code     string `json:"code,omitempty"`
    Source   string `json:"source,omitempty"`
    Message  string `json:"message"`
}
type PublishDiagnosticsParams struct {
    URI         string       `json:"uri"`
    Version     int          `json:"version,omitempty"`
    Diagnostics []Diagnostic `json:"diagnostics"`
}
// + InitializeResult{Capabilities, ServerInfo}, ServerCapabilities{TextDocumentSync int},
//   TextDocumentItem, VersionedTextDocumentIdentifier, didOpen/didChange/didClose params.
```

### Transport (`jsonrpc.go`)

- `readMessage(r *bufio.Reader) (rawMessage, error)`: parse headers until blank line; read
  `Content-Length` bytes via `io.ReadFull`; `json.Unmarshal` into
  `struct{ JSONRPC string; ID *json.RawMessage; Method string; Params json.RawMessage }`.
- `writeMessage(w io.Writer, mu *sync.Mutex, v any)`: marshal; write
  `Content-Length: N\r\n\r\n` + body; flush. Mutex-guarded.
- Responses carry the request `ID`; notifications omit it. Errors use JSON-RPC `error` object
  (`-32601` method not found).

### Server (`server.go`)

```go
type doc struct { text string; version int }
type Server struct {
    mu     sync.Mutex
    docs   map[string]doc
    timers map[string]*time.Timer
    out    io.Writer
    outMu  sync.Mutex
}
func NewServer(out io.Writer) *Server
func (s *Server) Run(in io.Reader) error   // read loop over bufio.NewReader(in)
```

Dispatch (`switch method`): `initialize`→reply `{capabilities:{textDocumentSync:1},
serverInfo:{name:"goal-lsp",version}}`; `initialized`/`$/setTrace`→ignore; `shutdown`→reply
null; `exit`→return nil; `textDocument/didOpen`/`didChange`→`upsert`+`schedule`;
`textDocument/didClose`→delete + publish empty. Unknown request id→`-32601`; unknown
notification→ignore. All diagnostic/error logging via `fmt.Fprintln(os.Stderr, ...)`.

### Diagnostics (`diagnostics.go`)

```go
func (s *Server) compile(uri, text string, version int) {
    diags, err := check.Analyze(text)
    if err != nil { log to stderr; return }            // internal bug only
    out := make([]Diagnostic, 0, len(diags))
    for _, d := range diags {
        p := check.OffsetToPosition(text, d.Pos)        // 1-based
        start := Position{Line: p.Line - 1, Character: p.Col - 1}
        end := Position{Line: p.Line - 1, Character: lineEndCol(text, p.Line)} // M1 default
        sev := 1; if d.Severity == check.Warning { sev = 2 }
        out = append(out, Diagnostic{Range{start, end}, sev, d.Code, "goal", d.Message})
    }
    if s.stale(uri, version) { return }
    s.publish(uri, version, out)
}
```

`schedule(uri)`: reset a per-URI `time.Timer` (~200ms) that fires `compile` in a goroutine.
`stale`: compare stored version to the version this compile ran on.

## CLI integration (`cmd/goal/main.go`)

In `run`'s `switch cmd`, add:
```go
case "lsp":
    return lsp.NewServer(os.Stdout).Run(os.Stdin)
```
Add `lsp` to `topUsage` and the `guideCommands` registry (so `goal ai` documents it).
The server writes only framed messages to stdout; everything else to stderr — preserving
stdout purity required by the protocol.

## VS Code client (`editors/vscode`)

`src/extension.ts` (bundled by esbuild to `dist/extension.js`):
```ts
const cfg = workspace.getConfiguration('goal');
const server: ServerOptions = {
  command: cfg.get('lsp.path', 'goal'), args: ['lsp'], transport: TransportKind.stdio,
};
client = new LanguageClient('goal', 'Goal Language Server',
  server, { documentSelector: [{ language: 'goal' }] });
client.start();
```
`package.json`: `main`, `activationEvents:["onLanguage:goal"]`, dep `vscode-languageclient@^9`,
devDeps `esbuild`,`typescript`,`@types/vscode`,`@types/node`; `contributes.configuration` with
`goal.lsp.path` (default `"goal"`). Build script: `esbuild src/extension.ts --bundle
--outfile=dist/extension.js --external:vscode --format=cjs --platform=node`.

## Non-goals (M1)

Type-backed checks (typecheck/go-types), hover, completion, go-to-def, semantic tokens,
multi-file analysis, pull diagnostics, incremental sync, UTF-16 column handling for non-ASCII.
