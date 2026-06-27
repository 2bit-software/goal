# Emit the Go Subset from AST — Business Specification

## Overview

The AST front-end's Go backend turns a parsed goal file into Go source text. The
seed emitter from US-026 covered only the ordinary-Go nodes a single tiny fixture
exercised. This feature completes the emitter so it covers the full ordinary-Go
subset that goal source can use, formatting the result once via the Formatter.
Goal-specific constructs (enums, match, `?`, construction, spread, assert, and
from/derive) remain the responsibility of later lowering stories and continue to
report a descriptive unsupported error rather than producing wrong output.

## Functional Requirements

### FR-1: Emit the full ordinary-Go subset
The backend SHALL emit valid Go source text for every ordinary-Go construct that
goal source can contain, including expression `switch` statements with their
`case` and `default` clauses — which the seed emitter did not handle.

### FR-2: Format once via the Formatter
The emitted Go SHALL be normalized exactly once through the Formatter so the
final output is canonical gofmt layout.

### FR-3: Behavioral conformance
A goal file using only ordinary-Go constructs SHALL transpile through the backend
and the generated Go SHALL build and vet cleanly in an isolated module.

### FR-4: Goal constructs still gated
Any goal-specific construct SHALL still produce a descriptive, non-crashing
unsupported error (lowered by later stories), never silent or wrong output.

## Acceptance Criteria

- [ ] A goal source file containing an expression `switch` (with `case` and
      `default` clauses) transpiles through the backend to valid Go.
- [ ] A goal source file using only ordinary-Go constructs (functions, control
      flow including switch, struct/map/slice composites, defer, multi-return,
      type/const/var declarations) transpiles through the backend and the
      generated Go builds and vets cleanly in an isolated temp module.
- [ ] The backend never crashes; a goal-specific construct yields a descriptive
      unsupported-construct error.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

No direct end-user surface. The backend is exercised through the transpile engine
behind the existing `--engine=ast` driver flag and through the corpus behavioral
runner in tests.

## Error Handling

An ordinary-Go construct must always emit. A goal-specific or not-yet-supported
construct returns a descriptive error naming the unsupported node; the transpile
fails cleanly with that message rather than emitting malformed Go.

## Out of Scope

- Lowering of goal-specific constructs (enums, match, `?`, construction, spread,
  assert, from/derive) — later stories US-033+.
- Statement forms goal's grammar does not produce (labeled statements, channel
  send statements, select statements, type switches, trailing variadic call
  spread).
- Making the AST engine the default; it remains behind `--engine=ast`.

## Open Questions

None — the gap (expression switch emission) and the verification path (corpus
behavioral tier) are both well-defined by the existing codebase.
