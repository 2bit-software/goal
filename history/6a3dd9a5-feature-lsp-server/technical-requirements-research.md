# Technical Requirements & Decisions — Goal LSP (M1: diagnostics)

## Decisions (defaults chosen for milestone 1)

| Decision | Choice | Rationale |
|---|---|---|
| Server packaging | `goal lsp` subcommand of existing binary | No new artifact; reuses build/install |
| JSON-RPC / framing | Hand-rolled, stdlib only | Tiny surface; preserves zero-dep posture |
| Transport | stdio, `Content-Length` + CRLF framing | LSP default; simplest for VS Code |
| Sync mode | `textDocumentSync: 1` (Full) | Whole-buffer recompile; no rope/delta code |
| Diagnostics model | Push (`publishDiagnostics`) | Universal support; fire-and-forget |
| Check surface | Lexical only (`check.Analyze` on buffer) | In-memory, multi-error, no project context |
| Scheduling | 200ms debounce, goroutine, version-drop | Avoids per-keystroke compiles & stale results |
| Client lib | `vscode-languageclient` ^9 | Standard; handles framing/lifecycle |
| Client bundling | esbuild → `dist/extension.js` | Fastest, single-command bundle |
| Server discovery | `goal.lsp.path` setting, default `goal` on PATH | No hardcoded paths |

## Constraints

- **stdout purity**: nothing but framed LSP messages on stdout. All logging → stderr.
- **Zero Go deps**: server uses only stdlib (`encoding/json`, `bufio`, `os`, `sync`, `time`).
- **Position base**: convert Goal 1-based (line,col) → LSP 0-based; `end` exclusive.
- **CLI contract**: the existing `goal` Taskfile builds the binary; `goal lsp` rides along.

## Components to build

1. `internal/lsp/` — server package: transport (framing), JSON-RPC dispatch, document store,
   debounced compile, `check.Diagnostic` → LSP `Diagnostic` mapping.
2. `cmd/goal/main.go` — add `lsp` subcommand that runs the server over stdio.
3. `editors/vscode/` — add `main` entry, `vscode-languageclient` dep, `src/extension.ts`
   client, esbuild build, `goal.lsp.path` setting, `activationEvents`.

## Test strategy

- **Go unit tests** (`internal/lsp`): framing round-trip (encode/decode a message with
  `Content-Length`), and `check.Diagnostic`→LSP `Diagnostic` mapping (offset→0-based range,
  severity mapping). These need no editor.
- **Protocol smoke test**: drive the server over a pipe with a scripted `initialize` →
  `didOpen` (invalid buffer) → assert a `publishDiagnostics` with the expected range/code.
- **Manual editor verification**: install extension, open an invalid `.goal`, observe squiggle.

## Risks / notes

- Single-file lexical scope means cross-file-dependent guarantees won't fire in M1 (documented
  in business-spec Out of Scope). M2 adds type-backed/package diagnostics via `typecheck.Load`.
- UTF-16 column encoding nuance ignored for ASCII source (documented assumption).
