# Progress Log — Goal LSP M1

### T001-T005 — `internal/lsp` server package
- Status: Complete
- Files: `protocol.go` (wire structs), `jsonrpc.go` (Content-Length framing,
  request/notification/response), `server.go` (Server, dispatch, debounced
  scheduling, didOpen/didChange/didClose lifecycle), `diagnostics.go`
  (`check.Analyze` → LSP Diagnostic mapping, version-staleness drop).
- Notes: stdlib only — zero new Go deps. Debounce is configurable; 0 = synchronous
  (used by tests). stdout carries only framed messages; logs go to stderr.

### T006 — CLI subcommand
- Status: Complete
- Files: `cmd/goal/main.go` — `case "lsp"` → `lsp.NewServer(os.Stdout).Run(os.Stdin)`;
  added `lsp` to `guideCommands` + `topUsage`. Regenerated `AI-KNOWLEDGE-BOOTSTRAP.md`
  golden (the guide lists subcommands, so adding `lsp` changed it).

### T007 — Go tests
- Status: Complete
- Files: `internal/lsp/{jsonrpc,diagnostics,server}_test.go` — framing round-trip,
  offset→0-based range + severity mapping, invalid-buffer analysis, scripted
  initialize→didOpen→publishDiagnostics, initialize capabilities. All pass.

### T008-T010 — VS Code language client
- Status: Complete
- Files: `editors/vscode/package.json` (main, activationEvents, `vscode-languageclient`
  dep, esbuild/typescript devDeps, `goal.lsp.path`/`goal.lsp.enable` config, build
  scripts, engine bumped to ^1.82), `src/extension.ts` (LanguageClient over stdio),
  `esbuild.mjs`, `tsconfig.json` (strict), `.vscodeignore`/`.gitignore` updates.

### T011-T012 — Verification
- Status: Complete
- `go vet ./...` clean; `go test ./...` green (incl. `internal/lsp` and regenerated golden).
- `tsc --noEmit` clean; `esbuild` bundles `dist/extension.js`; grammar test 19/19 pass;
  `vsce package` produces `goal-lang-0.2.0.vsix`.
- **End-to-end**: drove the real `bin/goal lsp` over stdio (initialize → didOpen of a
  non-exhaustive match → after debounce) — received `publishDiagnostics` with
  `code:"non-exhaustive-match"`, `severity:1`, `source:"goal"`, correct 0-based range,
  and goal's exact message. Clean shutdown/exit (code 0, empty stderr).

## Decisions
- Lexical checks only (`check.Analyze` on the open buffer); depth/type-backed checks
  deferred to M2. Push diagnostics, Full sync, hand-rolled JSON-RPC (zero-dep posture).
- Diagnostic range spans from the finding to end-of-line (no token length from a byte offset).

## Blockers
- None.
