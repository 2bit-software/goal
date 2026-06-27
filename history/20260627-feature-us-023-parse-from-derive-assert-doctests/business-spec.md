# US-023 Parse from/derive, assert, doctests — Business Specification

## Overview

The goal front-end's hand-written parser already covers the Go subset plus
goal's enum/sealed/implements/match/construction surface. This feature adds the
last remaining goal-specific source constructs so the grammar is complete:
`from`/`derive` function declarations, `assert` statements, and `///` doctest
comments. With these parsed, an ordinary goal program of the kind shipped in the
`features/10-assert`, `features/11-doctests`, and `features/12-derive-convert`
examples produces a structured AST rather than parse errors or silently dropped
trivia.

## Functional Requirements

### FR-1: from/derive function declarations
The parser SHALL recognize `from func` and `derive func` declarations and record
which modifier (if any) prefixed the function, in both:
- bodied form, e.g. `from func parseUUID(s string) (UUID, error) { ... }`, and
- bodyless form, e.g. `derive func toIDs(g Group) IDList` (signature only, no
  `{ }`).
An ordinary `func` with no modifier SHALL remain unmodified.

### FR-2: assert statements
The parser SHALL recognize `assert` statements in both forms:
- bare: `assert <cond>` (e.g. `assert amount > 0`).
- printf-message: `assert <cond>, <format>, <args...>`
  (e.g. `assert age >= 0, "age must be non-negative, got %d", age`).
Only the FIRST top-level comma after the condition SHALL separate the condition
from the message; commas nested inside the condition (e.g. inside a call like
`clamp(lo, hi, n)`) SHALL NOT split it. A condition containing `%` (e.g.
`n%2 == 0`) SHALL parse as an ordinary expression, not a format string.

### FR-3: /// doctest comments as structured nodes
The parser SHALL capture a run of consecutive `///` doc-comment lines preceding a
function as a structured node attached to that function, rather than discarding
them as trivia. Within that node, each `>>>` line SHALL be captured as a doctest
example whose following non-`>>>` lines are its expected output. A doc-comment
run with no `>>>` line SHALL still be captured (as documentation with zero
doctests). Ordinary `//` comments SHALL continue to be ignored.

## Acceptance Criteria

- [ ] Parsing `features/12-derive-convert/examples/from_storage.goal` yields a
      `from func parseUUID` declaration carrying the from-modifier and a bodyless
      `derive func fromStorage` declaration carrying the derive-modifier and no body.
- [ ] Parsing `features/12-derive-convert/examples/slice.goal` yields a bodyless
      `derive func toIDs` with the derive-modifier.
- [ ] Parsing `features/12-derive-convert/examples/to_storage.goal` yields a
      bodied `from func` and a bodied `derive func`, each with the right modifier.
- [ ] Parsing `features/10-assert/examples/bank.goal` yields a bare assert
      statement with a condition and no message.
- [ ] Parsing `features/10-assert/examples/message.goal` yields a printf-message
      assert with a condition, a format-string message, and one argument.
- [ ] Parsing `features/10-assert/examples/multiple.goal` yields a bare assert, a
      message assert whose condition contains a call with internal commas (the
      message split fires only on the top-level comma), and a bare assert whose
      condition contains `%`.
- [ ] Parsing `features/11-doctests/examples/add.goal` yields a function whose
      attached doc node contains one doctest (`add(2, 3)` → `5`).
- [ ] Parsing `features/11-doctests/examples/multi.goal` yields a function whose
      doc node contains two doctests.
- [ ] Parsing `features/11-doctests/examples/mixed.goal` yields one function with
      a doc node containing zero doctests and one with a doc node containing one.
- [ ] The new AST nodes are traversed correctly by `Walk` (no node skipped, none
      visited twice).
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

None directly user-facing. The interaction is programmatic: a caller invokes the
parser on goal source and inspects the resulting AST. The behavior is observed
through the parser's unit tests.

## Error Handling

Consistent with the existing parser: a malformed construct records a positioned
parse error and the parser still advances (every loop makes progress), returning
a non-nil partial `*ast.File`. Well-formed inputs return a nil error.

## Out of Scope

- Lowering / Go code emission for assert, from/derive, or doctests (later
  backend stories US-038/US-039/US-040).
- Semantic validation (derive registry resolution, assert-condition typing,
  doctest execution).
- `goal fmt` comment-fidelity round-tripping (US-045).

## Open Questions

None. All three constructs are demonstrated by committed example inputs, and the
target AST shapes are determined by the existing node conventions.
