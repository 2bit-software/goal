# US-006 Port lexer package to goal — Business Specification

## Overview

The goal compiler is being self-hosted leaf-to-root up its dependency DAG. The
leaf, `token`, was ported to goal source in US-005. This story ports the next
layer, the `lexer` (tokenizer), to goal-native source so that turning source
text into a token stream is itself written in goal.

## Functional Requirements

### FR-1: Lexer exists as goal source
The lexer is available as goal source under `selfhost/lexer`, importing the
already-ported `token` package, and is behaviorally identical to the existing
Go lexer.

### FR-2: Transpiles and compiles
The goal lexer transpiles through the goal front-end and the generated Go
compiles, with `unicode` and `unicode/utf8` passing through as foreign imports
and the in-module `token` dependency resolving against the ported token package.

### FR-3: Behavioral equivalence proven by existing tests
The existing `internal/lexer` white-box tests pass against the transpiled
package output.

## Acceptance Criteria

- [ ] `selfhost/lexer` holds the lexer as goal source importing the ported token package.
- [ ] It transpiles and the generated Go compiles (unicode, unicode/utf8 pass through).
- [ ] The existing lexer tests pass against the transpiled package.

## User Interactions

None directly user-facing. The verification surface is the `internal/selfhost`
port test plus the project-wide `task check` / `task build` / `task fixpoint`.

## Error Handling

The port test fails with a package-identified error if transpilation fails, the
generated Go is invalid, the temp module fails to build, or the behavioral tests
fail.

## Out of Scope

- Wiring the ported lexer into a goal-written compiler main (that is US-012).
- Removing or altering the live Go `internal/lexer`.
- Porting any package above lexer (ast, parser, etc.).

## Open Questions

None.
