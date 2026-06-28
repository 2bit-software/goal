# Verify — Acceptance Coverage — US-032

Full suite: `go build ./...`, `go vet ./...`, `go test ./... -count=1` — all green.

## Acceptance criteria → evidence

| AC | Evidence | Result |
|----|----------|--------|
| Source with an expression `switch` (case + default) transpiles to valid Go | `internal/backend/backend_test.go` `TestASTEngineEmitsSwitch` — `backend.Transpile` then `go/format.Source` succeeds and output contains `switch`/`case`/`default` | PASS |
| Full ordinary-Go subset file builds + vets cleanly in an isolated module | `TestASTEngineBehavioralTierFull` over `internal/backend/testdata/plain_full.goal` via `corpus.RunCompile` (temp-module `go build` + `go vet`). Fixture exercises switch, struct type + composite, interface type, map literal + range, slice, defer, multi-return, const/var | PASS |
| Backend never crashes; goal-specific construct yields descriptive unsupported error | Unchanged `default` fail arms in emit.go `stmt`/`expr`/`decl`/`spec` and the preserved `struct implements` guard (now in `structType`); no goal node added | PASS (by construction) |
| `go build ./...`, `go vet ./...`, `go test ./... -count=1` green | Run at verify; no failures | PASS |

## Findings
- No CRITICAL or MAJOR. One MINOR (below).

### MINOR — Discovered & fixed a latent struct/interface bug
The US-026 `fieldList` joined struct fields / interface methods with commas,
which is invalid Go (`struct { A int, B int }`). It was never exercised because
the US-026 fixture declared no struct/interface type. US-032 replaced those two
cases with newline-separating `structType`/`interfaceType` emitters (interface
methods now render as `Name(params) results`, not `Name func(...)`). Covered by
the new behavioral fixture. This was necessary to satisfy "emit the Go subset".

## Assumptions
- Func-literal IIFEs and trailing variadic call spread are NOT in goal's parsed
  Go subset (the parser rejects them), so they are excluded from the fixture and
  out of scope.
- A single full-subset fixture is a sufficient AC-2 witness.
