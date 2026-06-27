# Plan Buildability Audit — US-048

## Findings

- Dependency order is valid: protocol types -> handler/resolution -> server
  routing -> tests. No forward references.
- Interface contracts agree: `rangeOf`, `check.OffsetToPosition`,
  `parser.ParseFile`, `ast.Walk`, and the `testServer` helpers all exist with
  the signatures the plan uses (verified by reading symbols.go,
  semantictokens.go, server_test.go, walk.go, ast nodes).
- File paths are concrete and non-conflicting (`definition.go`/`_test.go` are
  new in `internal/lsp`).
- The two-pass index-then-walk approach compiles incrementally; the index is
  built before refs so a reference to a later-declared symbol still resolves.
- No CRITICAL or MAJOR findings.

## Assumptions

- `reply` marshals a nil `*Location` as JSON `null` (confirmed: it
  `json.Marshal`s the result; a nil pointer encodes to `null`).
- Method/function name collisions are last-writer-wins in the index (not
  exercised by AC or corpus).
