# Reuse Audit — Goal LSP M1

| Planned item | Existing code | Verdict | Action |
|---|---|---|---|
| Run checks on a buffer | `check.Analyze(src)` (`internal/check/check.go:148`) | DUPLICATE | Reuse directly — do NOT re-walk the AST in the LSP |
| Offset → line/col | `check.OffsetToPosition` (`:88`, 1-based) | DUPLICATE | Reuse; only adapt to 0-based for LSP |
| Severity model | `check.Severity` (Error/Warning, `:53`) | DUPLICATE | Map to LSP severity ints; don't invent a new enum |
| Diagnostic message/code/feature | `check.Diagnostic` fields (`:72`) | DUPLICATE | Pass through Message/Code; Source="goal" |
| Subcommand dispatch | `run()` switch in `cmd/goal/main.go:84` | EXTEND | Add `case "lsp"`; reuse existing pattern, add to topUsage + guideCommands |
| JSON-RPC / LSP framing | none in repo; stdlib only by policy | CREATE NEW | Hand-roll in `internal/lsp` (rejected `go.lsp.dev`, `glsp` — add deps) |
| VS Code extension shell | `editors/vscode/` (grammar-only) | EXTEND | Add `main`/client to the existing extension; keep grammar + language id |
| Extension test harness | `editors/vscode/test/tokenize.test.mjs` | RELATED | Pattern reuse for any new node test; grammar test stays as-is |

**Why create-new for JSON-RPC**: the project is zero-dependency (`go.mod` has no requires).
Importing an LSP framework would break that posture for a ~150-LOC framing layer. Hand-rolling
keeps stdlib-only. Documented in technical-requirements-research.md.

No planned item duplicates existing functionality except the check surface, which the plan
already routes through `check.Analyze` rather than reimplementing.
