# US-026 Add Backend and Formatter interfaces — Business Specification

## Overview

This story lays the pluggable back-end seams that the rest of the AST-front-end
rewrite (Phase 2) plugs into. It introduces a `Backend` interface and a
`Formatter` interface, and a new AST-driven transpile engine selectable through a
`--engine` flag on the driver. The existing token-splice engine stays the default
and is unchanged, so the product keeps shipping while the new engine is proven
piece by piece. The seam is proven end-to-end by transpiling a goal file that
uses no goal-specific constructs through the new engine and confirming the output
compiles.

## Functional Requirements

### FR-1: Backend interface
A `Backend` abstraction SHALL exist whose single operation emits transpile output
from a parsed goal file and resolved semantic information:
`Emit(*ast.File, *sema.Info) (Output, error)`, where `Output` is the existing
transpile output (generated Go plus optional doctest sidecar).

### FR-2: Formatter interface
A `Formatter` abstraction SHALL exist that formats generated source bytes
(`Format([]byte) ([]byte, error)`). A Go implementation SHALL format via the Go
standard formatter, decoupling codegen from the formatting step.

### FR-3: Semantic info type
A semantic-information type (`sema.Info`) SHALL exist so the `Backend` signature
can reference it. For this story it MAY be minimal; later stories populate it.

### FR-4: AST engine
A new AST-driven transpile engine SHALL exist that takes goal source, parses it to
an AST, builds semantic info, emits Go through a `Backend`, formats it through a
`Formatter`, and returns the transpile output — for the plain-Go (no
goal-specific constructs) subset.

### FR-5: Engine selection flag
The driver's build/run/check commands SHALL accept an `--engine` flag. The value
`splice` (the default, also when the flag is absent) selects today's engine; the
value `ast` selects the new AST engine. An unrecognized value SHALL produce a
clear usage error.

### FR-6: Default behavior unchanged
When `--engine` is absent or `splice`, the driver's behavior and output SHALL be
identical to today's.

## Acceptance Criteria

- [ ] A `Backend` interface with method `Emit(*ast.File, *sema.Info) (Output, error)` exists.
- [ ] A `Formatter` interface exists, with a Go-formatter implementation.
- [ ] The driver selects the engine via a flag (`--engine=ast` / `--engine=splice`).
- [ ] An unknown `--engine` value yields a usage error.
- [ ] A test transpiles a goal file using no goal-specific constructs through the
      new (AST) engine and the generated Go compiles and vets cleanly via the
      behavioral tier (temp-module `go build` + `go vet`).
- [ ] The project verify gates stay green: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`.

## User Interactions

- CLI: `goal build --engine=ast [path]`, `goal run --engine=ast [path]`,
  `goal check --engine=ast [path]`. Default (no flag) is the splice engine.

## Error Handling

- Unknown `--engine` value: a usage error naming the offending value.
- AST engine encountering a construct its emitter does not yet support (any
  goal-specific node, or a not-yet-covered Go form): a clear "unsupported" error
  identifying the construct — these are completed by later stories.

## Out of Scope

- Full Go-subset emission across the whole corpus (US-032).
- Lowering of goal-specific constructs: enums, match, `?`, Result/Option,
  defaults, assert, derive/from, doctests (US-033 .. US-040).
- Populating `sema.Info` with real symbol facts (US-027).
- Routing package-mode (multi-file, prelude) transpilation through the AST engine.
- Making the AST engine the default (US-042).

## Open Questions

- None blocking. All required seams (parser, behavioral tier, Output type) already
  exist and are exercised by passing tests.
