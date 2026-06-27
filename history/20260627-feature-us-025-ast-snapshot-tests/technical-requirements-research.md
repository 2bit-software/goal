# Technical Requirements / Research — US-025

## Approach

- Add `ast.Sexpr(Node) string`: a reflection-driven, deterministic s-expression
  renderer in a new `internal/ast/dump.go`. It walks exported struct fields,
  skips `token.Pos` fields (so structure is pinned, not offsets), renders
  `token.Kind` via `String()`, names `FuncMod`/`ChanDir`, and omits
  zero/empty fields for compactness. Struct field order is fixed, so output is
  deterministic. reflect/strings/fmt are stdlib — keeps the zero-dependency rule.
- Snapshot test lives in `internal/parser` (package parser; parser imports ast):
  parse each representative `features/NN/examples/*.goal`, render via
  `ast.Sexpr`, compare to a committed golden under
  `internal/parser/testdata/snapshots/<name>.sexpr`. A `-update-snapshots` flag
  regenerates the goldens; `go test ./...` (no flag) compares.

## Representative inputs (one per goal construct)

- enum: features/01-enums/examples/status.goal
- match: features/02-match/examples/status_rest.goal
- question-prop: features/05-question-prop/examples/qprop_bare.goal
- error-e (closed Result): features/06-error-e/examples/qclosed_match.goal
- implements: features/07-implements/examples/value_recv.goal
- defaults: features/08-no-zero-value/examples/defaults_primitives.goal
- assert: features/10-assert/examples/message.goal
- doctests: features/11-doctests/examples/add.goal
- derive-convert: features/12-derive-convert/examples/from_storage.goal

## Notes

- The parser already parses 100% of the corpus (US-024), so every chosen input
  parses with zero errors.
- grammar.js remains a manual cross-check reference (per prd notes); this
  snapshot suite is the loop-ready structural gate.
