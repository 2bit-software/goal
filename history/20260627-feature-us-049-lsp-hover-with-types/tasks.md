# Tasks — US-049 LSP hover with types

- [x] T1 (foundation): Add hover protocol types to `internal/lsp/protocol.go` —
  `HoverParams`, `Hover`, `MarkupContent`, and `ServerCapabilities.HoverProvider`.
  No dependencies.

- [x] T2: Implement `internal/lsp/hover.go` — `hoverInfo`, `hoverIndex`,
  `buildHoverIndex`, `collectHoverSpans` (mirroring definition.go's refVisitor),
  `funcSignature`, `docText`, `resolveHover`, and the `(*Server).hover` handler.
  Depends on T1.

- [x] T3: Wire `internal/lsp/server.go` — advertise `HoverProvider: true` at
  initialize and route `textDocument/hover` to `s.hover`. Depends on T1, T2.

- [x] T4: Add `internal/lsp/hover_test.go` — Result-function signature (AC-2),
  function doc (AC-1), no-symbol/unparseable nulls, handler open/unknown URI, and
  the initialize capability assertion. Depends on T2, T3.

- [x] T5 (verify): Run prd verifyCommands (`go build ./...`, `go vet ./...`,
  `go test ./... -count=1`) and confirm green. Depends on T1–T4.
