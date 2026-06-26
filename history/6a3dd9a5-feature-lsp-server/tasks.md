# Tasks — Goal LSP M1 (diagnostics)

**Complexity**: Medium (~8-10 files across `internal/lsp`, `cmd/goal`, `editors/vscode`).
**Critical path**: T001 → T002 → T003 → T004 → T006 → T007 (server) ; T008-T010 (client) parallel.

## Server (Go, stdlib only)

- [ ] T001 Create `internal/lsp/protocol.go` — LSP wire structs (Position/Range/Diagnostic/
  PublishDiagnosticsParams/InitializeResult/ServerCapabilities + didOpen/didChange/didClose params).
- [ ] T002 [P] Create `internal/lsp/jsonrpc.go` — Content-Length framing read/write over stdio,
  mutex-guarded writer, JSON-RPC request/notification/response + `-32601` helper.
- [ ] T003 Create `internal/lsp/server.go` — `Server` (doc store, timers, dispatch switch,
  `Run(in io.Reader)`), depends on T001+T002.
- [ ] T004 Create `internal/lsp/diagnostics.go` — `compile()`: `check.Analyze` → map to LSP
  `Diagnostic` (1-based→0-based, severity, range to line end), version-staleness drop, publish.
- [ ] T005 Create `internal/lsp/debounce.go` — per-URI ~200ms timer scheduling `compile` in a goroutine.
- [ ] T006 Edit `cmd/goal/main.go` — add `case "lsp"` to `run`'s switch →
  `lsp.NewServer(os.Stdout).Run(os.Stdin)`; add `lsp` to `topUsage` and `guideCommands`.
- [ ] T007 Tests: `internal/lsp/jsonrpc_test.go` (frame round-trip), `diagnostics_test.go`
  (check.Diagnostic→LSP mapping + invalid-buffer integration via `check.Analyze`),
  `server_test.go` (scripted initialize→didOpen→expect publishDiagnostics over `io.Pipe`).

## VS Code client

- [ ] T008 [P] Edit `editors/vscode/package.json` — add `main`, `activationEvents`,
  `vscode-languageclient` dep, esbuild/typescript devDeps, build scripts, `goal.lsp.path` config.
- [ ] T009 [P] Create `editors/vscode/src/extension.ts` — LanguageClient spawning `goal lsp`
  over stdio for language id `goal`; plus `esbuild.mjs` build and `tsconfig.json`.
- [ ] T010 Edit `editors/vscode/.vscodeignore` — ship `dist/`, exclude `src/`/config/`node_modules`.

## Verify / wire-up

- [ ] T011 Run `task test` + `go vet ./...` (server + mapping tests green; build clean).
- [ ] T012 Build extension (esbuild) + `task build` (goal binary with `lsp`); manual smoke:
  install extension, open an invalid `.goal`, confirm squiggle appears and clears on fix.

## Spec traceability

All FR-001..008 are covered by T001-T006 (server behavior) and T008-T009 (client wiring);
T007/T011/T012 verify. No orphan tasks.
