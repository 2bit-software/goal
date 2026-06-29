# US-008 Port parser package to goal — Business Specification

## Overview

The self-host effort ports the goal compiler leaf-to-root onto goal itself. The
token, lexer, and ast packages are already ported (US-005/006/007). This story
ports internal/parser — the source-to-AST stage — to goal-native source under
selfhost/parser, and proves it transpiles to compiling Go and behaves identically
to the trusted package.

## Functional Requirements

### FR-1: Parser exists as goal source
selfhost/parser holds the parser as goal source, importing the already-ported
token, lexer, and ast packages (plus stdlib pass-through imports).

### FR-2: Transpiles and compiles
The ported parser transpiles through the goal front-end (US-002 smoke gate) and
the generated Go compiles, alongside its ported token/lexer/ast dependencies.

### FR-3: Behavioral equivalence
The existing parser test suite passes against the transpiled package, proving the
ported parser behaves identically to the trusted internal/parser.

## Acceptance Criteria

- [ ] selfhost/parser holds the parser as goal source importing the ported token,
      lexer, ast packages.
- [ ] It transpiles and the generated Go compiles (verified by the smoke gate over
      token+lexer+ast+parser).
- [ ] The existing parser tests pass against the transpiled package.
- [ ] Project gates stay green: `task check` and `task build`.

## User Interactions

None — this is internal compiler infrastructure verified through the test suite
(`go test ./internal/selfhost`).

## Error Handling

The harness reports a descriptive, package-identified error on any transpile
failure, invalid generated Go, build failure, or test failure.

## Out of Scope

- Porting any other package (parser's dependents sema/project/backend are later
  stories).
- Changing the live internal/parser package or its tests.
- Running the fixture-reading parser suites (goal_construct/decl/match/stmt_test)
  and the Sexpr snapshot suite inside the throwaway temp module — they depend on
  repo-relative `../../features` fixtures and the intentionally-dropped ast.Sexpr
  debug renderer, so they are not part of the self-contained behavioral gate.

## Open Questions

None.
