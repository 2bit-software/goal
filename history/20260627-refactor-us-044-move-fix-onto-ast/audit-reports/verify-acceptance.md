# Verify — Acceptance

US-044 acceptance criteria:

1. "internal/fix consumes the AST instead of re-lexing, and emits the same
   rewrites." — MET. fix.File parses each pass with `parser.ParseFile` and
   resolves facts with `sema.Resolve`; candidate detection is structural over
   AST nodes. `grep` confirms no `scan.Lex`, `analyze.Build`,
   `analyze.FuncSpans`, `analyze.SigAt`, `scan.ScanFuncs`, or `scan.Token`
   usage remains in internal/fix. Edits are still minimal `scan.Replacement`
   byte splices (a pure string utility), so untouched code/comments are
   preserved verbatim.

2. "The existing fix test suite passes unchanged in observable behavior." —
   MET. internal/fix/fix_test.go is byte-for-byte unchanged (no git diff) and
   `go test ./internal/fix` passes, covering tuple→Result conversion,
   Result/Option propagation collapse, init-guard collapse, comment/else/
   wrapped-error/multi-value refusals, switch-over-enum suggestion, and
   idempotence. internal/lsp (which compares against fix.File output) also
   passes.

Project verifyCommands: `go build ./...`, `go vet ./...`,
`go test ./... -count=1` — all green.
