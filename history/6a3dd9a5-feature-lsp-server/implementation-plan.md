# Implementation Plan — Goal LSP M1 (diagnostics)

## Verified reuse surface (read from source, not inferred)

`internal/check/check.go`:
- `func Analyze(src string) ([]Diagnostic, error)` (:148) — build tables + run all checks on one buffer.
- `type Diagnostic struct { Pos int; Severity Severity; Feature, Code, Message string }` (:72).
- `Severity`: `Error = 0`, `Warning = 1` (:53).
- `func OffsetToPosition(src string, off int) Position` → `{Line, Col}` 1-based (:88).
- `Run`/`Analyze` accumulate **all** diagnostics, sorted by `Pos`; return `error` only on internal bug.

This is the entire dependency M1 needs from the compiler. No `project.Package`, no `go/types`.

## Components & build order

### 1. `internal/lsp/` — server package (stdlib only)

Files:
- `jsonrpc.go` — stdio transport: read `Content-Length` + `\r\n\r\n` framed messages from a
  `bufio.Reader`; write framed messages to a mutex-guarded writer, flushing each. Decode into
  `{jsonrpc, id, method, params}`. Helpers: `readMessage`, `writeResponse`, `writeNotification`.
- `protocol.go` — the ~8 LSP structs used: `InitializeResult`, `ServerCapabilities`,
  `TextDocumentItem`, `VersionedTextDocumentIdentifier`, `Position`, `Range`, `Diagnostic`,
  `PublishDiagnosticsParams`, plus param structs for didOpen/didChange/didClose.
- `server.go` — `Server` with a document store (`map[uri]doc{text string, version int}` +
  `sync.Mutex`), the dispatch `switch` on method, and `Run(in io.Reader, out io.Writer) error`.
  - `initialize` → `{capabilities:{textDocumentSync:1}, serverInfo}`; `initialized` noop;
    `shutdown` → null; `exit` → return.
  - `didOpen`/`didChange` → store buffer, schedule debounced compile.
  - `didClose` → drop buffer, publish empty diagnostics.
  - unknown request → `-32601`; unknown notification → ignore. Logs → stderr only.
- `diagnostics.go` — `compile(uri, text, version)`: call `check.Analyze(text)`; for each
  `check.Diagnostic`, `OffsetToPosition` then map 1-based→0-based; build LSP `Diagnostic`
  (`severity`: Error→1, Warning→2; `source:"goal"`; `code`, `message`). Range: start at the
  offset; **end = end of the offending line** (M1 default, since byte offset has no length —
  see Decision below). Drop result if a newer `version` arrived; else `publishDiagnostics`.
- `debounce.go` — per-URI ~200ms debounce (a `map[uri]*time.Timer` guarded by the mutex);
  compile runs in a goroutine.

### 2. `cmd/goal/main.go` — `lsp` subcommand

Add an `lsp` subcommand that constructs an `lsp.Server` and calls `Run(os.Stdin, os.Stdout)`.
Wire into the existing subcommand dispatch alongside `build`/`run`/`check`. No flags for M1.

### 3. `editors/vscode/` — language client

- `package.json`: add `"main": "./dist/extension.js"`, `"activationEvents": ["onLanguage:goal"]`,
  runtime dep `vscode-languageclient: ^9`, dev deps `esbuild` + `@types/vscode` + `typescript`,
  scripts `build`/`watch` (esbuild) and update `package`/`install-local` to build first.
  Add a `goal.lsp.path` configuration (default `"goal"`).
- `src/extension.ts`: spawn `<goal.lsp.path> lsp` over `TransportKind.stdio`, documentSelector
  `{ language: 'goal' }`, start a `LanguageClient`; `deactivate` stops it.
- `esbuild.mjs` (or inline npm script): bundle `src/extension.ts` → `dist/extension.js`,
  `--external:vscode --platform=node --format=cjs`.
- Update `.vscodeignore` to ship `dist/` but not `src/`, `node_modules`, esbuild config.
- Keep existing grammar + language contributions untouched.

## Decisions resolved from audit

- **Diagnostic range width**: byte offset has no token length. M1 squiggles from the offset to
  the **end of that line** (clamp). Good enough to locate; a later milestone can use precise
  token spans from the lexer.
- **Server packaging**: `goal lsp` subcommand (no separate binary).
- **Scope**: lexical checks only (`check.Analyze` on the single open buffer).

## Test plan

- `internal/lsp/jsonrpc_test.go`: frame a message, read it back; assert `Content-Length` and body.
- `internal/lsp/diagnostics_test.go`: map a synthetic `check.Diagnostic` → LSP `Diagnostic`;
  assert 0-based range and severity. Plus an integration test that runs `check.Analyze` on a
  known-invalid buffer (non-exhaustive match) and asserts ≥1 diagnostic with expected code.
- `internal/lsp/server_test.go`: drive `Server.Run` over `io.Pipe` with scripted
  `initialize`→`didOpen`(invalid)→expect a `publishDiagnostics` containing the expected range.
- Manual: install extension, open an invalid `.goal`, see the squiggle clear when fixed.
- Gate: `task test` (go) + `npm test` (extension grammar) both pass; `go vet` clean.

## Traceability

| FR | Covered by |
|----|-----------|
| FR-001/005 | didOpen/didChange → compile → publishDiagnostics; sync mode Full |
| FR-002 | OffsetToPosition on original buffer text |
| FR-003 | Severity Error→1 / Warning→2 mapping |
| FR-004 | Diagnostic.message/code/source="goal"/feature |
| FR-006 | didClose + empty publish; valid file publishes `[]` |
| FR-007 | check.Analyze accumulates all diagnostics |
| FR-008 | goal.lsp.path defaults to `goal` on PATH |
