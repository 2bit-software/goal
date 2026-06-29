# US-007 Port ast package to goal — Business Specification

## Overview

The self-hosted compiler is being built leaf-to-root: each `internal/<pkg>` is
reimplemented as goal-native source under `selfhost/<pkg>` and validated against
the trusted Go implementation. token (US-005) and lexer (US-006) are ported.
This story ports the ast layer — the AST node definitions, the Walk traversal,
and the goal-specific node helpers — so the node layer above the token leaf is
goal-native.

## Functional Requirements

### FR-1: ast as goal source
The ast package SHALL exist as goal source under `selfhost/ast`, importing the
already-ported token package, covering node definitions, Walk, and the
goal-specific declaration/expression/statement node helpers.

### FR-2: Transpiles and compiles
The ported ast SHALL transpile through the goal front-end (US-002 smoke gate)
and the generated Go SHALL compile via `go build`.

### FR-3: Behavioral equivalence
The existing internal/ast tests SHALL pass against the transpiled package,
proving the ported ast is observably identical to internal/ast.

## Acceptance Criteria

- [ ] `selfhost/ast` holds the ast package as goal source importing the ported token package.
- [ ] It transpiles via the smoke gate and the generated Go compiles.
- [ ] The existing ast tests pass against the transpiled package.
- [ ] Project-wide gates green: `task check` and `task build`.

## User Interactions

No end-user surface. The deliverable is consumed by the self-host harness
(`internal/selfhost` port_test) and the fixpoint target.

## Error Handling

A transpile defect or behavioral divergence surfaces as a failing
`go build`/`go test` in the port_test gates; the story is not done until both
are green.

## Out of Scope

- The reflection-driven `dump.go` debug Sexpr renderer (dropped from the
  self-hosted build per prd notes; not on the compile path).
- Porting any adjacent package (parser, sema, etc.).
- Wiring ast into the selfhost main package (that is US-012).

## Open Questions

None — the port pattern is established by US-005/US-006 and the dependency shape
(ast -> token only) mirrors lexer.
