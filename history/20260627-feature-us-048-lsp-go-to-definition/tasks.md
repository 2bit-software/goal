# Implementation Tasks

## Task 1: Protocol types
**Status**: completed
**Files**: internal/lsp/protocol.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2..FR-5 (response shape)
**Verify**: `go build ./internal/lsp`
Add `Location`, `DefinitionParams`, and `ServerCapabilities.DefinitionProvider`.

## Task 2: Definition handler + resolution
**Status**: completed
**Files**: internal/lsp/definition.go
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-3, FR-4, FR-5
**Verify**: `go build ./internal/lsp`
`definition` handler, `resolveDefinition`, `buildDeclIndex`, `collectRefs` (AST
walk), `offsetForPosition`.

## Task 3: Server wiring
**Status**: completed
**Files**: internal/lsp/server.go
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-1
**Verify**: `go build ./internal/lsp`
Advertise `DefinitionProvider: true`; route `textDocument/definition`.

## Task 4: Tests
**Status**: completed
**Files**: internal/lsp/definition_test.go
**Depends on**: Task 2, Task 3
**Spec coverage**: all ACs
**Verify**: `go test ./internal/lsp -run Definition -count=1`
Function-call resolution, enum-variant resolution, type-name resolution, null
fallbacks (whitespace, unknown URI, unparseable), and capability advertisement.

## Task 5: Full verify gates
**Status**: completed
**Files**: (none)
**Depends on**: Task 1-4
**Spec coverage**: regression
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`
