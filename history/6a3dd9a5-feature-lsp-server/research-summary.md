# Research Summary — Goal LSP diagnostics

## Codebase: what we can reuse (exact surface)

- **Lexical check entry points** (in-memory, no disk):
  - `check.Analyze(src string) ([]check.Diagnostic, error)` — builds tables, runs all checks on one buffer.
  - `check.AnalyzePackage(srcs []string) ([][]check.Diagnostic, error)` — per-file over merged tables.
  - File: `internal/check/check.go:133-150`.
- **Diagnostic struct** (`internal/check/check.go:72-78`):
  - `Pos int` (byte offset into original `.goal` source), `Severity` (`Error=0`, `Warning=1`),
    `Feature string`, `Code string`, `Message string`.
- **Position conversion** (`internal/check/check.go:81-98`):
  - `check.OffsetToPosition(src string, off int) check.Position` → `{Line, Col}` **1-based**, pure string math.
- **Checks accumulate** all findings (not first-error), sorted by `Pos`. The check stage
  returns an `error` only on an internal bug; guarantee violations are `Diagnostic`s, never returned errors.
- **Positions map to ORIGINAL `.goal` source** — lexical checks run pre-lowering; offsets index the source the user sees. No `token.FileSet` needed for M1.
- **Module**: `module goal`, `go 1.26`, **zero external deps**. No existing JSON-RPC/LSP/server code.
- **Depth checks** (`internal/typecheck`: `Load`, `CheckImplements`, `CheckMustUse`, `CheckNoZeroValue`)
  need a `project.Package` + `go/types` (positions via `//line` directives). **Out of M1 scope** —
  more setup, requires multi-file project context.

## LSP: minimal diagnostics-only design

- **Required messages**: `initialize` (respond with capabilities), `initialized`, `shutdown`,
  `exit`; `textDocument/didOpen`, `didChange`, `didClose`; server→client
  `textDocument/publishDiagnostics`. Ignore unknown notifications; reply `-32601` to unknown requests.
- **Capabilities**: `textDocumentSync: 1` (Full) — each `didChange` carries the whole document; replace buffer and recompile. Simplest correct choice.
- **Push vs pull**: use **push** (`publishDiagnostics`) — universally supported, fire-and-forget. Pull (3.17) deferred.
- **Transport**: stdio with `Content-Length:` + `\r\n\r\n` framing. **stdout = framed messages only**; all logs → stderr. Flush writer after every message. Mutex-guard the writer (diagnostics may be sent from goroutines).
- **Diagnostic mapping**: LSP `Position` is **0-based** line AND character; Goal is 1-based →
  `lspLine = line-1`, `lspChar = col-1`. `end` is exclusive; with no token length, use a small
  range (start..start+width-of-construct, or a sensible default). ASCII assumption fine for M1
  (UTF-16 column nuance documented, not handled).
- **Scheduling**: compile whole buffer on didOpen/didChange, **debounce ~200ms**, run in goroutine,
  drop stale results by `version`, always publish (including empty `[]` to clear).

## Library decision

- **Server**: hand-roll JSON-RPC 2.0 over stdio with **stdlib only** (~150-250 LOC: framing +
  dispatch switch + ~8 structs). Matches the project's zero-dependency value. Rejected
  alternatives: `go.lsp.dev/protocol` (drags in zap), `tliron/glsp` (logging stack, "early release"),
  `x/tools` jsonrpc2 (internal, not importable).
- **Client**: `vscode-languageclient` ^9 (1 runtime dep) + **esbuild** (dev/build) to bundle a
  `dist/extension.js`. The current extension is grammar-only with no JS entry; add `main`,
  `activationEvents: onLanguage:goal`, ~30-line `extension.ts` spawning `goal lsp` over stdio.

## Recommended M1 architecture

`goal lsp` subcommand → stdlib stdio JSON-RPC server → on didOpen/didChange (debounced) call
`check.Analyze(buffer)` → map `check.Diagnostic` (offset→line/col via `OffsetToPosition`, 1-based→0-based)
to LSP `Diagnostic` → `publishDiagnostics`. Net new deps: server 0, client 1 + esbuild.
