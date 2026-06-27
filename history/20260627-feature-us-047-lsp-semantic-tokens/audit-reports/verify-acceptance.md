# Verify — Acceptance

## Acceptance criteria
1. The server advertises and serves `textDocument/semanticTokens` classified from
   the AST.
   - Advertise: `ServerCapabilities.SemanticTokensProvider` set in the initialize
     reply (server.go). `TestServerInitializeCapabilities` asserts the
     `"semanticTokensProvider"` capability appears.
   - Serve: `textDocument/semanticTokens/full` routed to `s.semanticTokens`
     (server.go); `TestSemanticTokensHandler` asserts non-empty tokens for an
     open doc and empty for an unknown URI.
   - From the AST: `astRoles` parses with `parser.ParseFile` and walks the AST to
     assign identifier roles. PASS.
2. A test asserts the classification of a sample containing an enum, a match, and
   a `?` expression.
   - `TestComputeSemanticTokensEnumMatchQuestion` asserts: enum name -> enum,
     variants -> enumMember, `match` -> keyword, `?` -> operator, function name ->
     function. PASS.

## Project gates (prd.json verifyCommands)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all 18 packages ok)

## Findings
No CRITICAL or MAJOR findings. Acceptance criteria met.

## Assumptions
- ASCII source: byte length == character length for token `length`.
- Unknown identifiers (no AST role) are intentionally not emitted, so builtins
  like `int` are not miscoloured; the editor grammar colours them.
- `full: true` only; range and delta requests are out of scope (documented).
