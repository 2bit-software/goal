# US-005 Port token package to goal — Business Specification

## Overview

The `token` package defines the lexical vocabulary of the goal language: a
`Kind` for every lexeme, the `Pos{Offset, Line, Col}` source position, the
`Token` lexeme, and the lookup/classification helpers. It is the leaf of the
compiler dependency DAG (no internal dependencies). This story reimplements it
as goal source in the self-hosted compiler tree so that the foundation of the
front-end is goal-native, validated against the trusted Go implementation.

## Functional Requirements

### FR-1: Goal-native token package
The token package SHALL exist as goal source under `selfhost/token`, declaring
`package token` and exposing the same public surface as `internal/token`
(Kind and its constants, Kind methods String/IsLiteral/IsOperator/IsKeyword,
Lookup, IsKeyword, Pos and its methods, OffsetToPosition, Token).

### FR-2: Transpiles to compiling Go
The ported package SHALL transpile through the goal front-end (the US-002 smoke
gate) and the generated Go SHALL compile.

### FR-3: Behavioral equivalence
The existing `internal/token` tests SHALL pass when run against the transpiled
package, proving behavioral equivalence (including the iota const-block ranges
fixed in US-001).

## Acceptance Criteria

- [ ] `selfhost/token` holds the token package as goal source.
- [ ] The ported package transpiles via the US-002 smoke gate and the generated
      Go compiles.
- [ ] The existing token tests pass against the transpiled package.
- [ ] The project-wide gates (`task check`, `task build`) stay green.

## User Interactions

None directly. The package is consumed by the self-hosted compiler and by the
self-host verification harness (internal/selfhost). Verification is exercised
through `go test`.

## Error Handling

Verification failures (transpile error, non-compiling Go, or a failing ported
test) SHALL surface as a failing test in internal/selfhost with a message
identifying the package.

## Out of Scope

- Wiring selfhost/token into the selfhost `main` build path (US-012).
- Porting any other package (lexer, ast, ...).
- Changing the trusted internal/token implementation.

## Open Questions

None.
