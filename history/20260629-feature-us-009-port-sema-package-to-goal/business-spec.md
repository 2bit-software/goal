# US-009 Port sema package to goal — Business Specification

## Overview

The self-host effort needs internal/sema (name resolution + semantic checking)
reimplemented as goal-native source so that this layer of the compiler's
dependency DAG is built by the goal front-end rather than the Go toolchain
directly. As with the earlier ports (token, lexer, ast, parser), the goal
source must transpile to Go that compiles and behaves identically to the
hand-written Go package.

## Functional Requirements

### FR-1: Goal-native sema source
A `selfhost/sema` directory holds the sema package as goal source, importing
the previously ported token, ast, and parser packages and passing the foreign
go/parser, go/format, go/types imports through unchanged.

### FR-2: Transpile + compile gate
The ported sema transpiles through the goal front-end and the generated Go
compiles, validated by the existing self-host smoke gate.

### FR-3: Behavioral equivalence
The existing internal/sema tests (those self-contained in the harness's
throwaway module) pass against the transpiled package.

## Acceptance Criteria

- [ ] selfhost/sema holds the sema package as goal source importing the ported
      token, ast, parser packages.
- [ ] It transpiles and the generated Go compiles (go/parser, go/format,
      go/types pass through as foreign imports).
- [ ] The existing sema tests pass against the transpiled package.
- [ ] task check and task build remain green.
- [ ] task fixpoint remains byte-identical with selfhost/sema present.

## User Interactions

None directly. Exercised through the self-host test harness
(`go test ./internal/selfhost`) and the fixpoint task target.

## Error Handling

If any ported source transpiles to non-compiling Go, the BuildTranspiled gate
fails. If behavior diverges, the copied white-box tests fail under BuildAndTest.

## Out of Scope

- Porting project/pipeline (US-010), backend (US-011), or typecheck (US-013).
- Any change to the behavior or API of internal/sema itself.
- Wiring sema into the self-hosted main (US-012).

## Open Questions

None — fourth port in an established, proven sequence.
