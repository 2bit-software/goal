# Audit: AI-Consumer Readiness — US-027

## Findings

### MINOR — Fact data shapes are defined by precedent, not restated in the spec
The spec describes facts behaviorally (FR-1..FR-5). The concrete field shapes
(FuncSig mode/T/E/arity/ends-in-error; Enum variants + membership sets; Field
name/type; ConvEntry name/fallible; Method per receiver) are fixed by the
existing `internal/analyze` types, which the implementation mirrors. An AI
implementer has an unambiguous reference (analyze.go) and the parsed AST node set
(ast.go, goal_decl.go). No guessing required.

### MINOR — Result/Option detection rule
Detecting mode from the AST result list: a single unnamed result typed
`Result[T,E]` (IndexExpr/IndexListExpr with head Ident "Result") -> open-E when
E == "error" else closed-E; `Option[T]` -> Option. Arity/ends-in-error then take
the lowered shape (Result -> 2/true, Option & closed -> 1/false), matching
`analyze.analyzeSig`. This is fully specified by the analyze precedent.

## AI-Consumer Verdict

Implementable without clarifying questions. Acceptance criteria are concrete
enough to write assertions from (compare specific resolved symbols between the
two resolvers; assert the comma-field resolves to the correct field count/types).

## Assumptions
- The test compares specific named symbols (not whole-map equality), so
  incidental differences (e.g. whether methods also appear in FuncSignatures) do
  not cause false failures.
- `sema` does not import `analyze`; the parity test imports both plus `parser`.
